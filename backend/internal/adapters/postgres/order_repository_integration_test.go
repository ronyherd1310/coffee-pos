//go:build integration

package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	apporders "coffee-pos/backend/internal/app/orders"
	domainmenu "coffee-pos/backend/internal/domain/menu"
	domainorders "coffee-pos/backend/internal/domain/orders"
)

func TestOrderRepositoryCreatesPaidOrdersWithSequentialQueuesAndSnapshots(t *testing.T) {
	ctx := context.Background()
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)
	seedApprovedMenu(t, ctx, db)

	repository := NewOrderRepository(db)
	input := newPersistPaidOrderInput(t, ctx, db, "11111111-1111-4111-8111-111111111111", []byte("first-request"))

	detail, result, err := repository.CreatePaidOrder(ctx, input)
	if err != nil {
		t.Fatalf("CreatePaidOrder returned error: %v", err)
	}
	if result != apporders.CreatePaidOrderCreated {
		t.Fatalf("result = %q, want %q", result, apporders.CreatePaidOrderCreated)
	}
	if detail.QueueNumber != 1 {
		t.Fatalf("queue number = %d, want 1", detail.QueueNumber)
	}
	assertPersistedOrderSnapshot(t, detail)

	secondInput := newPersistPaidOrderInput(t, ctx, db, "22222222-2222-4222-8222-222222222222", []byte("second-request"))
	secondDetail, _, err := repository.CreatePaidOrder(ctx, secondInput)
	if err != nil {
		t.Fatalf("CreatePaidOrder second returned error: %v", err)
	}
	if secondDetail.QueueNumber != 2 {
		t.Fatalf("second queue number = %d, want 2", secondDetail.QueueNumber)
	}
}

func TestOrderRepositoryAllocatesConcurrentQueuesOnce(t *testing.T) {
	ctx := context.Background()
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)
	seedApprovedMenu(t, ctx, db)

	repository := NewOrderRepository(db)
	const orderCount = 5
	inputs := make([]apporders.PersistPaidOrderInput, orderCount)
	for index := range orderCount {
		clientRequestID := fmt.Sprintf("33333333-3333-4333-8333-%012d", index+1)
		inputs[index] = newPersistPaidOrderInput(t, ctx, db, clientRequestID, []byte(fmt.Sprintf("request-%d", index)))
	}
	queueNumbers := make([]int, orderCount)
	errs := make([]error, orderCount)
	var wg sync.WaitGroup
	for index := range orderCount {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			detail, _, err := repository.CreatePaidOrder(ctx, inputs[index])
			if err != nil {
				errs[index] = err
				return
			}
			queueNumbers[index] = detail.QueueNumber
		}(index)
	}
	wg.Wait()
	for index, err := range errs {
		if err != nil {
			t.Fatalf("order %d returned error: %v", index, err)
		}
	}

	sort.Ints(queueNumbers)
	for index, queueNumber := range queueNumbers {
		if queueNumber != index+1 {
			t.Fatalf("queue numbers = %v, want 1 through %d", queueNumbers, orderCount)
		}
	}
}

func TestOrderRepositoryHandlesIdempotencyRetriesAndConflicts(t *testing.T) {
	ctx := context.Background()
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)
	seedApprovedMenu(t, ctx, db)

	repository := NewOrderRepository(db)
	input := newPersistPaidOrderInput(t, ctx, db, "44444444-4444-4444-8444-444444444444", []byte("same-request"))
	created, result, err := repository.CreatePaidOrder(ctx, input)
	if err != nil {
		t.Fatalf("CreatePaidOrder returned error: %v", err)
	}
	if result != apporders.CreatePaidOrderCreated {
		t.Fatalf("result = %q, want created", result)
	}

	existing, result, err := repository.CreatePaidOrder(ctx, input)
	if err != nil {
		t.Fatalf("CreatePaidOrder retry returned error: %v", err)
	}
	if result != apporders.CreatePaidOrderExisting {
		t.Fatalf("retry result = %q, want existing", result)
	}
	if existing.OrderID != created.OrderID || existing.QueueNumber != created.QueueNumber {
		t.Fatalf("retry returned %+v, want original %+v", existing, created)
	}

	conflicting := input
	conflicting.RequestHash = []byte("different-request")
	_, result, err = repository.CreatePaidOrder(ctx, conflicting)
	if err != nil {
		t.Fatalf("CreatePaidOrder conflicting retry returned unexpected error: %v", err)
	}
	if result != apporders.CreatePaidOrderIdempotencyConflict {
		t.Fatalf("conflict result = %q, want idempotency_conflict", result)
	}
}

