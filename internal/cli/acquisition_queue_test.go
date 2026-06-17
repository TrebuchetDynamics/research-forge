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

func TestExecuteOAAcquisitionQueueAndApprove(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	record, err := library.NewPaperRecord(library.PaperRecordInput{Title: "Acquisition fixture", Identifiers: library.Identifiers{DOI: "10.1000/acq"}, License: "CC-BY", OpenAccess: true, SourceRefs: []library.SourceRef{{Source: "unpaywall", Metadata: map[string]string{"pdf_url": "https://example.org/acq.pdf", "license": "CC-BY", "oa_status": "gold"}}}})
	if err != nil {
		t.Fatalf("new record: %v", err)
	}
	if _, err := store.ImportRecords([]library.PaperRecord{record}); err != nil {
		t.Fatalf("import: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "oa", "acquisition-queue"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("queue code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	path := filepath.Join(project, "data", "legal-acquisition-queue.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("queue not written: %v", err)
	}
	var queue documents.LegalAcquisitionQueue
	if err := json.Unmarshal(data, &queue); err != nil {
		t.Fatalf("decode queue: %v", err)
	}
	if len(queue.Items) != 1 || !queue.Items[0].ReviewerApprovalRequired || queue.Items[0].ReviewerApproved || queue.Items[0].ExpectedLocalPath == "" || queue.Items[0].SourceURL == "" {
		t.Fatalf("queue = %#v", queue)
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "oa", "acquisition-approve", queue.Items[0].ID, "--reviewer", "reviewer-a", "--reason", "license checked"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("approve code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, _ = os.ReadFile(path)
	if err := json.Unmarshal(data, &queue); err != nil {
		t.Fatalf("decode approved queue: %v", err)
	}
	if !queue.Items[0].ReviewerApproved || queue.Items[0].Reviewer != "reviewer-a" {
		t.Fatalf("approved queue = %#v", queue)
	}
	if err := documents.GuardAcquisition(queue.Items[0], documents.AcquisitionUseDownload); err != nil {
		t.Fatalf("approved item blocked: %v", err)
	}
}
