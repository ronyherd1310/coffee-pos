package orders

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	appmenu "coffee-pos/backend/internal/app/menu"
	domainorders "coffee-pos/backend/internal/domain/orders"
)

var (
	ErrInvalidClientRequestID = errors.New("invalid client request id")
	ErrInvalidOrderID         = errors.New("invalid order id")
	ErrInvalidOrder           = errors.New("invalid order")
	ErrIdempotencyConflict    = errors.New("idempotency conflict")
	ErrOrderNotFound          = errors.New("order not found")
	ErrOrderNotCancellable    = errors.New("order not cancellable")
)

var canonicalUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Dependencies struct {
	MenuReader MenuReader
	Repository Repository
	Clock      Clock
	Location   *time.Location
}

type Service struct {
	menuReader MenuReader
	repository Repository
	clock      Clock
	location   *time.Location
}

func NewService(deps Dependencies) Service {
	return Service{
		menuReader: deps.MenuReader,
		repository: deps.Repository,
		clock:      deps.Clock,
		location:   deps.Location,
	}
}

type CreatePaidOrderInput struct {
	ClientRequestID string
	PaymentMethod   string
	Note            *string
	Lines           []CreatePaidOrderLineInput
}

type CreatePaidOrderLineInput struct {
	MenuItemSlug string
	Quantity     int
	Modifiers    []CreatePaidOrderModifierInput
}

type CreatePaidOrderModifierInput struct {
	GroupSlug  string
	OptionSlug string
}

type CreatePaidOrderResult string

const (
	CreatePaidOrderCreated             CreatePaidOrderResult = "created"
	CreatePaidOrderExisting            CreatePaidOrderResult = "existing"
	CreatePaidOrderIdempotencyConflict CreatePaidOrderResult = "idempotency_conflict"
)

type CancelPaidOrderInput struct {
	OrderID string
}

type CancelPaidOrderResult string

const (
	CancelPaidOrderCancelled      CancelPaidOrderResult = "cancelled"
	CancelPaidOrderNotFound       CancelPaidOrderResult = "not_found"
	CancelPaidOrderNotCancellable CancelPaidOrderResult = "not_cancellable"
)

type PaidOrderDetail struct {
	OrderID       string
	QueueNumber   int
	BusinessDate  string
	Status        string
	PaymentMethod string
	PaidAt        time.Time
	CancelledAt   *time.Time
	Note          *string
	TotalRp       int64
	Lines         []PaidOrderLineDetail
}

type PaidOrderLineDetail struct {
	MenuItemSlug string
	MenuItemName string
	UnitPriceRp  int64
	Quantity     int
	LineTotalRp  int64
	Modifiers    []PaidOrderModifierDetail
}

type PaidOrderModifierDetail struct {
	GroupSlug    string
	GroupName    string
	OptionSlug   string
	OptionName   string
	PriceDeltaRp int64
}

func (service Service) CreatePaidOrder(ctx context.Context, input CreatePaidOrderInput) (PaidOrderDetail, CreatePaidOrderResult, error) {
	if !canonicalUUIDPattern.MatchString(input.ClientRequestID) {
		return PaidOrderDetail{}, "", ErrInvalidClientRequestID
	}
	if service.menuReader == nil || service.repository == nil || service.clock == nil {
		return PaidOrderDetail{}, "", fmt.Errorf("create paid order: dependencies are required")
	}

	menu, err := service.menuReader.GetCashierMenu(ctx)
	if err != nil {
		return PaidOrderDetail{}, "", fmt.Errorf("read cashier menu: %w", err)
	}
	draftInput, lineRefs, err := resolveDraft(input, menu)
	if err != nil {
		return PaidOrderDetail{}, "", err
	}
	draft, err := domainorders.NewPaidOrderDraft(draftInput)
	if err != nil {
		return PaidOrderDetail{}, "", fmt.Errorf("%w: %v", ErrInvalidOrder, err)
	}

	now := service.clock.Now()
	location := service.location
	if location == nil {
		location = time.UTC
	}
	hash, err := requestHash(input)
	if err != nil {
		return PaidOrderDetail{}, "", err
	}
	detail, result, err := service.repository.CreatePaidOrder(ctx, PersistPaidOrderInput{
		ClientRequestID: input.ClientRequestID,
		RequestHash:     hash,
		BusinessDate:    domainorders.BusinessDateFromTime(now, location),
		PaidAt:          now,
		Draft:           draft,
		Lines:           lineRefs,
	})
	if err != nil {
		return PaidOrderDetail{}, result, err
	}
	if result == CreatePaidOrderIdempotencyConflict {
		return PaidOrderDetail{}, result, ErrIdempotencyConflict
	}
	return detail, result, nil
}

