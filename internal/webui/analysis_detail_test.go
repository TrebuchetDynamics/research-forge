package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func writeAnalysisResult(t *testing.T, projectPath, runID string, result analysis.AnalysisResult) {
	t.Helper()
	dir := filepath.Join(projectPath, "analysis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, runID+"-result.json"), data, 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}
}

func TestBuildAnalysisDetailReadsHeterogeneityAndPlots(t *testing.T) {
	dir := t.TempDir()
	writeAnalysisResult(t, dir, "smd", analysis.AnalysisResult{
		Metrics:    analysis.HeterogeneityMetrics{I2: 42.5, Tau2: 0.12, Q: 8.4},
		ForestPlot: analysis.Artifact{Path: "analysis/smd-forest.svg"},
		FunnelPlot: analysis.Artifact{Path: "analysis/smd-funnel.svg"},
		Warnings:   []string{"few studies"},
	})

	state, err := BuildArtifactDashboardState(dir)
	if err != nil {
		t.Fatalf("BuildArtifactDashboardState: %v", err)
	}
	d := state.AnalysisDetail
	if !d.Ready || d.RunID != "smd" {
		t.Fatalf("analysis detail = %+v", d)
	}
	if d.I2 != 42.5 || d.Tau2 != 0.12 || d.Q != 8.4 {
		t.Fatalf("heterogeneity metrics = %+v", d)
	}
	if !d.HasForestPlot || !d.HasFunnelPlot {
		t.Fatalf("expected forest+funnel plots present: %+v", d)
	}
	if len(d.Warnings) != 1 || d.Warnings[0] != "few studies" {
		t.Fatalf("warnings = %v", d.Warnings)
	}
}

func TestArtifactsPageRendersAnalysisDetail(t *testing.T) {
	dir := t.TempDir()
	writeAnalysisResult(t, dir, "smd", analysis.AnalysisResult{
		Metrics:    analysis.HeterogeneityMetrics{I2: 42.5, Tau2: 0.12, Q: 8.4},
		ForestPlot: analysis.Artifact{Path: "analysis/smd-forest.svg"},
	})
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/artifacts")
	if status != http.StatusOK {
		t.Fatalf("GET /artifacts status = %d", status)
	}
	for _, want := range []string{"42.5", "Forest plot", "Heterogeneity"} {
		if !strings.Contains(body, want) {
			t.Fatalf("/artifacts missing %q: %s", want, body)
		}
	}
}

func TestBuildAnalysisDetailAbsentIsNotReady(t *testing.T) {
	state, err := BuildArtifactDashboardState(t.TempDir())
	if err != nil {
		t.Fatalf("BuildArtifactDashboardState: %v", err)
	}
	if state.AnalysisDetail.Ready {
		t.Fatalf("expected analysis detail not ready, got %+v", state.AnalysisDetail)
	}
}
