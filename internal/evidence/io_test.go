package evidence

import (
	"bytes"
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

func TestEvidenceExportsDoNotWriteThroughSymlinkedDestinations(t *testing.T) {
	exporters := []struct {
		name   string
		export func(string, []EvidenceItem) error
	}{
		{name: "csv", export: ExportCSV},
		{name: "json", export: ExportJSON},
		{name: "markdown", export: ExportMarkdown},
	}
	for _, tc := range exporters {
		t.Run(tc.name, func(t *testing.T) {
			outsidePath := filepath.Join(t.TempDir(), "outside-export")
			outsideBefore := []byte("outside evidence\n")
			if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
				t.Fatalf("write outside evidence: %v", err)
			}
			exportPath := filepath.Join(t.TempDir(), "evidence."+tc.name)
			if err := os.Symlink(outsidePath, exportPath); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}

			err := tc.export(exportPath, []EvidenceItem{{PaperID: "replacement"}})
			if err == nil {
				t.Errorf("export succeeded through symlink, want error")
			}
			outsideAfter, readErr := os.ReadFile(outsidePath)
			if readErr != nil {
				t.Fatalf("read outside evidence: %v", readErr)
			}
			if !bytes.Equal(outsideAfter, outsideBefore) {
				t.Errorf("export wrote through symlink:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
			}
			info, statErr := os.Stat(outsidePath)
			if statErr != nil {
				t.Fatalf("stat outside evidence: %v", statErr)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Errorf("outside evidence mode = %o, want 600", got)
			}
		})
	}
}
