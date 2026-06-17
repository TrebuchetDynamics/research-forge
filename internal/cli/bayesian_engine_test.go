package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestExecuteAnalysisBayesianGridEngine(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "analysis"), 0o755); err != nil {
		t.Fatal(err)
	}
	run := analysis.AnalysisRun{ID: "bayes-grid", InputRows: []analysis.InputRow{{PaperID: "p1", EffectSize: 0.2, Variance: 0.04}, {PaperID: "p2", EffectSize: 0.4, Variance: 0.09}}}
	writeJSONFixture(t, filepath.Join(project, "analysis", "bayes-grid.json"), run)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "analysis", "bayesian", "bayes-grid", "--method", "grid", "--prior-mean", "0", "--prior-variance", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "grid-bayesian-engine") {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(project, "analysis", "bayes-grid-bayesian.json")); err != nil {
		t.Fatalf("artifact missing: %v", err)
	}
}
