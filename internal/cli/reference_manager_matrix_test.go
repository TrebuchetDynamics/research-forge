package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteLibraryReferenceManagerMatrixJSON(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	record, err := library.NewPaperRecord(library.PaperRecordInput{Title: "Matrix CLI fixture", Identifiers: library.Identifiers{DOI: "10.1000/matrix-cli"}, SourceRefs: []library.SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"collections": "Reviews", "tags": "tag", "note": "note", "citation_key": "key", "attachment_files": "paper.pdf", "linked_file_privacy_check": "redacted-local-paths"}}}})
	if err != nil {
		t.Fatalf("new record: %v", err)
	}
	if _, err := store.ImportRecords([]library.PaperRecord{record}); err != nil {
		t.Fatalf("import record: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "library", "reference-manager-matrix"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Matrix struct {
				RecordCount   int            `json:"recordCount"`
				FieldsPresent map[string]int `json:"fieldsPresent"`
				Formats       []struct {
					Format string `json:"format"`
					Fields map[string]struct {
						Preserved int `json:"preserved"`
						Lost      int `json:"lost"`
					} `json:"fields"`
				} `json:"formats"`
			} `json:"matrix"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || env.Data.Matrix.RecordCount != 1 || env.Data.Matrix.FieldsPresent["better_bibtex_citation_key"] != 1 || len(env.Data.Matrix.Formats) < 4 {
		t.Fatalf("unexpected matrix: %#v", env)
	}
	if env.Data.Matrix.Formats[0].Fields == nil {
		t.Fatalf("round-trip field loss report missing: %#v", env.Data.Matrix.Formats)
	}
}
