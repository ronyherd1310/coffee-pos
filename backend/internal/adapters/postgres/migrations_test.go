package postgres

import (
	"testing"
	"testing/fstest"
)

func TestListMigrationsReturnsSQLFilesInVersionOrder(t *testing.T) {
	migrations, err := listMigrations(fstest.MapFS{
		"000002_second.sql": {Data: []byte("select 2;")},
		"notes.txt":         {Data: []byte("ignore me")},
		"000001_first.sql":  {Data: []byte("select 1;")},
	})
	if err != nil {
		t.Fatalf("expected migrations to load: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}
	if migrations[0].Version != "000001" || migrations[0].Name != "000001_first.sql" {
		t.Fatalf("expected first migration to be 000001_first.sql, got %+v", migrations[0])
	}
	if migrations[1].Version != "000002" || migrations[1].Name != "000002_second.sql" {
		t.Fatalf("expected second migration to be 000002_second.sql, got %+v", migrations[1])
	}
}

func TestListMigrationsRejectsDuplicateVersions(t *testing.T) {
	_, err := listMigrations(fstest.MapFS{
		"000001_first.sql": {Data: []byte("select 1;")},
		"000001_again.sql": {Data: []byte("select 1;")},
	})
	if err == nil {
		t.Fatal("expected duplicate migration versions to fail")
	}
}
