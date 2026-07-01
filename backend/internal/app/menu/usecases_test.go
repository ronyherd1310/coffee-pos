package menu

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainmenu "coffee-pos/backend/internal/domain/menu"
)

func TestServiceSeedInitialMenuPersistsApprovedSeed(t *testing.T) {
	repository := &fakeMenuRepository{}
	service := NewService(Dependencies{Repository: repository})

	if err := service.SeedInitialMenu(context.Background()); err != nil {
		t.Fatalf("expected seed to succeed: %v", err)
	}

	if repository.calls != 1 {
		t.Fatalf("expected repository to be called once, got %d", repository.calls)
	}
	if len(repository.seed.Categories) != 4 || repository.seed.Categories[0].Name != "Coffee" {
		t.Fatalf("expected approved seed to be persisted, got %+v", repository.seed)
	}
}

func TestServiceSeedRejectsInvalidSeedBeforeRepositoryWrite(t *testing.T) {
	repository := &fakeMenuRepository{}
	service := NewService(Dependencies{Repository: repository})
	seed := domainmenu.ApprovedSeed()
	seed.Items[0].PriceRp = -1

	err := service.Seed(context.Background(), seed)
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if !strings.Contains(err.Error(), "invalid menu seed") {
		t.Fatalf("expected invalid seed error, got %q", err.Error())
	}
	if repository.calls != 0 {
		t.Fatalf("expected repository not to be called, got %d calls", repository.calls)
	}
}

func TestServiceSeedReturnsRepositoryFailures(t *testing.T) {
	repository := &fakeMenuRepository{seedErr: errors.New("write failed")}
	service := NewService(Dependencies{Repository: repository})

	err := service.SeedInitialMenu(context.Background())
	if err == nil {
		t.Fatal("expected repository failure")
	}
	if !strings.Contains(err.Error(), "persist menu seed") {
		t.Fatalf("expected persistence error, got %q", err.Error())
	}
}

func TestServiceGetCashierMenuReturnsReadModel(t *testing.T) {
	repository := &fakeMenuRepository{
		menu: CashierMenu{
			Categories: []CashierMenuCategory{{
				Name: "Coffee",
				Slug: "coffee",
				Items: []CashierMenuItem{{
					Name:    "Kopi Susu",
					Slug:    "kopi-susu",
					PriceRp: 18000,
					ModifierGroups: []CashierModifierGroup{{
						Name:          "Temperature",
						Slug:          "temperature",
						Required:      true,
						SelectionType: "single",
						Options: []CashierModifierOption{{
							Name:         "Hot",
							Slug:         "hot",
							PriceDeltaRp: 0,
						}},
					}},
				}},
			}},
		},
	}
	service := NewService(Dependencies{Repository: repository})

	menu, err := service.GetCashierMenu(context.Background())
	if err != nil {
		t.Fatalf("GetCashierMenu returned error: %v", err)
	}

	if got := menu.Categories[0].Items[0].ModifierGroups[0].Options[0].Slug; got != "hot" {
		t.Fatalf("option slug = %q, want hot", got)
	}
}

func TestServiceGetCashierMenuAllowsEmptyMenu(t *testing.T) {
	service := NewService(Dependencies{Repository: &fakeMenuRepository{}})

	menu, err := service.GetCashierMenu(context.Background())
	if err != nil {
		t.Fatalf("GetCashierMenu returned error: %v", err)
	}

	if len(menu.Categories) != 0 {
		t.Fatalf("categories = %d, want 0", len(menu.Categories))
	}
}

func TestServiceGetCashierMenuReturnsRepositoryFailures(t *testing.T) {
	service := NewService(Dependencies{Repository: &fakeMenuRepository{readErr: errors.New("read failed")}})

	_, err := service.GetCashierMenu(context.Background())
	if err == nil {
		t.Fatal("expected repository failure")
	}
	if !strings.Contains(err.Error(), "read cashier menu") {
		t.Fatalf("expected read error, got %q", err.Error())
	}
}

type fakeMenuRepository struct {
	calls   int
	seed    domainmenu.Seed
	seedErr error
	menu    CashierMenu
	readErr error
}

func (repo *fakeMenuRepository) SeedMenu(_ context.Context, seed domainmenu.Seed) error {
	repo.calls++
	repo.seed = seed
	return repo.seedErr
}

func (repo *fakeMenuRepository) GetCashierMenu(_ context.Context) (CashierMenu, error) {
	return repo.menu, repo.readErr
}
