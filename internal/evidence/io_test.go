package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceExportsCSVJSONAndMarkdown(t *testing.T) {
	item := EvidenceItem{PaperID: "paper-1", SchemaName: "schema", Values: map[string]string{"catalyst": "TiO2"}, Support: Support{Kind: SupportPassage, Ref: "p1"}, Status: StatusAccepted}
	dir := t.TempDir()
	if err := ExportCSV(filepath.Join(dir, "evidence.csv"), []EvidenceItem{item}); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if err := ExportJSON(filepath.Join(dir, "evidence.json"), []EvidenceItem{item}); err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	if err := ExportMarkdown(filepath.Join(dir, "evidence.md"), []EvidenceItem{item}); err != nil {
		t.Fatalf("ExportMarkdown: %v", err)
	}
	csvData, _ := os.ReadFile(filepath.Join(dir, "evidence.csv"))
	jsonData, _ := os.ReadFile(filepath.Join(dir, "evidence.json"))
	mdData, _ := os.ReadFile(filepath.Join(dir, "evidence.md"))
	if !strings.Contains(string(csvData), "paper-1,schema,accepted,passage,p1") || !strings.Contains(string(jsonData), "TiO2") || !strings.Contains(string(mdData), "| paper-1 | schema | accepted | passage:p1 |") {
		t.Fatalf("exports:\n%s\n%s\n%s", csvData, jsonData, mdData)
	}
}
