//go:build integration

package postgres

import (
	"context"
	"testing"
)

func TestMigrationsCreateOrderTables(t *testing.T) {
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)

	tableNames := []string{
		"orders",
		"order_lines",
		"order_line_modifiers",
		"daily_queue_counters",
	}
	for _, tableName := range tableNames {
		t.Run(tableName, func(t *testing.T) {
			var exists bool
			if err := db.QueryRowContext(context.Background(), `
				select exists (
					select 1
					from information_schema.tables
					where table_schema = 'public' and table_name = $1
				)
			`, tableName).Scan(&exists); err != nil {
				t.Fatalf("check table exists: %v", err)
			}
			if !exists {
				t.Fatalf("expected table %q to exist", tableName)
			}
		})
	}
}
