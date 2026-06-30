package seed

import (
	"context"
	"database/sql"

	"coffee-pos/backend/internal/adapters/postgres"
	appmenu "coffee-pos/backend/internal/app/menu"
)

func SeedInitialMenu(ctx context.Context, db *sql.DB) error {
	service := appmenu.NewService(appmenu.Dependencies{
		Repository: postgres.NewMenuRepository(db),
	})
	return service.SeedInitialMenu(ctx)
}
