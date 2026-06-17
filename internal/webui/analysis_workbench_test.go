package webui

import (
	"net/http/httptest"
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
