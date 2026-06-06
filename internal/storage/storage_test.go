package storage

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInitializeCreatesSQLiteDatabaseAndMigrationRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rforge.sqlite")

	store, err := Initialize(path)
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	defer store.Close()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	var version int
	if err := db.QueryRow(`SELECT version FROM schema_migrations WHERE version = 1`).Scan(&version); err != nil {
		t.Fatalf("query schema migration: %v", err)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}
