package parsing

import "testing"

func TestArbitrateParserOutputsScoresFieldsAndRecordsDecision(t *testing.T) {
	docs := []ParsedDocument{
		{PaperID: "paper-1", ParserName: "grobid", Title: "Shared title", Abstract: "Abstract A", Sections: []Section{{ID: "s1", Title: "Methods", Passages: []Passage{{ID: "p1", Text: "Method text."}}}}, References: []Reference{{Title: "Ref"}}},
		{PaperID: "paper-1", ParserName: "s2orc", Title: "Shared title", Abstract: "", Sections: []Section{{ID: "s1", Title: "Methods", Passages: []Passage{{ID: "p1", Text: "Different method text."}}}}, Warnings: []string{"missing abstract"}},
	}
	report := ArbitrateParserOutputs(docs, ArbitrationDecisionInput{AcceptedParser: "grobid", Reason: "best abstract/reference coverage", Reviewer: "reviewer-a"})
	if report.SchemaVersion != "1" || report.PaperID != "paper-1" {
		t.Fatalf("report = %#v", report)
	}
	for _, field := range []string{"title", "abstract", "sections", "references"} {
		if _, ok := report.FieldScores[field]; !ok {
			t.Fatalf("missing field score %s: %#v", field, report.FieldScores)
		}
	}
	if len(report.Comparisons) == 0 || report.Comparisons[0].Field == "" || len(report.Comparisons[0].ParserValues) != 2 {
		t.Fatalf("comparisons missing raw values: %#v", report.Comparisons)
	}
	if report.Comparisons[0].ParserValues[0].Offset.End <= report.Comparisons[0].ParserValues[0].Offset.Start {
		t.Fatalf("comparison offset missing: %#v", report.Comparisons[0].ParserValues[0].Offset)
	}
	if len(report.WarningComparison) != 2 || report.WarningComparison[1].Warnings[0] != "missing abstract" {
		t.Fatalf("warnings missing: %#v", report.WarningComparison)
	}
	if report.Decision.AcceptedParser != "grobid" || report.Decision.Reason == "" || !report.Decision.ReviewerApprovalRequired {
		t.Fatalf("decision = %#v", report.Decision)
	}
}
