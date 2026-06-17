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

func TestExportReferenceAmbiguityQueueIncludesDeferredUnreviewedAndMatchProvenance(t *testing.T) {
	doc := ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", References: []Reference{{Title: "Accepted"}, {Title: "Deferred", Raw: "raw deferred"}, {Title: "Unreviewed"}}}
	matches := []ReferenceMatch{{Index: 1, Source: "crossref", SourceID: "cr-1", Ambiguous: true, AmbiguityReason: "multiple_candidates", ResponseRawRef: "crossref:req"}}
	decisions := []ReferenceAdjudication{{PaperID: "paper-1", ReferenceIndex: 0, Decision: "accept", ProvenanceRef: "data/provenance.jsonl#evt-1"}, {PaperID: "paper-1", ReferenceIndex: 1, Decision: "defer", ProvenanceRef: "data/provenance.jsonl#evt-2"}}
	queue := ExportReferenceAmbiguityQueue(doc, matches, decisions)
	if queue.PaperID != "paper-1" || len(queue.Items) != 2 {
		t.Fatalf("queue = %#v", queue)
	}
	if queue.Items[0].Status != "deferred" || queue.Items[0].Source != "crossref" || queue.Items[0].ProvenanceRef != "data/provenance.jsonl#evt-2" {
		t.Fatalf("deferred item = %#v", queue.Items[0])
	}
	if queue.Items[1].Status != "unreviewed" || queue.Items[1].ParserName != "grobid" {
		t.Fatalf("unreviewed item = %#v", queue.Items[1])
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
