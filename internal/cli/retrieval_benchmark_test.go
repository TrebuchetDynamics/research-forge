package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

func TestExecuteRetrieveBenchmarkWritesDeterministicReport(t *testing.T) {
	project := t.TempDir()
	out := filepath.Join(project, "data", "retrieval-benchmark.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "retrieve", "benchmark", "--out", out, "--k", "2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report retrieval.RetrievalBenchmarkReport
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	seen := map[string]bool{}
	for _, backend := range report.Backends {
		seen[backend.Backend] = true
	}
	for _, want := range []string{"sqlite-fts", "opensearch", "qdrant", "hybrid"} {
		if !seen[want] {
			t.Fatalf("missing backend %s in %#v", want, report.Backends)
		}
	}
	if report.QuerySetChecksum == "" || report.SelectedBackend == "" || len(report.QueryResults) == 0 {
		t.Fatalf("report = %#v", report)
	}
}
