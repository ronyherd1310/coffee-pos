package menu

import (
	"context"

	domainmenu "coffee-pos/backend/internal/domain/menu"
)

type SeedRepository interface {
	SeedMenu(ctx context.Context, seed domainmenu.Seed) error
}

type Repository interface {
	SeedRepository
	GetCashierMenu(ctx context.Context) (CashierMenu, error)
}

type CashierMenu struct {
	Categories []CashierMenuCategory
}

type CashierMenuCategory struct {
	ID    int64
	Name  string
	Slug  string
	Items []CashierMenuItem
}

type CashierMenuItem struct {
	ID             int64
	Name           string
	Slug           string
	PriceRp        int64
	ModifierGroups []CashierModifierGroup
}

type CashierModifierGroup struct {
	ID            int64
	Name          string
	Slug          string
	Required      bool
	SelectionType string
	Options       []CashierModifierOption
}

type CashierModifierOption struct {
	ID           int64
	Name         string
	Slug         string
	PriceDeltaRp int64
}
