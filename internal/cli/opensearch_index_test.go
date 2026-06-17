package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

func TestExecuteIndexOpenSearchWritesMappingLockAndBulkReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
