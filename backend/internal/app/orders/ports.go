package orders

import (
	"context"
	"time"

	appmenu "coffee-pos/backend/internal/app/menu"
	domainorders "coffee-pos/backend/internal/domain/orders"
)

type MenuReader interface {
	GetCashierMenu(ctx context.Context) (appmenu.CashierMenu, error)
}

type Repository interface {
	CreatePaidOrder(ctx context.Context, input PersistPaidOrderInput) (PaidOrderDetail, CreatePaidOrderResult, error)
	CancelPaidOrder(ctx context.Context, input PersistCancelPaidOrderInput) (PaidOrderDetail, CancelPaidOrderResult, error)
}

type Clock interface {
	Now() time.Time
}

type PersistPaidOrderInput struct {
	ClientRequestID string
	RequestHash     []byte
	BusinessDate    domainorders.BusinessDate
	PaidAt          time.Time
	Draft           domainorders.PaidOrderDraft
	Lines           []PersistPaidOrderLineInput
}

type PersistPaidOrderLineInput struct {
	MenuItemID int64
	Modifiers  []PersistPaidOrderModifierInput
}

type PersistPaidOrderModifierInput struct {
	ModifierGroupID  int64
	ModifierOptionID int64
}

type PersistCancelPaidOrderInput struct {
	OrderID      string
	BusinessDate domainorders.BusinessDate
	CancelledAt  time.Time
}
