package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	appmenu "coffee-pos/backend/internal/app/menu"
)

func TestServiceCreatePaidOrderResolvesMenuAndCalculatesTotals(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	repository := &fakeOrderRepository{}
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: repository,
		Clock:      fixedClock{now: time.Date(2026, 6, 29, 17, 30, 0, 0, time.UTC)},
		Location:   location,
	})

	detail, result, err := service.CreatePaidOrder(context.Background(), CreatePaidOrderInput{
		ClientRequestID: "11111111-1111-4111-8111-111111111111",
		PaymentMethod:   "cash",
		Lines: []CreatePaidOrderLineInput{{
			MenuItemSlug: "kopi-susu",
			Quantity:     2,
			Modifiers: []CreatePaidOrderModifierInput{
				{GroupSlug: "temperature", OptionSlug: "iced"},
				{GroupSlug: "sugar", OptionSlug: "less"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("CreatePaidOrder returned error: %v", err)
	}

	if result != CreatePaidOrderCreated {
		t.Fatalf("result = %q, want %q", result, CreatePaidOrderCreated)
	}
	if detail.BusinessDate != "2026-06-30" {
		t.Fatalf("business date = %q, want 2026-06-30", detail.BusinessDate)
	}
	if detail.TotalRp != 42000 {
		t.Fatalf("total = %d, want 42000", detail.TotalRp)
	}
	if repository.saved.Draft.Lines[0].MenuItemName != "Kopi Susu" {
		t.Fatalf("line item name = %q, want Kopi Susu", repository.saved.Draft.Lines[0].MenuItemName)
	}
}

func TestServiceCreatePaidOrderRejectsMissingRequiredModifier(t *testing.T) {
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: &fakeOrderRepository{},
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	_, _, err := service.CreatePaidOrder(context.Background(), CreatePaidOrderInput{
		ClientRequestID: "11111111-1111-4111-8111-111111111111",
		PaymentMethod:   "qris",
		Lines: []CreatePaidOrderLineInput{{
			MenuItemSlug: "kopi-susu",
			Quantity:     1,
			Modifiers:    []CreatePaidOrderModifierInput{{GroupSlug: "temperature", OptionSlug: "hot"}},
		}},
	})

	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidOrder)
	}
}

func TestServiceCreatePaidOrderRejectsInvalidMenuSelections(t *testing.T) {
	tests := []struct {
		name  string
		menu  appmenu.CashierMenu
		input CreatePaidOrderInput
	}{
		{
			name:  "wrong group option",
			menu:  seededCashierMenu(),
			input: createInputWithModifiers([]CreatePaidOrderModifierInput{{GroupSlug: "temperature", OptionSlug: "normal"}, {GroupSlug: "sugar", OptionSlug: "less"}}),
		},
		{
			name:  "unattached group",
			menu:  menuWithUnattachedModifierGroup(),
			input: createInputWithModifiers([]CreatePaidOrderModifierInput{{GroupSlug: "temperature", OptionSlug: "hot"}, {GroupSlug: "milk", OptionSlug: "oat"}}),
		},
		{
			name:  "duplicate group",
			menu:  seededCashierMenu(),
			input: createInputWithModifiers([]CreatePaidOrderModifierInput{{GroupSlug: "temperature", OptionSlug: "hot"}, {GroupSlug: "temperature", OptionSlug: "iced"}, {GroupSlug: "sugar", OptionSlug: "normal"}}),
		},
		{
			name:  "unknown item",
			menu:  seededCashierMenu(),
			input: createInputWithMenuItem("flat-white"),
		},
		{
			name:  "empty active menu",
			menu:  appmenu.CashierMenu{},
			input: validCreateInput(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(Dependencies{
				MenuReader: fakeMenuReader{menu: test.menu},
				Repository: &fakeOrderRepository{},
				Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
				Location:   time.UTC,
			})

			_, _, err := service.CreatePaidOrder(context.Background(), test.input)

			if !errors.Is(err, ErrInvalidOrder) {
				t.Fatalf("error = %v, want %v", err, ErrInvalidOrder)
			}
		})
	}
}

func TestServiceCreatePaidOrderAcceptsQRISAndDistinctModifierChoices(t *testing.T) {
	repository := &fakeOrderRepository{}
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: repository,
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	detail, result, err := service.CreatePaidOrder(context.Background(), CreatePaidOrderInput{
		ClientRequestID: "11111111-1111-4111-8111-111111111111",
		PaymentMethod:   "qris",
		Lines: []CreatePaidOrderLineInput{
			{
				MenuItemSlug: "kopi-susu",
				Quantity:     1,
				Modifiers:    []CreatePaidOrderModifierInput{{GroupSlug: "temperature", OptionSlug: "hot"}, {GroupSlug: "sugar", OptionSlug: "normal"}},
			},
			{
				MenuItemSlug: "kopi-susu",
				Quantity:     1,
				Modifiers:    []CreatePaidOrderModifierInput{{GroupSlug: "temperature", OptionSlug: "iced"}, {GroupSlug: "sugar", OptionSlug: "less"}},
			},
		},
	})

	if err != nil {
		t.Fatalf("CreatePaidOrder returned error: %v", err)
	}
	if result != CreatePaidOrderCreated {
		t.Fatalf("result = %q, want %q", result, CreatePaidOrderCreated)
	}
	if detail.PaymentMethod != "qris" {
		t.Fatalf("payment method = %q, want qris", detail.PaymentMethod)
	}
	if repository.saved.Draft.TotalRp != 39000 {
		t.Fatalf("total = %d, want 39000", repository.saved.Draft.TotalRp)
	}
}

func TestServiceCreatePaidOrderReturnsRepositoryFailure(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: &fakeOrderRepository{err: repositoryErr},
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	_, _, err := service.CreatePaidOrder(context.Background(), validCreateInput())

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("error = %v, want %v", err, repositoryErr)
	}
}

func TestServiceCreatePaidOrderMapsIdempotencyConflict(t *testing.T) {
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: &fakeOrderRepository{result: CreatePaidOrderIdempotencyConflict},
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	_, result, err := service.CreatePaidOrder(context.Background(), validCreateInput())

	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want %v", err, ErrIdempotencyConflict)
	}
	if result != CreatePaidOrderIdempotencyConflict {
		t.Fatalf("result = %q, want conflict", result)
	}
}

