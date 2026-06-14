package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportJSONSkipsNoIdentifierRecordsAndCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lib.json")
	if err := os.WriteFile(path, []byte(`[{"Title":"Valid","Identifiers":{"DOI":"10.1000/ok"}},{"Title":"No identifier"}]`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	records, skipped, err := ImportJSON(path)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if len(records) != 1 || records[0].Identifiers.DOI != "10.1000/ok" {
		t.Fatalf("records = %#v, want one valid record", records)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}

func TestImportJSONErrorsOnMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lib.json")
	if err := os.WriteFile(path, []byte("this is not json"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, _, err := ImportJSON(path); err == nil {
		t.Fatalf("want error for malformed JSON, got nil")
	}
}

func TestImportRecordsImportsNewRecords(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	summary, err := store.ImportRecords([]PaperRecord{
		{Title: "Artificial photosynthesis catalyst A", Identifiers: Identifiers{DOI: "10.1000/ap-a"}},
		{Title: "Artificial photosynthesis catalyst B", Identifiers: Identifiers{DOI: "10.1000/ap-b"}},
	})
	if err != nil {
		t.Fatalf("ImportRecords: %v", err)
	}
	if summary.Imported != 2 || len(summary.SkippedDuplicate) != 0 || summary.SkippedNoIdentifier != 0 {
		t.Fatalf("summary = %+v, want imported=2 no skips", summary)
	}
	records, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("store has %d records, want 2", len(records))
	}
}

func TestImportRecordsSkipsInStoreDuplicate(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(PaperRecord{Title: "Existing", Identifiers: Identifiers{DOI: "10.1000/dup"}}); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	summary, err := store.ImportRecords([]PaperRecord{
		{Title: "Incoming duplicate", Identifiers: Identifiers{DOI: "10.1000/dup"}},
		{Title: "Incoming new", Identifiers: Identifiers{DOI: "10.1000/new"}},
	})
	if err != nil {
		t.Fatalf("ImportRecords: %v", err)
	}
	if summary.Imported != 1 {
		t.Fatalf("imported = %d, want 1", summary.Imported)
	}
	if len(summary.SkippedDuplicate) != 1 || summary.SkippedDuplicate[0] != "10.1000/dup" {
		t.Fatalf("SkippedDuplicate = %v, want [10.1000/dup]", summary.SkippedDuplicate)
	}
	records, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("store has %d records, want 2 (existing + one new)", len(records))
	}
}

func TestImportRecordsSkipsInBatchDuplicate(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	summary, err := store.ImportRecords([]PaperRecord{
		{Title: "First", Identifiers: Identifiers{DOI: "10.1000/same"}},
		{Title: "Second with same DOI", Identifiers: Identifiers{DOI: "10.1000/same"}},
	})
	if err != nil {
		t.Fatalf("ImportRecords: %v", err)
	}
	if summary.Imported != 1 {
		t.Fatalf("imported = %d, want 1", summary.Imported)
	}
	if len(summary.SkippedDuplicate) != 1 || summary.SkippedDuplicate[0] != "10.1000/same" {
		t.Fatalf("SkippedDuplicate = %v, want [10.1000/same]", summary.SkippedDuplicate)
	}
}

func TestImportRecordsSkipsNoIdentifier(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	summary, err := store.ImportRecords([]PaperRecord{
		{Title: "No identifier record"},
		{Title: "Has identifier", Identifiers: Identifiers{DOI: "10.1000/ok"}},
	})
	if err != nil {
		t.Fatalf("ImportRecords: %v", err)
	}
	if summary.Imported != 1 || summary.SkippedNoIdentifier != 1 {
		t.Fatalf("summary = %+v, want imported=1 skippedNoIdentifier=1", summary)
	}
}
