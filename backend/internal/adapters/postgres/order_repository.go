package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	apporders "coffee-pos/backend/internal/app/orders"
	domainorders "coffee-pos/backend/internal/domain/orders"

	"github.com/lib/pq"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return OrderRepository{db: db}
}

func (repo OrderRepository) CreatePaidOrder(ctx context.Context, input apporders.PersistPaidOrderInput) (apporders.PaidOrderDetail, apporders.CreatePaidOrderResult, error) {
	if existing, hash, ok, err := repo.getIdempotency(ctx, input.ClientRequestID); err != nil {
		return apporders.PaidOrderDetail{}, "", err
	} else if ok {
		if !bytes.Equal(hash, input.RequestHash) {
			return apporders.PaidOrderDetail{}, apporders.CreatePaidOrderIdempotencyConflict, nil
		}
		detail, err := repo.GetPaidOrder(ctx, existing)
		return detail, apporders.CreatePaidOrderExisting, err
	}

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return apporders.PaidOrderDetail{}, "", fmt.Errorf("begin order transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	queueNumber, err := allocateDailyQueueNumber(ctx, tx, input.BusinessDate.String())
	if err != nil {
		return apporders.PaidOrderDetail{}, "", err
	}
	orderID, err := insertOrder(ctx, tx, input, queueNumber)
	if err != nil {
		if isUniqueViolationOn(err, "orders_client_request_id_key") {
			_ = tx.Rollback()
			committed = true
			existing, hash, ok, getErr := repo.getIdempotency(ctx, input.ClientRequestID)
			if getErr != nil {
				return apporders.PaidOrderDetail{}, "", getErr
			}
			if ok && bytes.Equal(hash, input.RequestHash) {
				detail, detailErr := repo.GetPaidOrder(ctx, existing)
				return detail, apporders.CreatePaidOrderExisting, detailErr
			}
			return apporders.PaidOrderDetail{}, apporders.CreatePaidOrderIdempotencyConflict, nil
		}
		return apporders.PaidOrderDetail{}, "", err
	}
	for index, line := range input.Draft.Lines {
		lineID, err := insertOrderLine(ctx, tx, orderID, input.Lines[index].MenuItemID, line)
		if err != nil {
			return apporders.PaidOrderDetail{}, "", err
		}
		for modifierIndex, modifier := range line.Modifiers {
			refs := input.Lines[index].Modifiers[modifierIndex]
			if err := insertOrderLineModifier(ctx, tx, lineID, refs, modifier); err != nil {
				return apporders.PaidOrderDetail{}, "", err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return apporders.PaidOrderDetail{}, "", fmt.Errorf("commit order transaction: %w", err)
	}
	committed = true

	detail, err := repo.GetPaidOrder(ctx, orderID)
	return detail, apporders.CreatePaidOrderCreated, err
}

func (repo OrderRepository) GetPaidOrder(ctx context.Context, orderID int64) (apporders.PaidOrderDetail, error) {
	rows, err := repo.db.QueryContext(ctx, paidOrderDetailSQL, orderID)
	if err != nil {
		return apporders.PaidOrderDetail{}, fmt.Errorf("query paid order detail: %w", err)
	}
	defer rows.Close()

	var detail apporders.PaidOrderDetail
	lineIndexes := map[int64]int{}
	found := false
	for rows.Next() {
		var row paidOrderDetailRow
		if err := rows.Scan(
			&row.orderID,
			&row.businessDate,
			&row.queueNumber,
			&row.status,
			&row.paymentMethod,
			&row.paidAt,
			&row.cancelledAt,
			&row.note,
			&row.totalRp,
			&row.orderLineID,
			&row.menuItemSlug,
			&row.menuItemName,
			&row.unitPriceRp,
			&row.quantity,
			&row.lineTotalRp,
			&row.lineDisplayOrder,
			&row.groupSlug,
			&row.groupName,
			&row.optionSlug,
			&row.optionName,
			&row.priceDeltaRp,
			&row.modifierDisplayOrder,
		); err != nil {
			return apporders.PaidOrderDetail{}, fmt.Errorf("scan paid order detail: %w", err)
		}
		if !found {
			found = true
			detail = apporders.PaidOrderDetail{
				OrderID:       strconv.FormatInt(row.orderID, 10),
				QueueNumber:   int(row.queueNumber),
				BusinessDate:  row.businessDate.Format("2006-01-02"),
				Status:        row.status,
				PaymentMethod: row.paymentMethod,
				PaidAt:        row.paidAt,
				CancelledAt:   nullableTime(row.cancelledAt),
				Note:          nullableString(row.note),
				TotalRp:       row.totalRp,
			}
		}
		lineIndex, ok := lineIndexes[row.orderLineID]
		if !ok {
			lineIndex = len(detail.Lines)
			lineIndexes[row.orderLineID] = lineIndex
			detail.Lines = append(detail.Lines, apporders.PaidOrderLineDetail{
				MenuItemSlug: row.menuItemSlug,
				MenuItemName: row.menuItemName,
				UnitPriceRp:  row.unitPriceRp,
				Quantity:     int(row.quantity),
				LineTotalRp:  row.lineTotalRp,
			})
		}
		if row.groupSlug.Valid {
			detail.Lines[lineIndex].Modifiers = append(detail.Lines[lineIndex].Modifiers, apporders.PaidOrderModifierDetail{
				GroupSlug:    row.groupSlug.String,
				GroupName:    row.groupName.String,
				OptionSlug:   row.optionSlug.String,
				OptionName:   row.optionName.String,
				PriceDeltaRp: row.priceDeltaRp.Int64,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return apporders.PaidOrderDetail{}, fmt.Errorf("iterate paid order detail: %w", err)
	}
	if !found {
		return apporders.PaidOrderDetail{}, apporders.ErrOrderNotFound
	}
	return detail, nil
}

func (repo OrderRepository) getIdempotency(ctx context.Context, clientRequestID string) (int64, []byte, bool, error) {
	var orderID int64
	var hash []byte
	err := repo.db.QueryRowContext(ctx, `
		select id, request_hash
		from orders
		where client_request_id = $1
	`, clientRequestID).Scan(&orderID, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, fmt.Errorf("get order idempotency: %w", err)
	}
	return orderID, hash, true, nil
}

func allocateDailyQueueNumber(ctx context.Context, tx *sql.Tx, businessDate string) (int, error) {
	var queueNumber int
	if err := tx.QueryRowContext(ctx, `
		insert into daily_queue_counters (business_date, last_queue_number, updated_at)
		values ($1, 1, now())
		on conflict (business_date) do update set
			last_queue_number = daily_queue_counters.last_queue_number + 1,
			updated_at = now()
		returning last_queue_number
	`, businessDate).Scan(&queueNumber); err != nil {
		return 0, fmt.Errorf("allocate queue number: %w", err)
	}
	return queueNumber, nil
}

func insertOrder(ctx context.Context, tx *sql.Tx, input apporders.PersistPaidOrderInput, queueNumber int) (int64, error) {
	var orderID int64
	if err := tx.QueryRowContext(ctx, `
		insert into orders (
			business_date,
			queue_number,
			status,
			payment_method,
			paid_at,
			cancelled_at,
			note,
			total_rp,
			client_request_id,
			request_hash,
			updated_at
		)
		values ($1, $2, 'paid', $3, $4, null, $5, $6, $7, $8, now())
		returning id
	`, input.BusinessDate.String(), queueNumber, string(input.Draft.PaymentMethod), input.PaidAt, input.Draft.Note, input.Draft.TotalRp, input.ClientRequestID, input.RequestHash).Scan(&orderID); err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}
	return orderID, nil
}

func (repo OrderRepository) CancelPaidOrder(ctx context.Context, input apporders.PersistCancelPaidOrderInput) (apporders.PaidOrderDetail, apporders.CancelPaidOrderResult, error) {
	orderID, err := strconv.ParseInt(input.OrderID, 10, 64)
	if err != nil || orderID <= 0 {
		return apporders.PaidOrderDetail{}, "", apporders.ErrInvalidOrderID
	}
	result, err := repo.db.ExecContext(ctx, `
		update orders
		set status = 'cancelled',
			cancelled_at = $2,
			updated_at = now()
		where id = $1
			and business_date = $3
			and status = 'paid'
	`, orderID, input.CancelledAt, input.BusinessDate.String())
	if err != nil {
		return apporders.PaidOrderDetail{}, "", fmt.Errorf("cancel paid order: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apporders.PaidOrderDetail{}, "", fmt.Errorf("read cancellation rows affected: %w", err)
	}
	if rowsAffected == 0 {
		var exists bool
		if err := repo.db.QueryRowContext(ctx, `select exists(select 1 from orders where id = $1)`, orderID).Scan(&exists); err != nil {
			return apporders.PaidOrderDetail{}, "", fmt.Errorf("check cancellable order: %w", err)
		}
		if !exists {
			return apporders.PaidOrderDetail{}, apporders.CancelPaidOrderNotFound, nil
		}
		return apporders.PaidOrderDetail{}, apporders.CancelPaidOrderNotCancellable, nil
	}
	detail, err := repo.GetPaidOrder(ctx, orderID)
	return detail, apporders.CancelPaidOrderCancelled, err
}

func insertOrderLine(ctx context.Context, tx *sql.Tx, orderID int64, menuItemID int64, line domainorders.OrderLine) (int64, error) {
	var orderLineID int64
	if err := tx.QueryRowContext(ctx, `
		insert into order_lines (
			order_id,
			menu_item_id,
			menu_item_slug,
			menu_item_name,
			unit_price_rp,
			quantity,
			line_total_rp,
			display_order
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		returning id
	`, orderID, menuItemID, line.MenuItemSlug, line.MenuItemName, line.UnitPriceRp, line.Quantity, line.LineTotalRp, line.DisplayOrder).Scan(&orderLineID); err != nil {
		return 0, fmt.Errorf("insert order line: %w", err)
	}
	return orderLineID, nil
}

func insertOrderLineModifier(ctx context.Context, tx *sql.Tx, orderLineID int64, refs apporders.PersistPaidOrderModifierInput, modifier domainorders.Modifier) error {
	if _, err := tx.ExecContext(ctx, `
		insert into order_line_modifiers (
			order_line_id,
			modifier_group_id,
			modifier_option_id,
			group_slug,
			group_name,
			option_slug,
			option_name,
			price_delta_rp,
			display_order
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, orderLineID, refs.ModifierGroupID, refs.ModifierOptionID, modifier.GroupSlug, modifier.GroupName, modifier.OptionSlug, modifier.OptionName, modifier.PriceDeltaRp, modifier.DisplayOrder); err != nil {
		return fmt.Errorf("insert order line modifier: %w", err)
	}
	return nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func isUniqueViolationOn(err error, constraint string) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == constraint
}

const paidOrderDetailSQL = `
	select
		o.id as order_id,
		o.business_date,
		o.queue_number,
		o.status,
		o.payment_method,
		o.paid_at,
		o.cancelled_at,
		o.note,
		o.total_rp,
		ol.id as order_line_id,
		ol.menu_item_slug,
		ol.menu_item_name,
		ol.unit_price_rp,
		ol.quantity,
		ol.line_total_rp,
		ol.display_order as line_display_order,
		olm.group_slug,
		olm.group_name,
		olm.option_slug,
		olm.option_name,
		olm.price_delta_rp,
		olm.display_order as modifier_display_order
	from orders o
	join order_lines ol on ol.order_id = o.id
	left join order_line_modifiers olm on olm.order_line_id = ol.id
	where o.id = $1
	order by ol.display_order, ol.id, olm.display_order, olm.id
`

type paidOrderDetailRow struct {
	orderID              int64
	businessDate         time.Time
	queueNumber          int32
	status               string
	paymentMethod        string
	paidAt               time.Time
	cancelledAt          sql.NullTime
	note                 sql.NullString
	totalRp              int64
	orderLineID          int64
	menuItemSlug         string
	menuItemName         string
	unitPriceRp          int64
	quantity             int32
	lineTotalRp          int64
	lineDisplayOrder     int32
	groupSlug            sql.NullString
	groupName            sql.NullString
	optionSlug           sql.NullString
	optionName           sql.NullString
	priceDeltaRp         sql.NullInt64
	modifierDisplayOrder sql.NullInt32
}
