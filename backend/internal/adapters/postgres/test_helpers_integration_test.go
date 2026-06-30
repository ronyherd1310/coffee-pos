//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"coffee-pos/backend/internal/config"

	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func startPostgresTestDB(t *testing.T) *sql.DB {
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
		t.Fatalf("get postgres connection string: %v", err)
	}

	db, err := Open(ctx, config.DatabaseConfig{
		URL:          databaseURL,
		MaxOpenConns: 3,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open postgres db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close postgres db: %v", err)
		}
	})

	return db
}

func applyTestMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	result, err := Migrate(context.Background(), db, os.DirFS("../../../migrations"))
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if result.Applied == 0 {
		t.Fatal("expected initial migrations to apply")
	}
}
