package analysis

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestPrepareAnalysisRunGeneratesInputSnapshotAndEffectSize(t *testing.T) {
	items := []evidence.EvidenceItem{{PaperID: "paper-1", Status: evidence.StatusAccepted, Values: map[string]string{"mean_treatment": "10", "mean_control": "8", "sd_pooled": "2", "n_treatment": "25", "n_control": "25"}, Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "p1"}}}
	run, err := Prepare("run-1", items)
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if run.SchemaVersion != "1" || run.ID != "run-1" || len(run.InputRows) != 1 {
		t.Fatalf("run = %#v", run)
	}
	if run.InputRows[0].EffectSize != 1.0 || run.InputRows[0].Variance != 0.08 {
		t.Fatalf("input row = %#v", run.InputRows[0])
	}
}

func TestGenerateMetaforScriptAndParseHeterogeneityKnownFixture(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1.0, Variance: 0.08}}}
	script := GenerateMetaforScript(run)
	if !strings.Contains(script, "library(metafor)") || !strings.Contains(script, "rma(yi = yi, vi = vi") {
		t.Fatalf("script = %s", script)
	}
	metrics, err := ParseHeterogeneity("I2=42.5\ntau2=0.12\nQ=3.4\n")
	if err != nil {
		t.Fatalf("ParseHeterogeneity returned error: %v", err)
	}
	if metrics.I2 != 42.5 || metrics.Tau2 != 0.12 || metrics.Q != 3.4 {
		t.Fatalf("metrics = %#v", metrics)
	}
}

func TestGenerateMetaforScriptEmitsMachineReadableHeterogeneity(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1.0, Variance: 0.08}}}
	script := GenerateMetaforScript(run)
	for _, want := range []string{
		`cat("I2=", model$I2, "\n", sep="")`,
		`cat("tau2=", model$tau2, "\n", sep="")`,
		`cat("Q=", model$QE, "\n", sep="")`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q:\n%s", want, script)
		}
	}
}

func TestParseHeterogeneityRejectsMalformedRecognizedMetric(t *testing.T) {
	_, err := ParseHeterogeneity("status=ok\nI2=not-a-number\n")
	if err == nil {
		t.Fatal("ParseHeterogeneity returned nil error for a malformed I2 value")
	}
	if !strings.Contains(err.Error(), "I2") {
		t.Fatalf("ParseHeterogeneity error = %v, want I2 context", err)
	}
}

func TestParseHeterogeneityRejectsNonfiniteMetric(t *testing.T) {
	_, err := ParseHeterogeneity("I2=NaN\n")
	if err == nil {
		t.Fatal("ParseHeterogeneity returned nil error for a non-finite I2 value")
	}
}

func TestRunExternalCommandCapturesVersionsOutputsWarningsChecksumsAndArtifacts(t *testing.T) {
	dir := t.TempDir()
	runner := FakeRunner{Stdout: "I2=42.5\n", Stderr: "warning fixture", Versions: map[string]string{"R": "4.3.0", "metafor": "4.0.0"}}
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}}
	result, err := RunMetafor(dir, run, runner)
	if err != nil {
		t.Fatalf("RunMetafor returned error: %v", err)
	}
	if result.OutputChecksum == "" || result.ScriptChecksum == "" || result.Versions["R"] != "4.3.0" || len(result.Warnings) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if result.ForestPlot.Path == "" || result.ForestPlot.Checksum == "" || result.FunnelPlot.Path == "" || result.FunnelPlot.Checksum == "" || result.MetaRegression.Available != false || result.SubgroupAnalysis.Available != false || result.PublicationBias.Available != false || result.SensitivityAnalysis.Available != false {
		t.Fatalf("artifacts/scaffolds = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "run-1-script.R")); err != nil {
		t.Fatalf("missing script: %v", err)
	}
	forestData, err := os.ReadFile(result.ForestPlot.Path)
	if err != nil || !strings.Contains(string(forestData), "Forest plot") || !strings.Contains(string(forestData), "paper-1") {
		t.Fatalf("forest artifact err=%v data=%s", err, forestData)
	}
	funnelData, err := os.ReadFile(result.FunnelPlot.Path)
	if err != nil || !strings.Contains(string(funnelData), "Funnel plot") || !strings.Contains(string(funnelData), "paper-1") {
		t.Fatalf("funnel artifact err=%v data=%s", err, funnelData)
	}
}

