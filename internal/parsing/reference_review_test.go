package parsing

import "testing"

func TestAmbiguousReferencesFlagsLowConfidenceAndRawOnlyItems(t *testing.T) {
	doc := ParsedDocument{PaperID: "paper-1", References: []Reference{
		{Title: "Confident", DOI: "10.1000/ok", Confidence: 0.95},
		{Title: "Low confidence", Raw: "raw low", Confidence: 0.5},
		{Raw: "raw only"},
	}}

	report := AmbiguousReferences(doc, 0.75)

	if report.PaperID != "paper-1" || report.Threshold != 0.75 || len(report.Items) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.Items[0].Index != 1 || report.Items[0].Reason != "low_confidence" {
		t.Fatalf("first item = %#v", report.Items[0])
	}
	if report.Items[1].Index != 2 || report.Items[1].Reason != "missing_title_and_doi" {
		t.Fatalf("second item = %#v", report.Items[1])
	}
}
