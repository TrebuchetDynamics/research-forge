package storage

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInitializeBacksUpExistingDatabaseBeforeMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rforge.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE existing_data (value TEXT); INSERT INTO existing_data(value) VALUES ('keep');`); err != nil {
		_ = db.Close()
		t.Fatalf("seed existing database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seeded database: %v", err)
	}

	store, err := Initialize(path)
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	defer store.Close()

	backup, err := sql.Open("sqlite", path+".pre-migration.bak")
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer backup.Close()
	var value string
	if err := backup.QueryRow(`SELECT value FROM existing_data`).Scan(&value); err != nil {
		t.Fatalf("backup missing existing data: %v", err)
	}
	if value != "keep" {
		t.Fatalf("backup value = %q, want keep", value)
	}
	var migrationTables int
	if err := backup.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&migrationTables); err != nil {
		t.Fatalf("query backup schema: %v", err)
	}
	if migrationTables != 0 {
		t.Fatalf("backup contains schema_migrations, want pre-migration copy")
	}
}

func TestOpenUsesSQLiteDriverAndNamesPostgresFutureAdapter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rforge.sqlite")
	store, err := Open(Config{Driver: DriverSQLite, Path: path})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()
	if err := store.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}

	if _, err := Open(Config{Driver: DriverPostgres, Path: "postgres://example"}); err == nil {
		t.Fatalf("Open with postgres driver returned nil error, want future adapter error")
	}
}

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
