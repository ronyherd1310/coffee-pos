package menu

import (
	"context"

	domainmenu "coffee-pos/backend/internal/domain/menu"
)

type SeedRepository interface {
	SeedMenu(ctx context.Context, seed domainmenu.Seed) error
}
