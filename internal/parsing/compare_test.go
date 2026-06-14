package parsing

import "testing"

func TestCompareParsedDocumentsReportsDeltasAndFallbackReview(t *testing.T) {
	docs := []ParsedDocument{
		{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "1", Title: "Shared title", Sections: []Section{{ID: "s1", Passages: []Passage{{ID: "p1"}}}}, References: []Reference{{Title: "Ref 1"}}},
		{PaperID: "paper-1", ParserName: "s2orc", ParserVersion: "2", Title: "Different title", Sections: []Section{{ID: "s1", Passages: []Passage{{ID: "p1"}, {ID: "p2"}}}, {ID: "s2"}}, References: []Reference{{Title: "Ref 1"}, {Title: "Ref 2"}}, Warnings: []string{"low confidence"}},
	}

	report := CompareParsedDocuments(docs)

	if report.PaperID != "paper-1" || len(report.Documents) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.SectionDelta != 1 || report.PassageDelta != 1 || report.ReferenceDelta != 1 || report.WarningCount != 1 || !report.TitleMismatch {
		t.Fatalf("deltas = %#v", report)
	}
	if report.RecommendedUse != "review-required" {
		t.Fatalf("RecommendedUse = %q", report.RecommendedUse)
	}
	if len(report.Candidates) != 2 || report.Candidates[0].ParserName != "grobid" || report.Candidates[0].DependencyFootprint != "java-service" || report.Candidates[1].LicensePolicy != "adapter-only" {
		t.Fatalf("candidates = %#v", report.Candidates)
	}
}

func TestCompareParsedDocumentsScoresStaleFallbackCandidates(t *testing.T) {
	report := CompareParsedDocuments([]ParsedDocument{{PaperID: "paper-1", ParserName: "science-parse", References: []Reference{{Title: "Ref"}}}})

	if len(report.Candidates) != 1 || report.Candidates[0].MaintenanceRisk != "stale-reference" || report.Candidates[0].LicensePolicy != "pattern-reference" {
		t.Fatalf("candidates = %#v", report.Candidates)
	}
}

func TestCompareParsedDocumentsRecommendsConsistentParser(t *testing.T) {
	docs := []ParsedDocument{{PaperID: "paper-1", ParserName: "grobid", Title: "Title", Sections: []Section{{ID: "s1", Passages: []Passage{{ID: "p1"}}}}, References: []Reference{{Title: "Ref"}}}}

	report := CompareParsedDocuments(docs)

	if report.RecommendedUse != "grobid" || report.SectionDelta != 0 || report.PassageDelta != 0 || report.ReferenceDelta != 0 {
		t.Fatalf("report = %#v", report)
	}
}
