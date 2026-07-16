package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

func TestExecuteIndexSQLiteRejectsInvalidRetrievalLockBeforeCreatingDatabase(t *testing.T) {
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "SQLite text"}}}}})
	dataDir := filepath.Join(project, "data")
	if err := os.MkdirAll(filepath.Join(dataDir, "retrieval.lock.json"), 0o755); err != nil {
		t.Fatalf("block retrieval lock: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "index", "rebuild", "--backend", "sqlite"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("index code = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"index_metadata_preflight_failed"`) {
		t.Fatalf("missing metadata preflight error: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dataDir, "retrieval.db")); !os.IsNotExist(err) {
		t.Fatalf("SQLite database exists after metadata preflight failure: %v", err)
	}
}

func TestExecuteIndexHybridRejectsInvalidRetrievalLockBeforeLocalOrRemoteMutation(t *testing.T) {
	var requests atomic.Int32
	server := newQdrantIndexTestServer(t, func() { requests.Add(1) })
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	t.Setenv("RFORGE_QDRANT_PAYLOAD_PRIVACY", retrieval.PayloadPrivacyRedacted)
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Hybrid text"}}}}})
	dataDir := filepath.Join(project, "data")
	if err := os.MkdirAll(filepath.Join(dataDir, "retrieval.lock.json"), 0o755); err != nil {
		t.Fatalf("block retrieval lock: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "index", "rebuild", "--backend", "hybrid"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("index code = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"index_metadata_preflight_failed"`) {
		t.Fatalf("missing metadata preflight error: %s", stdout.String())
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("Qdrant received %d requests before hybrid metadata preflight failure", got)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "retrieval.db")); !os.IsNotExist(err) {
		t.Fatalf("hybrid SQLite database exists after metadata preflight failure: %v", err)
	}
}

func TestExecuteIndexSQLiteDoesNotWriteThroughSymlinkedDatabase(t *testing.T) {
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "SQLite text"}}}}})
	dataDir := filepath.Join(project, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.db")
	outsideBefore := []byte{}
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside database target: %v", err)
	}
	databasePath := filepath.Join(dataDir, "retrieval.db")
	if err := os.Symlink(outsidePath, databasePath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "index", "rebuild", "--backend", "sqlite"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("index code = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"index_open_failed"`) {
		t.Fatalf("missing index open error: %s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside database target: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("outside database target changed through symlink: got %d bytes, want %d", len(outsideAfter), len(outsideBefore))
	}
	info, err := os.Lstat(databasePath)
	if err != nil {
		t.Fatalf("lstat database path: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("database symlink was replaced: mode=%v", info.Mode())
	}
}

func TestExecuteIndexSQLiteDoesNotCreateDatabaseThroughSymlinkedDataDirectory(t *testing.T) {
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "SQLite text"}}}}})
	outsideDir := t.TempDir()
	dataDir := filepath.Join(project, "data")
	if err := os.Symlink(outsideDir, dataDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "index", "rebuild", "--backend", "sqlite"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("index code = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"index_metadata_preflight_failed"`) {
		t.Fatalf("missing metadata preflight error: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "retrieval.db")); !os.IsNotExist(err) {
		t.Fatalf("database created through symlinked data directory: %v", err)
	}
	info, err := os.Lstat(dataDir)
	if err != nil {
		t.Fatalf("lstat data directory: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("data directory symlink was replaced: mode=%v", info.Mode())
	}
}
