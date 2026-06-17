package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestExecuteAnalysisEngineCompareWritesPyMAREStyleReport(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Engine Compare"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.MkdirAll(filepath.Join(project, "analysis"), 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 3, Variance: 1}}}
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run1.json"), run)
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run1-result.json"), analysis.AnalysisResult{Versions: map[string]string{"R": "fixture", "metafor": "fixture"}, Warnings: []string{"metafor warning"}})
	out := filepath.Join(project, "analysis", "run1-engine-comparison.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "analysis", "engine-compare", "run1", "--out", out, "--secondary-delta", "0.2", "--tolerance", "0.1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("engine-compare code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report analysis.EngineComparisonReport
	if err := readJSONFile(out, &report); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if report.PrimaryEngine != "metafor" || report.SecondaryEngine != "pymare-fixture" || !report.Disagreement.RequiresReview || len(report.EnvironmentLocks) != 2 || !report.ModelSettingParity || len(report.Warnings) == 0 || report.OutputDeltas.EstimateDelta == 0 {
		t.Fatalf("report = %#v", report)
	}
}
