package screening

import (
	"bytes"
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

func TestExportCSVDoesNotWriteThroughSymlinkedDestination(t *testing.T) {
	outsidePath := filepath.Join(t.TempDir(), "outside-screening.csv")
	outsideBefore := []byte("outside screening\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside screening: %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "screening.csv")
	if err := os.Symlink(outsidePath, exportPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	err := ExportCSV(exportPath, []DecisionEvent{{PaperID: "replacement"}})
	if err == nil {
		t.Errorf("ExportCSV succeeded through symlink")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside screening: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Errorf("ExportCSV changed outside screening:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
	}
	info, statErr := os.Stat(outsidePath)
	if statErr != nil {
		t.Fatalf("stat outside screening: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("outside screening mode = %o, want 600", got)
	}
}

func TestImportCSVRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write empty CSV: %v", err)
	}

	if _, err := ImportCSV(path); err == nil {
		t.Fatal("ImportCSV returned nil error for an empty file")
	}
}

func TestImportCSVRejectsShortRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "short.csv")
	content := "paper_id,stage\npaper-1,title_abstract\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write short CSV: %v", err)
	}

	if _, err := ImportCSV(path); err == nil {
		t.Fatal("ImportCSV returned nil error for a short row")
	}
}

func TestImportCSVRejectsUnexpectedHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wrong-header.csv")
	content := "id,phase,outcome,comment,person\npaper-1,title_abstract,include,,ada\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write CSV with unexpected header: %v", err)
	}

	if _, err := ImportCSV(path); err == nil {
		t.Fatal("ImportCSV returned nil error for an unexpected header")
	}
}
