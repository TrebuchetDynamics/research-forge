package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteOAPrivacyReviewAndApprove(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	record, err := library.NewPaperRecord(library.PaperRecordInput{Title: "Private import", Identifiers: library.Identifiers{DOI: "10.1000/private"}, SourceRefs: []library.SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"attachment_files": "paper.pdf", "note": "private", "annotations": "highlight", "linked_file_privacy_check": "redacted-local-paths"}}}})
	if err != nil {
		t.Fatalf("new record: %v", err)
	}
	if _, err := store.ImportRecords([]library.PaperRecord{record}); err != nil {
		t.Fatalf("import: %v", err)
	}
	reportPath := filepath.Join(project, "report.md")
	if err := os.WriteFile(reportPath, []byte("leaks /home/alice/private.pdf private note"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "oa", "privacy-review", "--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("review code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	path := filepath.Join(project, "data", "privacy-licensing-review.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("review not written: %v", err)
	}
	var review documents.PrivacyLicensingReview
	if err := json.Unmarshal(data, &review); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if len(review.Issues) < 4 || !review.Blocked || review.Approved {
		t.Fatalf("review = %#v", review)
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "oa", "privacy-approve", "--reviewer", "reviewer-a", "--reason", "redacted"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("approve code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, _ = os.ReadFile(path)
	if err := json.Unmarshal(data, &review); err != nil {
		t.Fatalf("decode approved: %v", err)
	}
	if !review.Approved || review.Blocked || review.Reviewer != "reviewer-a" {
		t.Fatalf("approved review = %#v", review)
	}
}
