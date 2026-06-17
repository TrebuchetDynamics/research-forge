package library

import "testing"

func TestReferenceManagerRoundTripMatrixReportsPerFieldLoss(t *testing.T) {
	record := PaperRecord{Title: "Roundtrip", Identifiers: Identifiers{DOI: "10.1000/rt"}, Year: 2026, Venue: "Journal", SourceRefs: []SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"citation_key": "rt2026", "tags": "tag1; tag2", "note": "note", "collections": "Reviews/Sub", "collection_hierarchy": "Reviews/Sub", "annotations": "highlight", "attachment_files": "paper.pdf", "linked_file_privacy_check": "redacted-local-paths"}}}}
	report := BuildReferenceManagerRoundTripMatrix([]PaperRecord{record})
	for _, format := range []string{"bibtex", "csl-json", "zotero-rdf"} {
		if !report.HasFormat(format) {
			t.Fatalf("missing format %s", format)
		}
	}
	zotero, _ := report.Format("zotero-rdf")
	for _, field := range []string{"better_bibtex_citation_key", "tags", "notes", "collections", "annotations", "redacted_attachments"} {
		if zotero.Fields[field].Status == FidelityUnsupported || zotero.Fields[field].Lost != 0 {
			t.Fatalf("zotero field %s = %#v", field, zotero.Fields[field])
		}
	}
	csl, _ := report.Format("csl-json")
	if csl.Fields["annotations"].Lost == 0 || csl.Fields["collections"].Lost == 0 {
		t.Fatalf("expected csl loss report: %#v", csl.Fields)
	}
	bibtex, _ := report.Format("bibtex")
	if bibtex.Fields["better_bibtex_citation_key"].Preserved != 1 || bibtex.Fields["redacted_attachments"].Status != FidelityRedacted {
		t.Fatalf("bibtex fields = %#v", bibtex.Fields)
	}
}