func TestOrderRepositoryDoesNotTreatQueueCollisionAsIdempotencyConflict(t *testing.T) {
	ctx := context.Background()
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)
	seedApprovedMenu(t, ctx, db)

	if _, err := db.ExecContext(ctx, `
		insert into orders (
			business_date,
			queue_number,
			status,
			payment_method,
			paid_at,
			total_rp,
			client_request_id,
			request_hash,
			updated_at
		)
		values ('2026-06-30', 1, 'paid', 'cash', '2026-06-30T03:00:00Z', 18000, '55555555-5555-4555-8555-555555555555', '\x01', now())
	`); err != nil {
		t.Fatalf("insert colliding order: %v", err)
	}

	repository := NewOrderRepository(db)
	input := newPersistPaidOrderInput(t, ctx, db, "66666666-6666-4666-8666-666666666666", []byte("queue-collision"))
	_, result, err := repository.CreatePaidOrder(ctx, input)

	if err == nil {
		t.Fatal("CreatePaidOrder returned nil error for queue collision")
	}
	if result == apporders.CreatePaidOrderIdempotencyConflict {
		t.Fatalf("result = %q, want internal error with no idempotency conflict", result)
	}
}

func TestOrderRepositoryCancelsSameDayPaidOrderOnly(t *testing.T) {
	ctx := context.Background()
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)
	seedApprovedMenu(t, ctx, db)

	repository := NewOrderRepository(db)
	input := newPersistPaidOrderInput(t, ctx, db, "77777777-7777-4777-8777-777777777777", []byte("cancel-same-day"))
	created, _, err := repository.CreatePaidOrder(ctx, input)
	if err != nil {
		t.Fatalf("CreatePaidOrder returned error: %v", err)
	}

	cancelled, result, err := repository.CancelPaidOrder(ctx, apporders.PersistCancelPaidOrderInput{
		OrderID:      created.OrderID,
		BusinessDate: domainorders.BusinessDate{Year: 2026, Month: time.June, Day: 30},
		CancelledAt:  time.Date(2026, 6, 30, 4, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CancelPaidOrder returned error: %v", err)
	}
	if result != apporders.CancelPaidOrderCancelled || cancelled.Status != "cancelled" || cancelled.CancelledAt == nil {
		t.Fatalf("cancel result = %q detail = %+v, want cancelled detail", result, cancelled)
	}

	previousDayInput := newPersistPaidOrderInput(t, ctx, db, "88888888-8888-4888-8888-888888888888", []byte("cancel-previous-day"))
	previousDayInput.BusinessDate = domainorders.BusinessDate{Year: 2026, Month: time.June, Day: 29}
	previousDay, _, err := repository.CreatePaidOrder(ctx, previousDayInput)
	if err != nil {
		t.Fatalf("CreatePaidOrder previous day returned error: %v", err)
	}

	_, result, err = repository.CancelPaidOrder(ctx, apporders.PersistCancelPaidOrderInput{
		OrderID:      previousDay.OrderID,
		BusinessDate: domainorders.BusinessDate{Year: 2026, Month: time.June, Day: 30},
		CancelledAt:  time.Date(2026, 6, 30, 4, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CancelPaidOrder previous day returned unexpected error: %v", err)
	}
	if result != apporders.CancelPaidOrderNotCancellable {
		t.Fatalf("previous-day cancel result = %q, want not_cancellable", result)
	}
}

func seedApprovedMenu(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	repository := NewMenuRepository(db)
	if err := repository.SeedMenu(ctx, domainmenu.ApprovedSeed()); err != nil {
		t.Fatalf("seed menu: %v", err)
	}
}

func newPersistPaidOrderInput(t *testing.T, ctx context.Context, db *sql.DB, clientRequestID string, requestHash []byte) apporders.PersistPaidOrderInput {
	t.Helper()

	menuRepository := NewMenuRepository(db)
	menu, err := menuRepository.GetCashierMenu(ctx)
	if err != nil {
		t.Fatalf("get cashier menu: %v", err)
	}
	item := menu.Categories[0].Items[0]
	if len(item.ModifierGroups) < 2 {
		t.Fatalf("expected at least two modifier groups, got %+v", item.ModifierGroups)
	}
	firstGroup := item.ModifierGroups[0]
	secondGroup := item.ModifierGroups[1]
	firstOption := firstGroup.Options[0]
	secondOption := secondGroup.Options[0]

	draft, err := domainorders.NewPaidOrderDraft(domainorders.PaidOrderDraftInput{
		PaymentMethod: "cash",
		Lines: []domainorders.OrderLineInput{{
			MenuItemSlug: item.Slug,
			MenuItemName: item.Name,
			UnitPriceRp:  item.PriceRp,
			Quantity:     1,
			Modifiers: []domainorders.ModifierInput{
				{GroupSlug: firstGroup.Slug, GroupName: firstGroup.Name, OptionSlug: firstOption.Slug, OptionName: firstOption.Name, PriceDeltaRp: firstOption.PriceDeltaRp},
				{GroupSlug: secondGroup.Slug, GroupName: secondGroup.Name, OptionSlug: secondOption.Slug, OptionName: secondOption.Name, PriceDeltaRp: secondOption.PriceDeltaRp},
			},
		}},
	})
	if err != nil {
		t.Fatalf("create paid order draft: %v", err)
	}

	return apporders.PersistPaidOrderInput{
		ClientRequestID: clientRequestID,
		RequestHash:     bytes.Clone(requestHash),
		BusinessDate:    domainorders.BusinessDate{Year: 2026, Month: time.June, Day: 30},
		PaidAt:          time.Date(2026, 6, 30, 3, 0, 0, 0, time.UTC),
		Draft:           draft,
		Lines: []apporders.PersistPaidOrderLineInput{{
			MenuItemID: item.ID,
			Modifiers: []apporders.PersistPaidOrderModifierInput{
				{ModifierGroupID: firstGroup.ID, ModifierOptionID: firstOption.ID},
				{ModifierGroupID: secondGroup.ID, ModifierOptionID: secondOption.ID},
			},
		}},
	}
}

func assertPersistedOrderSnapshot(t *testing.T, detail apporders.PaidOrderDetail) {
	t.Helper()

	if detail.BusinessDate != "2026-06-30" || detail.Status != "paid" || detail.PaymentMethod != "cash" {
		t.Fatalf("unexpected detail fields: %+v", detail)
	}
	if len(detail.Lines) != 1 {
		t.Fatalf("lines = %+v, want one line", detail.Lines)
	}
	line := detail.Lines[0]
	if line.MenuItemSlug == "" || line.MenuItemName == "" || line.UnitPriceRp <= 0 || line.LineTotalRp <= 0 {
		t.Fatalf("line snapshot was not persisted: %+v", line)
	}
	if len(line.Modifiers) != 2 {
		t.Fatalf("modifiers = %+v, want two modifier snapshots", line.Modifiers)
	}
	if line.Modifiers[0].GroupSlug == "" || line.Modifiers[0].OptionSlug == "" {
		t.Fatalf("modifier snapshot was not persisted: %+v", line.Modifiers[0])
	}
}
