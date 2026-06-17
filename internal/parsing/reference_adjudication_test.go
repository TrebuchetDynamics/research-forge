package parsing

import (
	"path/filepath"
	"testing"
)

func TestRecordAndLoadReferenceAdjudicationsPersistsReviewerDecisions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference-adjudications.jsonl")
	doc := ParsedDocument{PaperID: "paper-1", References: []Reference{{Title: "Original", DOI: "10.1/a"}}}
	for _, decision := range []string{"accept", "correct", "reject", "defer"} {
		record, err := NewReferenceAdjudication(doc, 0, decision, "reviewer-a", "because", ReferenceCorrection{Title: "Corrected", DOI: "10.1/c"})
		if err != nil {
			t.Fatalf("NewReferenceAdjudication(%s): %v", decision, err)
		}
		if err := AppendReferenceAdjudication(path, record); err != nil {
			t.Fatalf("append %s: %v", decision, err)
		}
	}
	records, err := LoadReferenceAdjudications(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 4 || records[0].Decision != "accept" || records[1].Correction.Title != "Corrected" || records[3].Reviewer != "reviewer-a" {
		t.Fatalf("records = %#v", records)
	}
}

func TestApplyReferenceAdjudicationsAcceptCorrectRejectDefer(t *testing.T) {
	doc := ParsedDocument{PaperID: "paper-1", References: []Reference{{Title: "A"}, {Title: "B"}, {Title: "C"}, {Title: "D"}}}
	records := []ReferenceAdjudication{
		{PaperID: "paper-1", ReferenceIndex: 0, Decision: "accept"},
		{PaperID: "paper-1", ReferenceIndex: 1, Decision: "correct", Correction: ReferenceCorrection{Title: "B fixed", DOI: "10.1/b"}},
		{PaperID: "paper-1", ReferenceIndex: 2, Decision: "reject"},
		{PaperID: "paper-1", ReferenceIndex: 3, Decision: "defer"},
	}
	report := ApplyReferenceAdjudications(doc, records)
	if len(report.Items) != 4 || report.Items[1].Reference.Title != "B fixed" || report.Items[2].Status != "rejected" || report.Items[3].Status != "deferred" {
		t.Fatalf("report = %#v", report)
	}
	if report.Accepted != 1 || report.Corrected != 1 || report.Rejected != 1 || report.Deferred != 1 {
		t.Fatalf("counts = %#v", report)
	}
}
