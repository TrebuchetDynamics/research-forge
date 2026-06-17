package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteLibraryIdentityResolveJSON(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Identity DOI", Identifiers: library.Identifiers{DOI: "10.1000/identity"}})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Identity Crossref", Identifiers: library.Identifiers{CrossrefID: "https://doi.org/10.1000/identity"}})
	if _, err := store.ImportRecords([]library.PaperRecord{left, right}); err != nil {
		t.Fatalf("import: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "library", "identity-resolve"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Report struct {
				SupportedIdentifiers map[string]bool `json:"supportedIdentifiers"`
				Clusters             []struct {
					Matches []struct {
						Rule string `json:"rule"`
					} `json:"matches"`
				} `json:"clusters"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || !env.Data.Report.SupportedIdentifiers["ads_bibcode"] || len(env.Data.Report.Clusters) != 1 {
		t.Fatalf("unexpected identity report: %#v", env)
	}
	matched := false
	for _, match := range env.Data.Report.Clusters[0].Matches {
		if match.Rule == "exact_doi_crossref" || match.Rule == "exact_doi" {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("identity report missing DOI/Crossref rule: %#v", env.Data.Report.Clusters[0].Matches)
	}
}
