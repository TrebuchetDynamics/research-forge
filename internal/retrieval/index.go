package retrieval

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	_ "modernc.org/sqlite"
)

// SearchAdapter is the passage retrieval seam for local and optional search backends.
type SearchAdapter interface {
	Rebuild([]parsing.ParsedDocument) error
	Retrieve(string) ([]PassageResult, error)
	Close() error
}

// VectorAdapter is the optional vector-store seam for future Qdrant integration.
type VectorAdapter interface{ VectorBackendName() string }

// EmbeddingAdapter is the optional embedding-provider seam.
type EmbeddingAdapter interface{ EmbeddingBackendName() string }

// NoopVectorAdapter reserves the optional Qdrant seam without requiring the service.
type NoopVectorAdapter struct{}

func (NoopVectorAdapter) VectorBackendName() string { return "noop" }

// NoopEmbeddingAdapter reserves the optional embeddings seam without requiring a provider.
type NoopEmbeddingAdapter struct{}

func (NoopEmbeddingAdapter) EmbeddingBackendName() string { return "noop" }

// PassageResult is one full-text retrieval hit.
type PassageResult struct {
	PaperID   string
	SectionID string
	PassageID string
	Text      string
}

// SQLiteIndex stores parsed passages in a local SQLite FTS index.
type SQLiteIndex struct{ db *sql.DB }

// OpenSQLiteIndex opens a local SQLite FTS passage index.
func OpenSQLiteIndex(path string) (*SQLiteIndex, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	idx := &SQLiteIndex{db: db}
	if err := idx.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return idx, nil
}

func (i *SQLiteIndex) migrate() error {
	_, err := i.db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS passages_fts USING fts5(paper_id UNINDEXED, section_id UNINDEXED, passage_id UNINDEXED, text);`)
	return err
}

// Rebuild replaces all indexed parsed passages.
func (i *SQLiteIndex) Rebuild(docs []parsing.ParsedDocument) error {
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM passages_fts`); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO passages_fts(paper_id, section_id, passage_id, text) VALUES (?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, doc := range docs {
		for _, section := range doc.Sections {
			for _, passage := range section.Passages {
				if _, err := stmt.Exec(passage.PaperID, passage.SectionID, passage.ID, passage.Text); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	}
	return tx.Commit()
}

// Retrieve returns passages matching a full-text query.
func (i *SQLiteIndex) Retrieve(query string) ([]PassageResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("retrieve query is required")
	}
	rows, err := i.db.Query(`SELECT paper_id, section_id, passage_id, text FROM passages_fts WHERE passages_fts MATCH ? LIMIT 20`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []PassageResult
	for rows.Next() {
		var result PassageResult
		if err := rows.Scan(&result.PaperID, &result.SectionID, &result.PassageID, &result.Text); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// Close closes the SQLite index.
func (i *SQLiteIndex) Close() error {
	if i == nil || i.db == nil {
		return nil
	}
	return i.db.Close()
}
