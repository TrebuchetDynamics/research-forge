package library

import "testing"

func TestPaperRecordNormalizesIdentifiers(t *testing.T) {
	record, err := NewPaperRecord(PaperRecordInput{
		Title: "  Artificial photosynthesis catalyst review  ",
		Identifiers: Identifiers{
			DOI:               " https://doi.org/10.1000/Example ",
			ArXivID:           " arXiv:2401.00001 ",
			OpenAlexID:        " https://openalex.org/W123 ",
			CrossrefID:        " 10.1000/Example ",
			SemanticScholarID: " S2-123 ",
			PMID:              " 123456 ",
		},
	})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if record.Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Identifiers.DOI != "10.1000/example" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Identifiers.ArXivID != "2401.00001" {
		t.Fatalf("ArXivID = %q", record.Identifiers.ArXivID)
	}
	if record.Identifiers.OpenAlexID != "W123" {
		t.Fatalf("OpenAlexID = %q", record.Identifiers.OpenAlexID)
	}
}

func TestPaperRecordNormalizesDescriptiveMetadata(t *testing.T) {
	record, err := NewPaperRecord(PaperRecordInput{
		Title:       "Artificial photosynthesis catalyst review",
		Identifiers: Identifiers{DOI: "10.1000/example"},
		Authors: []Author{
			{Given: "  Ada ", Family: " Lovelace ", ORCID: " https://orcid.org/0000-0001-2345-6789 "},
		},
		Abstract:      "  Structured review of artificial photosynthesis catalysts.  ",
		Year:          2026,
		Venue:         " Journal of Reproducible Reviews ",
		Publisher:     " Open Research Press ",
		URLs:          []string{" https://example.org/paper ", ""},
		License:       " CC-BY-4.0 ",
		OpenAccess:    true,
		SourcePayload: "cache/openalex/example.json",
	})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if record.Authors[0].Given != "Ada" || record.Authors[0].Family != "Lovelace" {
		t.Fatalf("author = %#v", record.Authors[0])
	}
	if record.Authors[0].ORCID != "0000-0001-2345-6789" {
		t.Fatalf("ORCID = %q", record.Authors[0].ORCID)
	}
	if record.Abstract != "Structured review of artificial photosynthesis catalysts." {
		t.Fatalf("Abstract = %q", record.Abstract)
	}
	if record.Year != 2026 || record.Venue != "Journal of Reproducible Reviews" || record.Publisher != "Open Research Press" {
		t.Fatalf("metadata = %#v", record)
	}
	if len(record.URLs) != 1 || record.URLs[0] != "https://example.org/paper" {
		t.Fatalf("URLs = %#v", record.URLs)
	}
	if record.License != "CC-BY-4.0" || !record.OpenAccess || record.SourcePayload != "cache/openalex/example.json" {
		t.Fatalf("access/source metadata = %#v", record)
	}
}

func TestPaperRecordStoresRawSourcePayloadReferencesAndSourceProvenance(t *testing.T) {
	record, err := NewPaperRecord(PaperRecordInput{
		Title:       "Artificial photosynthesis catalyst review",
		Identifiers: Identifiers{OpenAlexID: "W123"},
		SourceRefs: []SourceRef{
			{Source: " openalex ", RawPayloadRef: " cache/openalex/w123.json ", RetrievedAt: "2026-06-08T00:00:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if len(record.SourceRefs) != 1 {
		t.Fatalf("len(SourceRefs) = %d", len(record.SourceRefs))
	}
	ref := record.SourceRefs[0]
	if ref.Source != "openalex" || ref.RawPayloadRef != "cache/openalex/w123.json" || ref.RetrievedAt != "2026-06-08T00:00:00Z" {
		t.Fatalf("SourceRef = %#v", ref)
	}
}

func TestPaperRecordRequiresTitleAndIdentifier(t *testing.T) {
	if _, err := NewPaperRecord(PaperRecordInput{Title: "   "}); err == nil {
		t.Fatalf("NewPaperRecord returned nil error for empty title")
	}
	if _, err := NewPaperRecord(PaperRecordInput{Title: "Untitled"}); err == nil {
		t.Fatalf("NewPaperRecord returned nil error for missing identifier")
	}
}
