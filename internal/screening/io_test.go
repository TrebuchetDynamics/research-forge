package screening

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCSVScreeningExportImportAndActiveLearningScaffold(t *testing.T) {
	workflow, _ := Configure(Options{ExclusionReasons: []string{"wrong population"}})
	store := NewMemoryStore(workflow)
	_ = store.Decide(DecisionInput{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"})
	path := filepath.Join(t.TempDir(), "screening.csv")
	if err := ExportCSV(path, store.History("paper-1")); err != nil {
		t.Fatalf("ExportCSV returned error: %v", err)
	}
	imported, err := ImportCSV(path)
	if err != nil {
		t.Fatalf("ImportCSV returned error: %v", err)
	}
	if len(imported) != 1 || imported[0].PaperID != "paper-1" || imported[0].Reviewer != "ada" {
		t.Fatalf("imported = %#v", imported)
	}
	data, _ := os.ReadFile(path)
	want := "paper_id,stage,decision,reason,reviewer\npaper-1,title_abstract,include,,ada\n"
	if string(data) != want {
		t.Fatalf("csv = %s", data)
	}
	prioritized := PrioritizeActiveLearning([]string{"paper-b", "paper-a"})
	if len(prioritized) != 2 || prioritized[0] != "paper-a" {
		t.Fatalf("prioritized = %#v", prioritized)
	}
}
