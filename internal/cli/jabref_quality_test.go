package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteLibraryJabRefQualityJSON(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	record := library.PaperRecord{Title: "JabRef CLI", Identifiers: library.Identifiers{DOI: "10.1000/jabref"}, SourceRefs: []library.SourceRef{{Source: "bibtex", Metadata: map[string]string{"citation_key": "dup", "groups": "Included", "cleanup_diff": "doi normalized", "linked_file_privacy_check": "redacted", "normalization_status": "reviewer-approved"}}}}
	other := library.PaperRecord{Title: "JabRef duplicate", Identifiers: library.Identifiers{DOI: "10.1000/jabref2"}, SourceRefs: []library.SourceRef{{Source: "bibtex", Metadata: map[string]string{"citation_key": "dup"}}}}
	if _, err := store.ImportRecords([]library.PaperRecord{record, other}); err != nil {
		t.Fatalf("import record: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "library", "jabref-quality"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Report library.JabRefQualityReport `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || env.Data.Report.RecordCount != 2 || len(env.Data.Report.CitationKeyCollisions) != 1 || len(env.Data.Report.FieldCleanupDiffs) != 1 {
		t.Fatalf("report = %#v", env.Data.Report)
	}
}
