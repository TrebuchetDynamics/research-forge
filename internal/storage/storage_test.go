package storage

import (
	"database/sql"
	"os"
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

func TestInitializeDoesNotWriteThroughSymlinkedMigrationBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rforge.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE existing_data (value TEXT)`); err != nil {
		_ = db.Close()
		t.Fatalf("seed existing database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seeded database: %v", err)
	}

	target := filepath.Join(t.TempDir(), "outside.txt")
	const sentinel = "do not overwrite"
	if err := os.WriteFile(target, []byte(sentinel), 0o640); err != nil {
		t.Fatalf("write target: %v", err)
	}
	backupPath := path + ".pre-migration.bak"
	if err := os.Symlink(target, backupPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	store, err := Initialize(path)
	if err == nil {
		_ = store.Close()
		t.Fatal("Initialize followed symlinked migration backup path")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != sentinel {
		t.Fatalf("symlink target changed: got %q, want %q", data, sentinel)
	}
}

func TestCheckExistingDoesNotReadThroughSymlinkedDatabasePath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "outside.sqlite")
	db, err := sql.Open("sqlite", target)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE outside_data (value TEXT)`); err != nil {
		_ = db.Close()
		t.Fatalf("seed outside database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close outside database: %v", err)
	}

	path := filepath.Join(t.TempDir(), "rforge.sqlite")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := CheckExisting(path); err == nil {
		t.Fatal("CheckExisting followed symlinked database path")
	}
}

func TestCheckExistingDoesNotReadThroughSymlinkedParent(t *testing.T) {
	outsideDir := t.TempDir()
	target := filepath.Join(outsideDir, "outside.sqlite")
	db, err := sql.Open("sqlite", target)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE outside_data (value TEXT)`); err != nil {
		_ = db.Close()
		t.Fatalf("seed outside database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close outside database: %v", err)
	}

	parent := filepath.Join(t.TempDir(), "data")
	if err := os.Symlink(outsideDir, parent); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := CheckExisting(filepath.Join(parent, "outside.sqlite")); err == nil {
		t.Fatal("CheckExisting followed symlinked parent directory")
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

func TestInitializeDoesNotWriteThroughSymlinkedDatabasePath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "outside.sqlite")
	if err := os.WriteFile(target, nil, 0o640); err != nil {
		t.Fatalf("write target: %v", err)
	}
	path := filepath.Join(t.TempDir(), "rforge.sqlite")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	store, err := Initialize(path)
	if err == nil {
		_ = store.Close()
		t.Fatal("Initialize followed symlinked database path")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("symlink target changed: got %d bytes", len(data))
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat database path: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("database symlink was replaced: mode=%v", info.Mode())
	}
}

func TestInitializeDoesNotCreateDatabaseThroughSymlinkedParent(t *testing.T) {
	outsideDir := t.TempDir()
	parent := filepath.Join(t.TempDir(), "data")
	if err := os.Symlink(outsideDir, parent); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	store, err := Initialize(filepath.Join(parent, "rforge.sqlite"))
	if err == nil {
		_ = store.Close()
		t.Fatal("Initialize followed symlinked parent directory")
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "rforge.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("database created through symlinked parent: %v", err)
	}
}

func TestInitializeCreatesMissingParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "data", "rforge.sqlite")
	store, err := Initialize(path)
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat initialized database: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("initialized database is not a regular file: mode=%v", info.Mode())
	}
}
