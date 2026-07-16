package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestExecuteAnalysisRunWritesArtifactManifest(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Manifest"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.MkdirAll(filepath.Join(project, "analysis"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run1.json"), analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "p1", EffectSize: 1, Variance: 0.1}}})
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "--project", project, "analysis", "run", "run1"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run code=%d stderr=%s", code, stderr.String())
	}
	var manifest analysis.AnalysisArtifactManifest
	if err := readJSONFile(filepath.Join(project, "analysis", "run1-artifact-manifest.json"), &manifest); err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest.RunID != "run1" || len(manifest.Plots) != 2 || manifest.Script.Checksum == "" || len(manifest.ReportEmbedding) != 2 {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func TestExecuteAnalysisRunReportsSensitivityFailures(t *testing.T) {
	tests := []struct {
		name      string
		blocked   string
		errorCode string
	}{
		{name: "run artifact", blocked: "run1-excl-floor-script.R", errorCode: "analysis_sensitivity_failed"},
		{name: "result artifact", blocked: "run1-excl-floor-result.json", errorCode: "analysis_sensitivity_store_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := filepath.Join(t.TempDir(), "demo")
			if code := Execute([]string{"project", "create", project, "--title", "Sensitivity"}, ioDiscard{}, ioDiscard{}); code != 0 {
				t.Fatalf("project create code=%d", code)
			}
			analysisDir := filepath.Join(project, "analysis")
			if err := os.MkdirAll(analysisDir, 0o755); err != nil {
				t.Fatalf("mkdir analysis: %v", err)
			}
			writeJSONForCLITest(t, filepath.Join(analysisDir, "run1.json"), analysis.AnalysisRun{
				SchemaVersion: "1",
				ID:            "run1",
				InputRows: []analysis.InputRow{
					{PaperID: "p1", EffectSize: 1, Variance: 0.1, ViSource: "ci"},
					{PaperID: "p2", EffectSize: 2, Variance: 0.2, ViSource: "floor"},
					{PaperID: "p3", EffectSize: 3, Variance: 0.3, ViSource: "se"},
				},
			})
			if err := os.Mkdir(filepath.Join(analysisDir, tt.blocked), 0o755); err != nil {
				t.Fatalf("block sensitivity artifact path: %v", err)
			}
			var stdout, stderr bytes.Buffer

			code := Execute([]string{"--json", "--project", project, "analysis", "run", "run1"}, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("run code=%d stdout=%s stderr=%s, want sensitivity failure", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"code":"`+tt.errorCode+`"`) {
				t.Fatalf("run output=%s, want %s", stdout.String(), tt.errorCode)
			}
		})
	}
}

func TestExecuteAnalysisRunWritesSensitivityArtifactWithOneNonFloorRow(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Sensitivity"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	analysisDir := filepath.Join(project, "analysis")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	writeJSONForCLITest(t, filepath.Join(analysisDir, "run1.json"), analysis.AnalysisRun{
		SchemaVersion: "1",
		ID:            "run1",
		InputRows: []analysis.InputRow{
			{PaperID: "p1", EffectSize: 1, Variance: 0.1, ViSource: "ci"},
			{PaperID: "p2", EffectSize: 2, Variance: 0.2, ViSource: "floor"},
		},
	})
	var stdout, stderr bytes.Buffer

	if code := Execute([]string{"--json", "--project", project, "analysis", "run", "run1"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(analysisDir, "run1-excl-floor-result.json")); err != nil {
		t.Fatalf("required sensitivity artifact missing: %v", err)
	}
}

func TestExecuteAnalysisRunRejectsAllFloorRowsBeforeWritingResults(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Sensitivity"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	analysisDir := filepath.Join(project, "analysis")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	writeJSONForCLITest(t, filepath.Join(analysisDir, "run1.json"), analysis.AnalysisRun{
		SchemaVersion: "1",
		ID:            "run1",
		InputRows: []analysis.InputRow{
			{PaperID: "p1", EffectSize: 1, Variance: 0.1, ViSource: "floor"},
			{PaperID: "p2", EffectSize: 2, Variance: 0.2, ViSource: "floor"},
		},
	})
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "analysis", "run", "run1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run code=%d stdout=%s stderr=%s, want sensitivity failure", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"analysis_sensitivity_failed"`) {
		t.Fatalf("run output=%s, want analysis_sensitivity_failed", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(analysisDir, "run1-result.json")); !os.IsNotExist(err) {
		t.Fatalf("primary result written before sensitivity preflight failure: %v", err)
	}
}

func TestExecuteAnalysisRunDoesNotWriteThroughSymlinkedArtifactManifest(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Manifest"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	analysisDir := filepath.Join(project, "analysis")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeJSONForCLITest(t, filepath.Join(analysisDir, "run1.json"), analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "p1", EffectSize: 1, Variance: 0.1}}})
	outsidePath := filepath.Join(t.TempDir(), "outside-manifest.json")
	outsideBefore := []byte("outside analysis manifest\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside manifest: %v", err)
	}
	manifestPath := filepath.Join(analysisDir, "run1-artifact-manifest.json")
	if err := os.Symlink(outsidePath, manifestPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "analysis", "run", "run1"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("run code=%d stdout=%s stderr=%s, want manifest failure", code, stdout.String(), stderr.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside manifest: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Errorf("analysis run wrote through manifest symlink:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
	}
	info, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatalf("stat outside manifest: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("outside manifest mode = %o, want 600", got)
	}
}
