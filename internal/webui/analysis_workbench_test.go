package webui

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestAnalysisWorkbenchHandlerShowsInputsMetaforDiagnosticsAndManifests(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "analysis", "run1.json"), analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "paper-1", EffectSize: 0.4, Variance: 0.1}}})
	writeJSON(t, filepath.Join(project, "analysis", "run1-artifact-manifest.json"), analysis.AnalysisArtifactManifest{SchemaVersion: "1", RunID: "run1", Script: analysis.ScriptArtifactManifest{Path: "analysis/run1-script.R", Engine: "R/metafor"}, Warnings: []string{"metafor warning"}, Plots: []analysis.PlotArtifactManifest{{Kind: "forest", Path: "analysis/run1-forest.svg"}, {Kind: "funnel", Path: "analysis/run1-funnel.svg"}}})
	rec := httptest.NewRecorder()
	newAnalysisWorkbenchHandler(func() string { return project }).ServeHTTP(rec, httptest.NewRequest("GET", "/analysis", nil))
	body := rec.Body.String()
	for _, want := range []string{"meta-analysis Workbench", "prepared effect-size inputs", "model choices", "metafor script", "warnings", "heterogeneity", "sensitivity/influence diagnostics", "forest", "funnel", "publication-ready artifact manifests", "paper-1", "R/metafor"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestAnalysisWorkbenchHandlerDoesNotReadSymlinkedRun(t *testing.T) {
	projectPath := t.TempDir()
	analysisDir := filepath.Join(projectPath, "analysis")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		t.Fatalf("mkdir project analysis: %v", err)
	}
	externalPath := filepath.Join(t.TempDir(), "external-run.json")
	writeJSON(t, externalPath, analysis.AnalysisRun{SchemaVersion: "1", ID: "external-private-run", InputRows: []analysis.InputRow{{
		PaperID: "external-private-paper", EffectSize: 1.25, Variance: 0.2,
	}}})
	runPath := filepath.Join(analysisDir, "external-run.json")
	if err := os.Symlink(externalPath, runPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	rec := httptest.NewRecorder()
	newAnalysisWorkbenchHandler(func() string { return projectPath }).ServeHTTP(rec, httptest.NewRequest("GET", "/analysis", nil))
	body := rec.Body.String()
	for _, private := range []string{"external-private-run", "external-private-paper"} {
		if strings.Contains(body, private) {
			t.Fatalf("analysis workbench disclosed %q from symlinked run: %s", private, body)
		}
	}
	if info, err := os.Lstat(runPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("analysis run symlink changed: info=%v err=%v", info, err)
	}
}
