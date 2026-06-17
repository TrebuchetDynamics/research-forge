package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteSemanticScholarExpandWritesRunState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"R1","title":"Ref"}}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	project := t.TempDir()
	out := filepath.Join(project, "graph.json")
	runPath := filepath.Join(project, "semantic-run.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "expand", "--source", "semantic-scholar", "--paper", "S1", "--direction", "references", "--out", out, "--run-state", runPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatalf("read run: %v", err)
	}
	var run struct {
		SeedID    string `json:"seedId"`
		EdgeCount int    `json:"edgeCount"`
		Completed bool   `json:"completed"`
	}
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("json: %v", err)
	}
	if run.SeedID != "S1" || run.EdgeCount != 1 || !run.Completed {
		t.Fatalf("run = %#v", run)
	}
}