func TestServiceCreatePaidOrderReturnsExistingIdempotencyResult(t *testing.T) {
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: &fakeOrderRepository{result: CreatePaidOrderExisting},
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	_, result, err := service.CreatePaidOrder(context.Background(), validCreateInput())

	if err != nil {
		t.Fatalf("CreatePaidOrder returned error: %v", err)
	}
	if result != CreatePaidOrderExisting {
		t.Fatalf("result = %q, want %q", result, CreatePaidOrderExisting)
	}
}

func TestServiceCreatePaidOrderRejectsMalformedClientRequestID(t *testing.T) {
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: &fakeOrderRepository{},
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	_, _, err := service.CreatePaidOrder(context.Background(), CreatePaidOrderInput{
		ClientRequestID: "not-a-uuid",
		PaymentMethod:   "cash",
		Lines:           validCreateInput().Lines,
	})

	if !errors.Is(err, ErrInvalidClientRequestID) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidClientRequestID)
	}
}

func TestServiceCancelPaidOrderDelegatesSameDayBusinessDate(t *testing.T) {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	repository := &fakeOrderRepository{cancelDetail: PaidOrderDetail{OrderID: "12", Status: "cancelled"}}
	service := NewService(Dependencies{
		MenuReader: fakeMenuReader{menu: seededCashierMenu()},
		Repository: repository,
		Clock:      fixedClock{now: time.Date(2026, 6, 29, 17, 30, 0, 0, time.UTC)},
		Location:   location,
	})

	detail, result, err := service.CancelPaidOrder(context.Background(), CancelPaidOrderInput{OrderID: "12"})
	if err != nil {
		t.Fatalf("CancelPaidOrder returned error: %v", err)
	}

	if result != CancelPaidOrderCancelled {
		t.Fatalf("result = %q, want %q", result, CancelPaidOrderCancelled)
	}
	if detail.Status != "cancelled" {
		t.Fatalf("status = %q, want cancelled", detail.Status)
	}
	if repository.cancelInput.BusinessDate.String() != "2026-06-30" {
		t.Fatalf("business date = %q, want 2026-06-30", repository.cancelInput.BusinessDate.String())
	}
}

func TestServiceCancelPaidOrderRejectsMalformedOrderID(t *testing.T) {
	service := NewService(Dependencies{
		Repository: &fakeOrderRepository{},
		Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
		Location:   time.UTC,
	})

	_, _, err := service.CancelPaidOrder(context.Background(), CancelPaidOrderInput{OrderID: "001"})

	if !errors.Is(err, ErrInvalidOrderID) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidOrderID)
	}
}