func TestRunMetaforRejectsMalformedHeterogeneityOutput(t *testing.T) {
	runner := FakeRunner{Stdout: "I2=not-a-number\n", Versions: map[string]string{"R": "4.3.0", "metafor": "4.0.0"}}
	run := AnalysisRun{ID: "run-invalid-output", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}}

	_, err := RunMetafor(t.TempDir(), run, runner)
	if err == nil {
		t.Fatal("RunMetafor returned nil error for malformed heterogeneity output")
	}
	if !strings.Contains(err.Error(), "I2") {
		t.Fatalf("RunMetafor error = %v, want I2 context", err)
	}
}

func TestRunMetaforRejectsNonfiniteRowsBeforeWriting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "analysis")
	run := AnalysisRun{ID: "run-invalid-row", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: math.NaN()}}}
	runner := FakeRunner{Stdout: "I2=0\ntau2=0\nQ=0\n"}

	if _, err := RunMetafor(dir, run, runner); err == nil {
		t.Fatal("RunMetafor returned nil error for a non-finite variance")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("RunMetafor created analysis directory for an invalid row: %v", err)
	}
}

func TestRunMetaforRejectsEmptyRunBeforeWriting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "analysis")
	run := AnalysisRun{ID: "run-empty"}
	runner := FakeRunner{Stdout: "I2=0\ntau2=0\nQ=0\n"}

	if _, err := RunMetafor(dir, run, runner); err == nil {
		t.Fatal("RunMetafor returned nil error for an empty analysis run")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("RunMetafor created analysis directory for an empty run: %v", err)
	}
}

func TestRunMetaforRejectsNilRunnerBeforeWriting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "analysis")
	run := AnalysisRun{ID: "run-no-runner", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}}}

	if _, err := RunMetafor(dir, run, nil); err == nil {
		t.Fatal("RunMetafor returned nil error for a nil runner")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("RunMetafor created analysis directory without a runner: %v", err)
	}
}

func TestRunMetaforRejectsTraversalRunIDBeforeWriting(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "analysis")
	outsidePath := filepath.Join(root, "outside-script.R")
	outsideBefore := []byte("outside analysis artifact\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside artifact: %v", err)
	}
	run := AnalysisRun{ID: "../outside", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}}
	runner := FakeRunner{Stdout: "I2=0\ntau2=0\nQ=0\n"}

	if _, err := RunMetafor(dir, run, runner); err == nil {
		t.Fatal("RunMetafor accepted a traversal run ID")
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside artifact: %v", err)
	}
	if string(outsideAfter) != string(outsideBefore) {
		t.Fatalf("outside artifact changed:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
	}
}

func TestRunMetaforDoesNotWriteThroughSymlinkedArtifacts(t *testing.T) {
	for _, suffix := range []string{"-script.R", "-output.txt", "-forest.svg", "-funnel.svg"} {
		t.Run(suffix, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "analysis")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("create analysis directory: %v", err)
			}
			outsidePath := filepath.Join(root, "outside")
			outsideBefore := []byte("outside analysis artifact\n")
			if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
				t.Fatalf("write outside artifact: %v", err)
			}
			artifactPath := filepath.Join(dir, "run-safe"+suffix)
			if err := os.Symlink(outsidePath, artifactPath); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			run := AnalysisRun{ID: "run-safe", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}}
			runner := FakeRunner{Stdout: "I2=0\ntau2=0\nQ=0\n"}

			if _, err := RunMetafor(dir, run, runner); err == nil {
				t.Fatalf("RunMetafor accepted symlinked artifact %s", artifactPath)
			}
			outsideAfter, err := os.ReadFile(outsidePath)
			if err != nil {
				t.Fatalf("read outside artifact: %v", err)
			}
			if string(outsideAfter) != string(outsideBefore) {
				t.Fatalf("outside artifact changed through %s:\n got: %s\nwant: %s", artifactPath, outsideAfter, outsideBefore)
			}
		})
	}
}
