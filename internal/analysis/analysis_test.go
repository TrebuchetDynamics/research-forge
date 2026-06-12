package analysis

import (
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
	if result.ForestPlot.Path == "" || result.FunnelPlot.Path == "" || result.MetaRegression.Available != false || result.SubgroupAnalysis.Available != false || result.PublicationBias.Available != false || result.SensitivityAnalysis.Available != false {
		t.Fatalf("artifacts/scaffolds = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "run-1-script.R")); err != nil {
		t.Fatalf("missing script: %v", err)
	}
}
