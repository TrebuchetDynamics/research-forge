package storage

import (
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Driver names a storage adapter.
type Driver string

const (
	// DriverSQLite stores project data in a local SQLite database.
	DriverSQLite Driver = "sqlite"
	// DriverPostgres reserves the future PostgreSQL adapter seam.
	DriverPostgres Driver = "postgres"
)

// Config selects a storage adapter and location.
type Config struct {
	Driver Driver
	Path   string
}

// Handle is the storage boundary used by application services.
type Handle interface {
	HealthCheck() error
	Close() error
}

// Store is a local project database handle.
type Store struct {
	db *sql.DB
}

var _ Handle = (*Store)(nil)

// Open opens a storage adapter from config.
func Open(config Config) (*Store, error) {
	switch config.Driver {
	case "", DriverSQLite:
		return Initialize(config.Path)
	case DriverPostgres:
		return nil, fmt.Errorf("postgres storage adapter is not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", config.Driver)
	}
}

// Initialize opens or creates a SQLite database and applies migrations.
func Initialize(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if err := ensureParent(path); err != nil {
		return nil, err
	}
	if err := backupBeforeMigrations(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// CheckExisting verifies that an existing SQLite database can answer a simple query without creating, migrating, or backing it up.
func CheckExisting(path string) error {
	if path == "" {
		return fmt.Errorf("database path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("database path is a directory")
	}
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	var one int
	return db.QueryRow("SELECT 1").Scan(&one)
}

// HealthCheck verifies that the database can answer a simple query.
func (s *Store) HealthCheck() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store is not open")
	}
	var one int
	return s.db.QueryRow("SELECT 1").Scan(&one)
}

// Close closes the underlying database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func backupBeforeMigrations(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() || info.Size() == 0 {
		return nil
	}

	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	backup, err := os.OpenFile(path+".pre-migration.bak", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	return copyAndClose(backup, source)
}

// copyAndClose copies src into dst and closes dst, returning the copy error
// if any, otherwise the close error. A bare deferred Close would silently
// discard a flush failure (e.g. disk full) and report a successful backup
// that was actually truncated.
func copyAndClose(dst io.WriteCloser, src io.Reader) error {
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (s *Store) migrate() error {
	migration := `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO schema_migrations(version) VALUES (1);
`
	_, err := s.db.Exec(migration)
	return err
}

func ensureParent(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return nil
}
