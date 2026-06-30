package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type MigrationResult struct {
	Applied int
}

type migration struct {
	Version string
	Name    string
	SQL     string
}

func Migrate(ctx context.Context, db *sql.DB, migrationsFS fs.FS) (MigrationResult, error) {
	migrations, err := listMigrations(migrationsFS)
	if err != nil {
		return MigrationResult{}, err
	}

	if _, err := db.ExecContext(ctx, `
		create table if not exists schema_migrations (
			version text primary key,
			name text not null,
			applied_at timestamptz not null default now()
		)
	`); err != nil {
		return MigrationResult{}, fmt.Errorf("ensure schema migrations table: %w", err)
	}

	var result MigrationResult
	for _, migration := range migrations {
		applied, err := migrationApplied(ctx, db, migration.Version)
		if err != nil {
			return MigrationResult{}, err
		}
		if applied {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return MigrationResult{}, fmt.Errorf("begin migration %s: %w", migration.Name, err)
		}
		if err := applyMigration(ctx, tx, migration); err != nil {
			_ = tx.Rollback()
			return MigrationResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return MigrationResult{}, fmt.Errorf("commit migration %s: %w", migration.Name, err)
		}
		result.Applied++
	}

	return result, nil
}

func applyMigration(ctx context.Context, tx *sql.Tx, migration migration) error {
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", migration.Name, err)
	}
	if _, err := tx.ExecContext(ctx, `
		insert into schema_migrations (version, name)
		values ($1, $2)
	`, migration.Version, migration.Name); err != nil {
		return fmt.Errorf("record migration %s: %w", migration.Name, err)
	}
	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, `
		select exists(select 1 from schema_migrations where version = $1)
	`, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return exists, nil
}

func listMigrations(migrationsFS fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}

	seenVersions := map[string]string{}
	var migrations []migration
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		version := migrationVersion(entry.Name())
		if version == "" {
			return nil, fmt.Errorf("migration %q must start with a version prefix", entry.Name())
		}
		if previous := seenVersions[version]; previous != "" {
			return nil, fmt.Errorf("duplicate migration version %s in %q and %q", version, previous, entry.Name())
		}
		seenVersions[version] = entry.Name()

		contents, err := fs.ReadFile(migrationsFS, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		migrations = append(migrations, migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(contents),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}

func migrationVersion(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	version, _, ok := strings.Cut(name, "_")
	if !ok {
		return ""
	}
	return version
}
