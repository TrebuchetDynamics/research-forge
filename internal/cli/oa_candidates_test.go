package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteOACompareCandidatesJSON(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	record, err := library.NewPaperRecord(library.PaperRecordInput{Title: "OA candidates", Identifiers: library.Identifiers{DOI: "10.1000/oa", ArXivID: "2401.12345"}, URLs: []string{"/tmp/local.pdf"}, License: "CC-BY", OpenAccess: true, SourceRefs: []library.SourceRef{{Source: "unpaywall", Metadata: map[string]string{"pdf_url": "https://example.org/u.pdf", "oa_status": "gold"}}, {Source: "doaj", Metadata: map[string]string{"full_text_url": "https://example.org/d.pdf", "license": "CC-BY"}}, {Source: "core", Metadata: map[string]string{"download_url": "https://example.org/c.pdf", "license": "CC-BY"}}, {Source: "europepmc", Metadata: map[string]string{"full_text_url": "https://example.org/e.pdf", "license": "CC-BY"}}}})
	if err != nil {
		t.Fatalf("new record: %v", err)
	}
	if _, err := store.ImportRecords([]library.PaperRecord{record}); err != nil {
		t.Fatalf("import: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "oa", "candidates"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Comparison struct {
				Candidates []struct {
					Source                   string `json:"source"`
					ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
				} `json:"candidates"`
			} `json:"comparison"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || len(env.Data.Comparison.Candidates) < 6 {
		t.Fatalf("env = %#v", env)
	}
	for _, candidate := range env.Data.Comparison.Candidates {
		if !candidate.ReviewerApprovalRequired {
			t.Fatalf("candidate not gated: %#v", candidate)
		}
	}
}
