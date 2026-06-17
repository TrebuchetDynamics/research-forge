package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalysisArtifactManifestCapturesPublicationReadyMetadata(t *testing.T) {
	dir := t.TempDir()
	run := AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []InputRow{{PaperID: "p1", EffectSize: 0.5, Variance: 0.1}}}
	result, err := RunMetafor(dir, run, FakeRunner{Stdout: "I2=0\ntau2=0\nQ=1\n", Stderr: "warning", Versions: map[string]string{"R": "4.3", "metafor": "4.0"}})
	if err != nil {
		t.Fatalf("RunMetafor: %v", err)
	}
	manifest := NewAnalysisArtifactManifest(run, result)
	if manifest.SchemaVersion != "1" || manifest.RunID != "run1" || len(manifest.Plots) != 2 || manifest.Script.Path == "" || manifest.Script.Checksum == "" || manifest.EngineVersions["R"] != "4.3" || len(manifest.Warnings) != 1 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if manifest.Plots[0].Settings["format"] != "svg" || manifest.ReportEmbedding[0].Markdown == "" {
		t.Fatalf("plot/embedding metadata = %#v %#v", manifest.Plots, manifest.ReportEmbedding)
	}
	path := filepath.Join(dir, "run1-analysis-artifacts.json")
	if err := WriteAnalysisArtifactManifest(path, manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing manifest: %v", err)
	}
}