func TestServiceCancelPaidOrderMapsRepositoryResults(t *testing.T) {
	tests := []struct {
		name       string
		result     CancelPaidOrderResult
		wantErr    error
		repository error
	}{
		{name: "not found", result: CancelPaidOrderNotFound, wantErr: ErrOrderNotFound},
		{name: "not cancellable", result: CancelPaidOrderNotCancellable, wantErr: ErrOrderNotCancellable},
		{name: "repository failure", repository: errors.New("database unavailable"), wantErr: errors.New("database unavailable")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(Dependencies{
				Repository: &fakeOrderRepository{cancelResult: test.result, err: test.repository},
				Clock:      fixedClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)},
				Location:   time.UTC,
			})

			_, _, err := service.CancelPaidOrder(context.Background(), CancelPaidOrderInput{OrderID: "12"})

			if test.repository != nil {
				if !errors.Is(err, test.repository) {
					t.Fatalf("error = %v, want %v", err, test.repository)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func validCreateInput() CreatePaidOrderInput {
	return CreatePaidOrderInput{
		ClientRequestID: "11111111-1111-4111-8111-111111111111",
		PaymentMethod:   "cash",
		Lines: []CreatePaidOrderLineInput{{
			MenuItemSlug: "kopi-susu",
			Quantity:     1,
			Modifiers: []CreatePaidOrderModifierInput{
				{GroupSlug: "temperature", OptionSlug: "hot"},
				{GroupSlug: "sugar", OptionSlug: "normal"},
			},
		}},
	}
}

func createInputWithModifiers(modifiers []CreatePaidOrderModifierInput) CreatePaidOrderInput {
	input := validCreateInput()
	input.Lines[0].Modifiers = modifiers
	return input
}

func createInputWithMenuItem(menuItemSlug string) CreatePaidOrderInput {
	input := validCreateInput()
	input.Lines[0].MenuItemSlug = menuItemSlug
	return input
}

func seededCashierMenu() appmenu.CashierMenu {
	return appmenu.CashierMenu{Categories: []appmenu.CashierMenuCategory{{
		Name: "Coffee",
		Slug: "coffee",
		Items: []appmenu.CashierMenuItem{{
			ID:      10,
			Name:    "Kopi Susu",
			Slug:    "kopi-susu",
			PriceRp: 18000,
			ModifierGroups: []appmenu.CashierModifierGroup{
				{
					ID:            20,
					Name:          "Temperature",
					Slug:          "temperature",
					Required:      true,
					SelectionType: "single",
					Options: []appmenu.CashierModifierOption{
						{ID: 30, Name: "Hot", Slug: "hot", PriceDeltaRp: 0},
						{ID: 31, Name: "Iced", Slug: "iced", PriceDeltaRp: 2000},
					},
				},
				{
					ID:            21,
					Name:          "Sugar",
					Slug:          "sugar",
					Required:      true,
					SelectionType: "single",
					Options: []appmenu.CashierModifierOption{
						{ID: 32, Name: "Normal", Slug: "normal", PriceDeltaRp: 0},
						{ID: 33, Name: "Less Sugar", Slug: "less", PriceDeltaRp: 1000},
					},
				},
			},
		}},
	}}}
}

func menuWithUnattachedModifierGroup() appmenu.CashierMenu {
	menu := seededCashierMenu()
	menu.Categories[0].Items[0].ModifierGroups = menu.Categories[0].Items[0].ModifierGroups[:1]
	return menu
}

type fakeMenuReader struct {
	menu appmenu.CashierMenu
	err  error
}

func (reader fakeMenuReader) GetCashierMenu(context.Context) (appmenu.CashierMenu, error) {
	return reader.menu, reader.err
}

type fakeOrderRepository struct {
	saved        PersistPaidOrderInput
	result       CreatePaidOrderResult
	err          error
	cancelInput  PersistCancelPaidOrderInput
	cancelDetail PaidOrderDetail
	cancelResult CancelPaidOrderResult
}

func (repo *fakeOrderRepository) CreatePaidOrder(_ context.Context, input PersistPaidOrderInput) (PaidOrderDetail, CreatePaidOrderResult, error) {
	repo.saved = input
	result := repo.result
	if result == "" {
		result = CreatePaidOrderCreated
	}
	return PaidOrderDetail{
		OrderID:       "1",
		QueueNumber:   1,
		BusinessDate:  input.BusinessDate.String(),
		Status:        "paid",
		PaymentMethod: string(input.Draft.PaymentMethod),
		PaidAt:        input.PaidAt,
		Note:          input.Draft.Note,
		TotalRp:       input.Draft.TotalRp,
		Lines: []PaidOrderLineDetail{{
			MenuItemSlug: input.Draft.Lines[0].MenuItemSlug,
			MenuItemName: input.Draft.Lines[0].MenuItemName,
			UnitPriceRp:  input.Draft.Lines[0].UnitPriceRp,
			Quantity:     input.Draft.Lines[0].Quantity,
			LineTotalRp:  input.Draft.Lines[0].LineTotalRp,
		}},
	}, result, repo.err
}

func (repo *fakeOrderRepository) CancelPaidOrder(_ context.Context, input PersistCancelPaidOrderInput) (PaidOrderDetail, CancelPaidOrderResult, error) {
	repo.cancelInput = input
	result := repo.cancelResult
	if result == "" {
		result = CancelPaidOrderCancelled
	}
	return repo.cancelDetail, result, repo.err
}

type fixedClock struct {
	now time.Time
}

func (clock fixedClock) Now() time.Time {
	return clock.now
}
