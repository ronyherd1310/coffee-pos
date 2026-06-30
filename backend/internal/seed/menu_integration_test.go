//go:build integration

package seed

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"coffee-pos/backend/internal/adapters/postgres"
	"coffee-pos/backend/internal/adapters/postgres/sqlc"
	"coffee-pos/backend/internal/config"

	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestSeedInitialMenuMigratesAndSeedsExactMenuIdempotently(t *testing.T) {
	db := startSeedTestDB(t)
	ctx := context.Background()

	if _, err := postgres.Migrate(ctx, db, os.DirFS("../../migrations")); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := SeedInitialMenu(ctx, db); err != nil {
		t.Fatalf("first seed failed: %v", err)
	}
	if err := SeedInitialMenu(ctx, db); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}

	queries := sqlc.New(db)
	categories, err := queries.ListMenuCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].Name != "Coffee" {
		t.Fatalf("expected exactly one Coffee category, got %+v", categories)
	}

	items, err := queries.ListMenuItems(ctx)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 2 || items[0].Name != "Americano" || items[0].PriceRp != 18000 || items[1].Name != "Latte" || items[1].PriceRp != 25000 {
		t.Fatalf("unexpected seeded items: %+v", items)
	}

	groups, err := queries.ListModifierGroups(ctx)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 2 || groups[0].Name != "Temperature" || groups[1].Name != "Sugar" {
		t.Fatalf("unexpected seeded groups: %+v", groups)
	}
	groupNames := map[int64]string{
		groups[0].ID: groups[0].Name,
		groups[1].ID: groups[1].Name,
	}

	options, err := queries.ListModifierOptions(ctx)
	if err != nil {
		t.Fatalf("list options: %v", err)
	}
	if len(options) != 5 {
		t.Fatalf("expected exactly 5 modifier options, got %+v", options)
	}
	optionsByGroup := map[string][]string{}
	for _, option := range options {
		if option.PriceDeltaRp != 0 {
			t.Fatalf("expected zero price delta, got %+v", option)
		}
		optionsByGroup[groupNames[option.ModifierGroupID]] = append(optionsByGroup[groupNames[option.ModifierGroupID]], option.Name)
	}
	assertSeedOptions(t, optionsByGroup["Temperature"], []string{"Hot", "Iced"})
	assertSeedOptions(t, optionsByGroup["Sugar"], []string{"Normal", "Less sugar", "No sugar"})

	linkCount, err := queries.CountMenuItemModifierGroups(ctx)
	if err != nil {
		t.Fatalf("count item/group links: %v", err)
	}
	if linkCount != 4 {
		t.Fatalf("expected exactly 4 item/group links, got %d", linkCount)
	}
}

func assertSeedOptions(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected options %v, got %v", expected, actual)
	}
	for index, expectedOption := range expected {
		if actual[index] != expectedOption {
			t.Fatalf("expected option %d to be %q, got %q", index, expectedOption, actual[index])
		}
	}
}

func startSeedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()
	container, err := testpostgres.Run(
		ctx,
		"docker.io/library/postgres:16-alpine",
		testpostgres.WithDatabase("coffee_pos_test"),
		testpostgres.WithUsername("coffee_pos"),
		testpostgres.WithPassword("coffee_pos_dev"),
		testpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	db, err := postgres.Open(ctx, config.DatabaseConfig{
		URL:          databaseURL,
		MaxOpenConns: 3,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open postgres db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})

	return db
}
