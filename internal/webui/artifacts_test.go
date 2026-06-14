package webui

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

func TestArtifactsHandlerRendersCLIGeneratedOutputs(t *testing.T) {
	state := ArtifactDashboardState{
		Papers:        ui.NewLibraryViewModel([]ui.PaperRow{{Title: "Catalyst study"}}),
		Analysis:      ui.NewAnalysisViewModel("run-42", true),
		CitationGraph: ui.NewCitationGraphViewModel([]ui.GraphNode{{ID: "paper-a"}, {ID: "paper-b"}}, []ui.GraphEdge{{Source: "paper-a", Target: "paper-b"}}),
		PRISMA:        PRISMAFlowState{Records: 12, Screened: 8, Included: 3},
		Reports:       ui.NewReportViewModel([]string{"markdown", "html"}),
	}

	req := httptest.NewRequest("GET", "/artifacts", nil)
	rec := httptest.NewRecorder()

	NewArtifactsHandler(state).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"CLI-generated artifacts",
		"hx-get=\"/artifacts/refresh\"",
		"Papers",
		"Catalyst study",
		"Meta-analysis outputs",
		"run-42",
		"Ready",
		"PRISMA diagram",
		"Records: 12",
		"Screened: 8",
		"Included: 3",
		"Citation graph",
		"Citation graph visualization",
		"<svg",
		"paper-a",
		"paper-a → paper-b",
		"Report artifacts",
		"markdown",
		"html",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("artifacts screen missing %q:\n%s", want, body)
		}
	}
}

func TestArtifactsHandlerRendersEmptyArtifactState(t *testing.T) {
	req := httptest.NewRequest("GET", "/artifacts", nil)
	rec := httptest.NewRecorder()

	NewArtifactsHandler(ArtifactDashboardState{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{"No papers exported yet", "No analysis run ready", "No citation graph exported yet", "No report formats exported yet"} {
		if !strings.Contains(body, want) {
			t.Fatalf("empty artifact state missing %q:\n%s", want, body)
		}
	}
}
