package orders

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrEmptyOrder             = errors.New("order must contain at least one line")
	ErrInvalidPaymentMethod   = errors.New("invalid payment method")
	ErrInvalidQuantity        = errors.New("quantity must be from 1 through 99")
	ErrInvalidLine            = errors.New("invalid order line")
	ErrInvalidPrice           = errors.New("prices must be non-negative")
	ErrDuplicateModifierGroup = errors.New("duplicate modifier group")
	ErrInvalidNote            = errors.New("note must be at most 500 characters")
	ErrTotalOverflow          = errors.New("order total overflow")
	ErrInvalidOrderID         = errors.New("order id must be positive base-10")
	ErrInvalidPaidOrder       = errors.New("invalid paid order")
	ErrOrderNotCancellable    = errors.New("order is not cancellable")
)

type PaymentMethod string

const (
	PaymentMethodCash PaymentMethod = "cash"
	PaymentMethodQRIS PaymentMethod = "qris"
)

type OrderStatus string

const (
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type BusinessDate struct {
	Year  int
	Month time.Month
	Day   int
}

func BusinessDateFromTime(now time.Time, location *time.Location) BusinessDate {
	if location == nil {
		location = time.UTC
	}
	local := now.In(location)
	return BusinessDate{Year: local.Year(), Month: local.Month(), Day: local.Day()}
}

func (date BusinessDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", date.Year, int(date.Month), date.Day)
}

type PaidOrderDraftInput struct {
	PaymentMethod string
	Note          *string
	Lines         []OrderLineInput
}

type OrderLineInput struct {
	MenuItemSlug string
	MenuItemName string
	UnitPriceRp  int64
	Quantity     int
	Modifiers    []ModifierInput
}

type ModifierInput struct {
	GroupSlug    string
	GroupName    string
	OptionSlug   string
	OptionName   string
	PriceDeltaRp int64
}

type PaidOrderDraft struct {
	PaymentMethod PaymentMethod
	Note          *string
	Lines         []OrderLine
	TotalRp       int64
}

type PaidOrderInput struct {
	ID           string
	BusinessDate BusinessDate
	QueueNumber  int
	PaidAt       time.Time
	Draft        PaidOrderDraft
}

type PaidOrder struct {
	ID            string
	BusinessDate  BusinessDate
	QueueNumber   int
	Status        OrderStatus
	PaymentMethod PaymentMethod
	PaidAt        time.Time
	CancelledAt   *time.Time
	Note          *string
	Lines         []OrderLine
	TotalRp       int64
}

type OrderLine struct {
	MenuItemSlug string
	MenuItemName string
	UnitPriceRp  int64
	Quantity     int
	Modifiers    []Modifier
	LineTotalRp  int64
	DisplayOrder int
}

type Modifier struct {
	GroupSlug    string
	GroupName    string
	OptionSlug   string
	OptionName   string
	PriceDeltaRp int64
	DisplayOrder int
}

func NewPaidOrderDraft(input PaidOrderDraftInput) (PaidOrderDraft, error) {
	paymentMethod, err := ParsePaymentMethod(input.PaymentMethod)
	if err != nil {
		return PaidOrderDraft{}, err
	}
	if input.Note != nil && len(*input.Note) > 500 {
		return PaidOrderDraft{}, ErrInvalidNote
	}
	if len(input.Lines) == 0 {
		return PaidOrderDraft{}, ErrEmptyOrder
	}

	lines := make([]OrderLine, 0, len(input.Lines))
	var total int64
	for index, lineInput := range input.Lines {
		line, err := newOrderLine(lineInput, index)
		if err != nil {
			return PaidOrderDraft{}, err
		}
		nextTotal, ok := checkedAdd(total, line.LineTotalRp)
		if !ok {
			return PaidOrderDraft{}, ErrTotalOverflow
		}
		total = nextTotal
		lines = append(lines, line)
	}

	return PaidOrderDraft{
		PaymentMethod: paymentMethod,
		Note:          input.Note,
		Lines:         lines,
		TotalRp:       total,
	}, nil
}

func ParsePaymentMethod(value string) (PaymentMethod, error) {
	switch PaymentMethod(value) {
	case PaymentMethodCash:
		return PaymentMethodCash, nil
	case PaymentMethodQRIS:
		return PaymentMethodQRIS, nil
	default:
		return "", ErrInvalidPaymentMethod
	}
}

func NewPaidOrder(input PaidOrderInput) (PaidOrder, error) {
	if !validOrderID(input.ID) {
		return PaidOrder{}, ErrInvalidOrderID
	}
	if input.QueueNumber <= 0 || len(input.Draft.Lines) == 0 || input.Draft.TotalRp < 0 {
		return PaidOrder{}, ErrInvalidPaidOrder
	}
	return PaidOrder{
		ID:            input.ID,
		BusinessDate:  input.BusinessDate,
		QueueNumber:   input.QueueNumber,
		Status:        OrderStatusPaid,
		PaymentMethod: input.Draft.PaymentMethod,
		PaidAt:        input.PaidAt,
		Note:          input.Draft.Note,
		Lines:         cloneLines(input.Draft.Lines),
		TotalRp:       input.Draft.TotalRp,
	}, nil
}

func (order PaidOrder) Cancel(cancelledAt time.Time) (PaidOrder, error) {
	if order.Status != OrderStatusPaid || order.CancelledAt != nil {
		return PaidOrder{}, ErrOrderNotCancellable
	}
	cancelled := order
	cancelled.Status = OrderStatusCancelled
	cancelled.CancelledAt = &cancelledAt
	cancelled.Lines = cloneLines(order.Lines)
	return cancelled, nil
}

func newOrderLine(input OrderLineInput, displayOrder int) (OrderLine, error) {
	if strings.TrimSpace(input.MenuItemSlug) == "" || strings.TrimSpace(input.MenuItemName) == "" {
		return OrderLine{}, ErrInvalidLine
	}
	if input.Quantity < 1 || input.Quantity > 99 {
		return OrderLine{}, ErrInvalidQuantity
	}
	if input.UnitPriceRp < 0 {
		return OrderLine{}, ErrInvalidPrice
	}

	modifiers := make([]Modifier, 0, len(input.Modifiers))
	groupSlugs := map[string]struct{}{}
	var modifierTotal int64
	for index, modifierInput := range input.Modifiers {
		modifier, err := newModifier(modifierInput, index)
		if err != nil {
			return OrderLine{}, err
		}
		if _, exists := groupSlugs[modifier.GroupSlug]; exists {
			return OrderLine{}, ErrDuplicateModifierGroup
		}
		groupSlugs[modifier.GroupSlug] = struct{}{}
		nextTotal, ok := checkedAdd(modifierTotal, modifier.PriceDeltaRp)
		if !ok {
			return OrderLine{}, ErrTotalOverflow
		}
		modifierTotal = nextTotal
		modifiers = append(modifiers, modifier)
	}

	unitWithModifiers, ok := checkedAdd(input.UnitPriceRp, modifierTotal)
	if !ok {
		return OrderLine{}, ErrTotalOverflow
	}
	lineTotal, ok := checkedMul(unitWithModifiers, int64(input.Quantity))
	if !ok {
		return OrderLine{}, ErrTotalOverflow
	}

	return OrderLine{
		MenuItemSlug: input.MenuItemSlug,
		MenuItemName: input.MenuItemName,
		UnitPriceRp:  input.UnitPriceRp,
		Quantity:     input.Quantity,
		Modifiers:    modifiers,
		LineTotalRp:  lineTotal,
		DisplayOrder: displayOrder,
	}, nil
}

func newModifier(input ModifierInput, displayOrder int) (Modifier, error) {
	if strings.TrimSpace(input.GroupSlug) == "" || strings.TrimSpace(input.GroupName) == "" || strings.TrimSpace(input.OptionSlug) == "" || strings.TrimSpace(input.OptionName) == "" {
		return Modifier{}, ErrInvalidLine
	}
	if input.PriceDeltaRp < 0 {
		return Modifier{}, ErrInvalidPrice
	}
	return Modifier{
		GroupSlug:    input.GroupSlug,
		GroupName:    input.GroupName,
		OptionSlug:   input.OptionSlug,
		OptionName:   input.OptionName,
		PriceDeltaRp: input.PriceDeltaRp,
		DisplayOrder: displayOrder,
	}, nil
}

func validOrderID(value string) bool {
	if value == "" || value[0] == '0' {
		return false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return err == nil && parsed > 0
}

func checkedAdd(left int64, right int64) (int64, bool) {
	if right > 0 && left > math.MaxInt64-right {
		return 0, false
	}
	return left + right, true
}

func checkedMul(left int64, right int64) (int64, bool) {
	if left == 0 || right == 0 {
		return 0, true
	}
	if left > math.MaxInt64/right {
		return 0, false
	}
	return left * right, true
}

func cloneLines(lines []OrderLine) []OrderLine {
	cloned := make([]OrderLine, len(lines))
	for i, line := range lines {
		cloned[i] = line
		cloned[i].Modifiers = append([]Modifier(nil), line.Modifiers...)
	}
	return cloned
}
