package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestExecuteAnalysisMethodWorkbenchWritesComparison(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Method Workbench"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	out := filepath.Join(project, "analysis", "method-workbench.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "analysis", "method-workbench", "run1", "--category", "effect-size models", "--method", "smd", "--method", "risk-difference", "--select", "risk-difference", "--reviewer", "ada", "--reason", "absolute effect needed for report", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("method-workbench code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report analysis.MethodComparisonReport
	if err := readJSONFile(out, &report); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if report.Category != "effect-size models" || len(report.Options) != 2 || !report.RequiresReviewerChoice || report.LockedSelection.Method != "risk-difference" || !report.LockedSelection.LockedIntoFinalReport {
		t.Fatalf("report = %#v", report)
	}
}
