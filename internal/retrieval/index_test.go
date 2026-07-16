package retrieval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestSQLiteFTSIndexRebuildAndRetrievePassages(t *testing.T) {
	index, err := OpenSQLiteIndex(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteIndex returned error: %v", err)
	}
	defer index.Close()
	doc := parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "paper-1-sec-1", Title: "Introduction", Passages: []parsing.Passage{{ID: "paper-1-sec-1-p-1", PaperID: "paper-1", SectionID: "paper-1-sec-1", Text: "Solar fuel catalysts split water."}}}}}
	if err := index.Rebuild([]parsing.ParsedDocument{doc}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	results, err := index.Retrieve("solar catalysts")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if len(results) != 1 || results[0].PaperID != "paper-1" || results[0].SectionID != "paper-1-sec-1" || results[0].PassageID != "paper-1-sec-1-p-1" || results[0].Text != "Solar fuel catalysts split water." {
		t.Fatalf("results = %#v", results)
	}
}

func TestOpenSQLiteIndexDoesNotFollowSymlinkedDatabasePath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "outside.db")
	if err := os.WriteFile(target, nil, 0o640); err != nil {
		t.Fatalf("write target: %v", err)
	}
	path := filepath.Join(t.TempDir(), "index.db")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	index, err := OpenSQLiteIndex(path)
	if err == nil {
		_ = index.Close()
		t.Fatal("OpenSQLiteIndex followed symlinked database path")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("OpenSQLiteIndex error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("symlink target changed: got %d bytes", len(data))
	}
}

func TestOpenSQLiteIndexDoesNotCreateDatabaseThroughSymlinkedParent(t *testing.T) {
	outsideDir := t.TempDir()
	parent := filepath.Join(t.TempDir(), "data")
	if err := os.Symlink(outsideDir, parent); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	index, err := OpenSQLiteIndex(filepath.Join(parent, "index.db"))
	if err == nil {
		_ = index.Close()
		t.Fatal("OpenSQLiteIndex followed symlinked parent directory")
	}
	if !strings.Contains(err.Error(), "parent is not a directory") {
		t.Fatalf("OpenSQLiteIndex error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "index.db")); !os.IsNotExist(err) {
		t.Fatalf("database created through symlinked parent: %v", err)
	}
}

func TestOpenSQLiteIndexCreatesMissingParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "data", "index.db")
	index, err := OpenSQLiteIndex(path)
	if err != nil {
		t.Fatalf("OpenSQLiteIndex returned error: %v", err)
	}
	if err := index.Close(); err != nil {
		t.Fatalf("close index: %v", err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat created parent: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("created parent is not a directory: mode=%v", info.Mode())
	}
}

func TestAdapterSeamsExistForOptionalSearchVectorAndEmbeddingBackends(t *testing.T) {
	var _ SearchAdapter = (*SQLiteIndex)(nil)
	var _ SearchAdapter = (*OpenSearchIndex)(nil)
	var _ SearchAdapter = (*QdrantIndex)(nil)
	var _ VectorAdapter = (*QdrantIndex)(nil)
	var _ VectorAdapter = NoopVectorAdapter{}
	var _ EmbeddingAdapter = NoopEmbeddingAdapter{}
}
