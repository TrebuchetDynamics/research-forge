package webui

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestEvidenceExtractionGridHandlerShowsTraceAndAnalysisFields(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "evidence-grid.json"), evidence.ExtractionGrid{SchemaVersion: "1", Rows: []evidence.ExtractionGridRow{{PaperID: "paper-1", SchemaName: "outcomes", FieldName: "mean", FieldValue: "10", SupportKind: evidence.SupportPassage, SupportRef: "passage-1", ParserName: "grobid", ParserOffset: parsing.TextOffset{Start: 3, End: 10}, PDFViewURL: "/papers/paper-1/pdf#passage-1", ReviewerStatus: evidence.StatusAccepted, CorrectionHistory: []evidence.CorrectionEvent{{Reviewer: "ada", Note: "checked"}}, DownstreamAnalysisIncluded: true}}})
	rec := httptest.NewRecorder()
	newEvidenceGridHandler(func() string { return project }).ServeHTTP(rec, httptest.NewRequest("GET", "/evidence", nil))
	body := rec.Body.String()
	for _, want := range []string{"Evidence extraction grid", "source passage/table/figure/equation", "parser offset", "PDF view", "reviewer status", "correction history", "downstream analysis inclusion", "paper-1", "passage-1", "ada"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}
