package library

import "testing"

func TestReferenceManagerInterchangeMatrixCoversFormatsAndFields(t *testing.T) {
	matrix := DefaultReferenceManagerInterchangeMatrix()
	for _, format := range []string{"bibtex", "ris", "csl-json", "zotero-rdf"} {
		row, ok := matrix.Format(format)
		if !ok {
			t.Fatalf("missing format %s in %#v", format, matrix.Formats)
		}
		for _, field := range []string{"better_bibtex_citation_key", "tags", "notes", "collections", "redacted_attachments"} {
			if _, ok := row.Fields[field]; !ok {
				t.Fatalf("format %s missing field %s: %#v", format, field, row.Fields)
			}
		}
	}
	bibtex, _ := matrix.Format("bibtex")
	if bibtex.Fields["better_bibtex_citation_key"].Status != FidelitySupported || bibtex.Fields["redacted_attachments"].Status != FidelityRedacted {
		t.Fatalf("bibtex key/attachment support wrong: %#v", bibtex.Fields)
	}
	zotero, _ := matrix.Format("zotero-rdf")
	if zotero.Fields["collections"].Status != FidelitySupported || zotero.Fields["notes"].Status != FidelitySupported {
		t.Fatalf("zotero rdf support wrong: %#v", zotero.Fields)
	}
	ris, _ := matrix.Format("ris")
	if ris.Fields["better_bibtex_citation_key"].Status != FidelityUnsupported {
		t.Fatalf("ris should document citation-key loss: %#v", ris.Fields["better_bibtex_citation_key"])
	}
}

func TestReferenceManagerInterchangeMatrixEvaluatesRecords(t *testing.T) {
	record := PaperRecord{Title: "Matrix fixture", Identifiers: Identifiers{DOI: "10.1000/matrix"}, SourceRefs: []SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"collections": "Reviews", "tags": "tag", "note": "note", "citation_key": "key", "attachment_files": "paper.pdf", "linked_file_privacy_check": "redacted-local-paths"}}}}
	matrix := BuildReferenceManagerInterchangeMatrix([]PaperRecord{record})
	if matrix.RecordCount != 1 {
		t.Fatalf("record count = %d", matrix.RecordCount)
	}
	if matrix.FieldsPresent["collections"] != 1 || matrix.FieldsPresent["redacted_attachments"] != 1 || matrix.FieldsPresent["better_bibtex_citation_key"] != 1 {
		t.Fatalf("field coverage wrong: %#v", matrix.FieldsPresent)
	}
}
