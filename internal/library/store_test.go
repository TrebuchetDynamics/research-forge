package library

import (
	"path/filepath"
	"testing"
)

func TestStoreCreateUpdateListAndSearchPaperRecords(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	record, err := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis catalyst review", Identifiers: Identifiers{DOI: "10.1000/example"}, Year: 2026})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	record.Abstract = "Updated abstract"
	if err := store.Update(record); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].Abstract != "Updated abstract" {
		t.Fatalf("items = %#v", items)
	}
	results, err := store.Search("catalyst")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].Identifiers.DOI != "10.1000/example" {
		t.Fatalf("results = %#v", results)
	}
}
