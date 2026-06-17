package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestExecuteAnalysisInfluenceAndBeggPublicationBias(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Diagnostics"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.MkdirAll(filepath.Join(project, "analysis"), 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "p1", EffectSize: 0.1, Variance: 0.01}, {PaperID: "p2", EffectSize: 0.2, Variance: 0.04}, {PaperID: "p3", EffectSize: 0.4, Variance: 0.09}, {PaperID: "p4", EffectSize: 0.8, Variance: 0.16}}}
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run1.json"), run)
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "--project", project, "analysis", "influence", "run1"}, &stdout, &stderr); code != 0 {
		t.Fatalf("influence code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "absoluteDelta") {
		t.Fatalf("influence output=%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", project, "analysis", "publication-bias", "run1", "--method", "begg"}, &stdout, &stderr); code != 0 {
		t.Fatalf("begg code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "begg-rank-correlation") {
		t.Fatalf("begg output=%s", stdout.String())
	}
}
