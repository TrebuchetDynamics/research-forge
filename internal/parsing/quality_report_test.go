package parsing

import "testing"

func TestBuildParserQualityReportComparesExpectedParsersWithoutAutoAccept(t *testing.T) {
	docs := []ParsedDocument{
		{PaperID: "p1", ParserName: "grobid", ParserVersion: "1", Title: "Same", Sections: []Section{{Passages: []Passage{{ID: "g1"}}}}, References: []Reference{{Title: "Ref", DOI: "10.1/ref", Confidence: 0.9}}, ParserConfidence: 0.95},
		{PaperID: "p1", ParserName: "s2orc-doc2json", ParserVersion: "2", Title: "Different", Sections: []Section{{Passages: []Passage{{ID: "s1"}, {ID: "s2"}}}}, References: []Reference{{Title: "Ref raw", Confidence: 0.4}}, Warnings: []string{"low confidence"}, ParserConfidence: 0.55},
		{PaperID: "p1", ParserName: "papermage", ParserVersion: "3", Title: "Same", Sections: []Section{{Passages: []Passage{{ID: "p1"}}}}, ParserConfidence: 0.8},
		{PaperID: "p1", ParserName: "cermine", ParserVersion: "4", Title: "Same", ParserConfidence: 0.6},
		{PaperID: "p1", ParserName: "science-parse", ParserVersion: "5", Title: "Same", ParserConfidence: 0.5},
		{PaperID: "p1", ParserName: "anystyle", ParserVersion: "6", References: []Reference{{Title: "Ref", DOI: "10.1/ref", Confidence: 0.95}}, ParserConfidence: 0.7},
	}
	report := BuildParserQualityReport(docs)
	for _, parser := range []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"} {
		if !report.HasParser(parser) {
			t.Fatalf("missing parser %s in %#v", parser, report.ParserRuns)
		}
	}
	if !report.ReviewerRequired || report.AutoAcceptedFields || len(report.Conflicts) == 0 {
		t.Fatalf("gate/conflicts wrong: %#v", report)
	}
	if report.Conflicts[0].Status != "review-required" {
		t.Fatalf("conflict status = %#v", report.Conflicts)
	}
	if report.FieldConfidence["references"] <= 0 || report.FieldConfidence["title"] <= 0 {
		t.Fatalf("confidence = %#v", report.FieldConfidence)
	}
}