func (service Service) CancelPaidOrder(ctx context.Context, input CancelPaidOrderInput) (PaidOrderDetail, CancelPaidOrderResult, error) {
	if !validOrderID(input.OrderID) {
		return PaidOrderDetail{}, "", ErrInvalidOrderID
	}
	if service.repository == nil || service.clock == nil {
		return PaidOrderDetail{}, "", fmt.Errorf("cancel paid order: dependencies are required")
	}
	now := service.clock.Now()
	location := service.location
	if location == nil {
		location = time.UTC
	}
	detail, result, err := service.repository.CancelPaidOrder(ctx, PersistCancelPaidOrderInput{
		OrderID:      input.OrderID,
		BusinessDate: domainorders.BusinessDateFromTime(now, location),
		CancelledAt:  now,
	})
	if err != nil {
		return PaidOrderDetail{}, result, err
	}
	switch result {
	case CancelPaidOrderNotFound:
		return PaidOrderDetail{}, result, ErrOrderNotFound
	case CancelPaidOrderNotCancellable:
		return PaidOrderDetail{}, result, ErrOrderNotCancellable
	default:
		return detail, result, nil
	}
}

func resolveDraft(input CreatePaidOrderInput, menu appmenu.CashierMenu) (domainorders.PaidOrderDraftInput, []PersistPaidOrderLineInput, error) {
	if len(input.Lines) == 0 {
		return domainorders.PaidOrderDraftInput{}, nil, ErrInvalidOrder
	}
	items := map[string]appmenu.CashierMenuItem{}
	for _, category := range menu.Categories {
		for _, item := range category.Items {
			items[item.Slug] = item
		}
	}
	if len(items) == 0 {
		return domainorders.PaidOrderDraftInput{}, nil, ErrInvalidOrder
	}

	draft := domainorders.PaidOrderDraftInput{
		PaymentMethod: input.PaymentMethod,
		Note:          input.Note,
		Lines:         make([]domainorders.OrderLineInput, 0, len(input.Lines)),
	}
	lineRefs := make([]PersistPaidOrderLineInput, 0, len(input.Lines))
	for _, line := range input.Lines {
		item, ok := items[line.MenuItemSlug]
		if !ok {
			return domainorders.PaidOrderDraftInput{}, nil, ErrInvalidOrder
		}
		lineInput, lineRef, err := resolveLine(line, item)
		if err != nil {
			return domainorders.PaidOrderDraftInput{}, nil, err
		}
		draft.Lines = append(draft.Lines, lineInput)
		lineRefs = append(lineRefs, lineRef)
	}
	return draft, lineRefs, nil
}

func resolveLine(input CreatePaidOrderLineInput, item appmenu.CashierMenuItem) (domainorders.OrderLineInput, PersistPaidOrderLineInput, error) {
	groups := map[string]appmenu.CashierModifierGroup{}
	for _, group := range item.ModifierGroups {
		groups[group.Slug] = group
	}
	selectedGroups := map[string]struct{}{}
	line := domainorders.OrderLineInput{
		MenuItemSlug: item.Slug,
		MenuItemName: item.Name,
		UnitPriceRp:  item.PriceRp,
		Quantity:     input.Quantity,
		Modifiers:    make([]domainorders.ModifierInput, 0, len(input.Modifiers)),
	}
	lineRef := PersistPaidOrderLineInput{
		MenuItemID: item.ID,
		Modifiers:  make([]PersistPaidOrderModifierInput, 0, len(input.Modifiers)),
	}

	for _, modifier := range input.Modifiers {
		group, ok := groups[modifier.GroupSlug]
		if !ok {
			return domainorders.OrderLineInput{}, PersistPaidOrderLineInput{}, ErrInvalidOrder
		}
		if _, duplicate := selectedGroups[group.Slug]; duplicate {
			return domainorders.OrderLineInput{}, PersistPaidOrderLineInput{}, ErrInvalidOrder
		}
		selectedGroups[group.Slug] = struct{}{}
		option, ok := findOption(group, modifier.OptionSlug)
		if !ok {
			return domainorders.OrderLineInput{}, PersistPaidOrderLineInput{}, ErrInvalidOrder
		}
		line.Modifiers = append(line.Modifiers, domainorders.ModifierInput{
			GroupSlug:    group.Slug,
			GroupName:    group.Name,
			OptionSlug:   option.Slug,
			OptionName:   option.Name,
			PriceDeltaRp: option.PriceDeltaRp,
		})
		lineRef.Modifiers = append(lineRef.Modifiers, PersistPaidOrderModifierInput{
			ModifierGroupID:  group.ID,
			ModifierOptionID: option.ID,
		})
	}
	for _, group := range item.ModifierGroups {
		if group.Required {
			if _, ok := selectedGroups[group.Slug]; !ok {
				return domainorders.OrderLineInput{}, PersistPaidOrderLineInput{}, ErrInvalidOrder
			}
		}
	}
	return line, lineRef, nil
}

func findOption(group appmenu.CashierModifierGroup, slug string) (appmenu.CashierModifierOption, bool) {
	for _, option := range group.Options {
		if option.Slug == slug {
			return option, true
		}
	}
	return appmenu.CashierModifierOption{}, false
}

func requestHash(input CreatePaidOrderInput) ([]byte, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("hash create request: %w", err)
	}
	sum := sha256.Sum256(payload)
	return sum[:], nil
}

func validOrderID(value string) bool {
	if value == "" || value[0] == '0' {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
