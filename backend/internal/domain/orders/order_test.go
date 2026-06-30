package orders

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestNewPaidOrderDraftCalculatesTotals(t *testing.T) {
	note := strings.Repeat("a", 500)
	draft, err := NewPaidOrderDraft(PaidOrderDraftInput{
		PaymentMethod: "cash",
		Note:          &note,
		Lines: []OrderLineInput{
			{
				MenuItemSlug: "kopi-susu",
				MenuItemName: "Kopi Susu",
				UnitPriceRp:  18000,
				Quantity:     2,
				Modifiers: []ModifierInput{
					{GroupSlug: "temperature", GroupName: "Temperature", OptionSlug: "hot", OptionName: "Hot", PriceDeltaRp: 0},
					{GroupSlug: "sugar", GroupName: "Sugar", OptionSlug: "less", OptionName: "Less Sugar", PriceDeltaRp: 1000},
				},
			},
			{
				MenuItemSlug: "kopi-susu",
				MenuItemName: "Kopi Susu",
				UnitPriceRp:  18000,
				Quantity:     1,
				Modifiers: []ModifierInput{
					{GroupSlug: "temperature", GroupName: "Temperature", OptionSlug: "iced", OptionName: "Iced", PriceDeltaRp: 2000},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewPaidOrderDraft returned error: %v", err)
	}

	if draft.PaymentMethod != PaymentMethodCash {
		t.Fatalf("payment method = %q, want %q", draft.PaymentMethod, PaymentMethodCash)
	}
	if draft.TotalRp != 58000 {
		t.Fatalf("total = %d, want 58000", draft.TotalRp)
	}
	if got := draft.Lines[0].LineTotalRp; got != 38000 {
		t.Fatalf("first line total = %d, want 38000", got)
	}
	if got := draft.Lines[1].LineTotalRp; got != 20000 {
		t.Fatalf("second line total = %d, want 20000", got)
	}
}

func TestNewPaidOrderDraftRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input PaidOrderDraftInput
		want  error
	}{
		{name: "empty order", input: PaidOrderDraftInput{PaymentMethod: "cash"}, want: ErrEmptyOrder},
		{name: "invalid payment method", input: validDraftInput(func(input *PaidOrderDraftInput) { input.PaymentMethod = "card" }), want: ErrInvalidPaymentMethod},
		{name: "quantity zero", input: validDraftInput(func(input *PaidOrderDraftInput) { input.Lines[0].Quantity = 0 }), want: ErrInvalidQuantity},
		{name: "quantity one hundred", input: validDraftInput(func(input *PaidOrderDraftInput) { input.Lines[0].Quantity = 100 }), want: ErrInvalidQuantity},
		{name: "missing item name", input: validDraftInput(func(input *PaidOrderDraftInput) { input.Lines[0].MenuItemName = "" }), want: ErrInvalidLine},
		{name: "negative unit price", input: validDraftInput(func(input *PaidOrderDraftInput) { input.Lines[0].UnitPriceRp = -1 }), want: ErrInvalidPrice},
		{name: "negative modifier price", input: validDraftInput(func(input *PaidOrderDraftInput) { input.Lines[0].Modifiers[0].PriceDeltaRp = -1 }), want: ErrInvalidPrice},
		{name: "duplicate modifier group", input: validDraftInput(func(input *PaidOrderDraftInput) {
			input.Lines[0].Modifiers = append(input.Lines[0].Modifiers, ModifierInput{GroupSlug: "temperature", GroupName: "Temperature", OptionSlug: "iced", OptionName: "Iced"})
		}), want: ErrDuplicateModifierGroup},
		{name: "note too long", input: validDraftInput(func(input *PaidOrderDraftInput) {
			note := strings.Repeat("a", 501)
			input.Note = &note
		}), want: ErrInvalidNote},
		{name: "total overflow", input: validDraftInput(func(input *PaidOrderDraftInput) {
			input.Lines[0].UnitPriceRp = math.MaxInt64
			input.Lines[0].Quantity = 2
		}), want: ErrTotalOverflow},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewPaidOrderDraft(test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestBusinessDateFromTimeUsesLocation(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	instant := time.Date(2026, 6, 29, 17, 30, 0, 0, time.UTC)

	date := BusinessDateFromTime(instant, location)

	if got, want := date.String(), "2026-06-30"; got != want {
		t.Fatalf("business date = %q, want %q", got, want)
	}
}

func TestCancelPaidOrderPreservesOrderData(t *testing.T) {
	draft, err := NewPaidOrderDraft(validDraftInput(nil))
	if err != nil {
		t.Fatalf("NewPaidOrderDraft returned error: %v", err)
	}
	paidAt := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	paid, err := NewPaidOrder(PaidOrderInput{
		ID:           "12",
		BusinessDate: BusinessDate{Year: 2026, Month: 6, Day: 30},
		QueueNumber:  7,
		PaidAt:       paidAt,
		Draft:        draft,
	})
	if err != nil {
		t.Fatalf("NewPaidOrder returned error: %v", err)
	}
	cancelledAt := paidAt.Add(time.Hour)

	cancelled, err := paid.Cancel(cancelledAt)
	if err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}

	if cancelled.Status != OrderStatusCancelled {
		t.Fatalf("status = %q, want %q", cancelled.Status, OrderStatusCancelled)
	}
	if cancelled.CancelledAt == nil || !cancelled.CancelledAt.Equal(cancelledAt) {
		t.Fatalf("cancelledAt = %v, want %v", cancelled.CancelledAt, cancelledAt)
	}
	if cancelled.QueueNumber != paid.QueueNumber || cancelled.TotalRp != paid.TotalRp || len(cancelled.Lines) != len(paid.Lines) {
		t.Fatal("cancelled order did not preserve paid order data")
	}
}

func validDraftInput(mutator func(*PaidOrderDraftInput)) PaidOrderDraftInput {
	input := PaidOrderDraftInput{
		PaymentMethod: "qris",
		Lines: []OrderLineInput{{
			MenuItemSlug: "kopi-susu",
			MenuItemName: "Kopi Susu",
			UnitPriceRp:  18000,
			Quantity:     1,
			Modifiers: []ModifierInput{{
				GroupSlug:    "temperature",
				GroupName:    "Temperature",
				OptionSlug:   "hot",
				OptionName:   "Hot",
				PriceDeltaRp: 0,
			}},
		}},
	}
	if mutator != nil {
		mutator(&input)
	}
	return input
}
