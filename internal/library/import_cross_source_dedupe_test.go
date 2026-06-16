package library

import (
	"path/filepath"
	"testing"
)

func TestImportRecordsMergesCrossSourceDuplicateIdentifiersAndSourceRefs(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	summary, err := store.ImportRecords([]PaperRecord{
		{Title: "Shared DOI", Identifiers: Identifiers{DOI: "10.1000/shared", OpenAlexID: "W1"}, SourceRefs: []SourceRef{{Source: "openalex", RawPayloadRef: "openalex:/works/W1"}}},
		{Title: "Shared DOI from Crossref", Identifiers: Identifiers{DOI: "10.1000/shared", CrossrefID: "10.1000/shared"}, SourceRefs: []SourceRef{{Source: "crossref", RawPayloadRef: "crossref:/works/10.1000/shared"}}},
		{Title: "Shared DOI from Semantic Scholar", Identifiers: Identifiers{DOI: "10.1000/shared", SemanticScholarID: "S2-1"}, SourceRefs: []SourceRef{{Source: "semantic-scholar", RawPayloadRef: "s2:/paper/S2-1"}}},
	})
	if err != nil {
		t.Fatalf("ImportRecords: %v", err)
	}
	if summary.Imported != 1 || len(summary.SkippedDuplicate) != 2 {
		t.Fatalf("summary = %+v, want imported=1 skipped duplicates=2", summary)
	}
	records, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	ids := records[0].Identifiers
	if ids.OpenAlexID != "W1" || ids.CrossrefID != "10.1000/shared" || ids.SemanticScholarID != "S2-1" {
		t.Fatalf("identifiers not merged: %#v", ids)
	}
	if len(records[0].SourceRefs) != 3 {
		t.Fatalf("source refs = %#v, want three cross-source refs", records[0].SourceRefs)
	}
}
