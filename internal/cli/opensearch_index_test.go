package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

func TestExecuteIndexOpenSearchWritesMappingLockAndBulkReport(t *testing.T) {
	server := newOpenSearchIndexTestServer(t)
	defer server.Close()
	t.Setenv("RFORGE_OPENSEARCH_URL", server.URL)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "OpenSearch text"}}}}})
	code := Execute([]string{"--project", project, "index", "rebuild", "--backend", "opensearch"}, ioDiscard{}, ioDiscard{})
	if code != 0 {
		t.Fatalf("index code = %d", code)
	}
	lockData, err := os.ReadFile(filepath.Join(project, "data", "opensearch.mapping.lock.json"))
	if err != nil {
		t.Fatalf("read mapping lock: %v", err)
	}
	if !strings.Contains(string(lockData), retrieval.OpenSearchMappingVersion) {
		t.Fatalf("mapping lock = %s", string(lockData))
	}
	var report retrieval.OpenSearchBulkReport
	data, err := os.ReadFile(filepath.Join(project, "data", "opensearch.bulk-report.json"))
	if err != nil {
		t.Fatalf("read bulk report: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Attempted != 1 || report.Indexed != 1 || report.MappingVersion != retrieval.OpenSearchMappingVersion {
		t.Fatalf("report = %#v", report)
	}
}

func TestExecuteIndexOpenSearchPreservesMetadataWhenRetrievalLockCannotBeWritten(t *testing.T) {
	var requests atomic.Int32
	server := newOpenSearchIndexTestServer(t, func() { requests.Add(1) })
	defer server.Close()
	t.Setenv("RFORGE_OPENSEARCH_URL", server.URL)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "OpenSearch text"}}}}})
	dataDir := filepath.Join(project, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	prior := map[string][]byte{
		filepath.Join(dataDir, "opensearch.bulk-report.json"):  []byte("prior bulk report\n"),
		filepath.Join(dataDir, "opensearch.mapping.lock.json"): []byte("prior mapping lock\n"),
	}
	for path, data := range prior {
		if err := os.WriteFile(path, data, 0o640); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dataDir, "retrieval.lock.json"), 0o755); err != nil {
		t.Fatalf("block retrieval lock: %v", err)
	}
	code := Execute([]string{"--project", project, "index", "rebuild", "--backend", "opensearch"}, ioDiscard{}, ioDiscard{})
	if code != 1 {
		t.Fatalf("index code = %d, want 1", code)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("OpenSearch received %d requests before metadata preflight failure", got)
	}
	for path, want := range prior {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved %s: %v", path, err)
		}
		if string(got) != string(want) {
			t.Fatalf("metadata changed after lock failure: %s = %q, want %q", path, got, want)
		}
	}
}

func TestExecuteIndexOpenSearchRejectsSymlinkedMetadataDirectoryBeforeRemoteRequest(t *testing.T) {
	var requests atomic.Int32
	server := newOpenSearchIndexTestServer(t, func() { requests.Add(1) })
	defer server.Close()
	t.Setenv("RFORGE_OPENSEARCH_URL", server.URL)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "OpenSearch text"}}}}})
	outsideDir := t.TempDir()
	dataDir := filepath.Join(project, "data")
	if err := os.Symlink(outsideDir, dataDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr strings.Builder
	code := Execute([]string{"--json", "--project", project, "index", "rebuild", "--backend", "opensearch"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("index code = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"index_metadata_preflight_failed"`) {
		t.Fatalf("missing metadata preflight error: %s", stdout.String())
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("OpenSearch received %d requests before metadata parent preflight failure", got)
	}
	entries, err := os.ReadDir(outsideDir)
	if err != nil {
		t.Fatalf("read outside directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("metadata written through symlinked directory: %v", entries)
	}
}

func newOpenSearchIndexTestServer(t *testing.T, onRequest ...func()) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(onRequest) > 0 {
			onRequest[0]()
		}
		switch r.URL.Path {
		case "/researchforge-passages":
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case "/researchforge-passages/_bulk":
			_, _ = w.Write([]byte(`{"errors":false,"items":[{"index":{"_id":"p1","status":201}}]}`))
		case "/researchforge-passages/_refresh":
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
