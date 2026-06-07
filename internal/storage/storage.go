package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

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
	defer backup.Close()
	_, err = io.Copy(backup, source)
	return err
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
