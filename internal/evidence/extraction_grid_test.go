package evidence

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestBuildExtractionGridLinksFieldsSupportOffsetsPDFReviewerHistoryAndAnalysis(t *testing.T) {
	item := EvidenceItem{
		PaperID:    "paper-1",
		SchemaName: "outcomes",
		Values:     map[string]string{"mean_treatment": "10", "mean_control": "8"},
		Support:    Support{Kind: SupportPassage, Ref: "p1-passage-1"},
		Status:     StatusAccepted,
		History:    []CorrectionEvent{{Status: StatusCorrected, Reviewer: "ada", Note: "fixed numeric extraction"}, {Status: StatusAccepted, Reviewer: "bob", Note: "checked"}},
	}
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1-passage-1", PaperID: "paper-1", SectionID: "s1", Text: "Treatment mean was 10 and control mean was 8."}}}}})
	grid := BuildExtractionGrid(ExtractionGridInput{Items: []EvidenceItem{item}, ParsedDocuments: []parsing.ParsedDocument{doc}, AnalysisIncludedPaperIDs: []string{"paper-1"}, PDFBaseURL: "/papers"})
	if grid.SchemaVersion != "1" || len(grid.Rows) != 2 {
		t.Fatalf("grid = %#v", grid)
	}
	row := grid.Rows[0]
	if row.FieldName == "" || row.SupportKind != SupportPassage || row.SupportRef != "p1-passage-1" || row.ParserName != "grobid" || row.ParserVersion != "0.8" {
		t.Fatalf("row support/parser = %#v", row)
	}
	if row.ParserOffset.Start < 0 || row.ParserOffset.End <= row.ParserOffset.Start || row.PDFViewURL != "/papers/paper-1/pdf#p1-passage-1" {
		t.Fatalf("row offset/pdf = %#v", row)
	}
	if row.ReviewerStatus != StatusAccepted || len(row.CorrectionHistory) != 2 || !row.DownstreamAnalysisIncluded {
		t.Fatalf("row review/analysis = %#v", row)
	}
}

func TestBuildExtractionGridSupportsTableFigureEquationFallbacks(t *testing.T) {
	items := []EvidenceItem{
		{PaperID: "paper-1", SchemaName: "schema", Values: map[string]string{"table_value": "12"}, Support: Support{Kind: SupportTable, Ref: "tbl-1"}, Status: StatusSuggested},
		{PaperID: "paper-1", SchemaName: "schema", Values: map[string]string{"figure_value": "curve"}, Support: Support{Kind: SupportFigure, Ref: "fig-1"}, Status: StatusRejected},
		{PaperID: "paper-1", SchemaName: "schema", Values: map[string]string{"equation_value": "y=x"}, Support: Support{Kind: SupportEquation, Ref: "eq-1"}, Status: StatusCorrected},
	}
	grid := BuildExtractionGrid(ExtractionGridInput{Items: items})
	if len(grid.Rows) != 3 {
		t.Fatalf("rows = %#v", grid.Rows)
	}
	for _, row := range grid.Rows {
		if row.SupportRef == "" || row.ParserOffset.Start != -1 || row.ParserOffset.End != -1 || row.DownstreamAnalysisIncluded {
			t.Fatalf("fallback row = %#v", row)
		}
	}
}
