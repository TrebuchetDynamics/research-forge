package cli

import (
	"bytes"
	"os"
	"path/filepath"
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
