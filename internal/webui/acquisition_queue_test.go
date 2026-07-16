package webui

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
)

func TestAcquisitionQueueHandlerShowsLegalFieldsAndApprovalGate(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "legal-acquisition-queue.json"), documents.LegalAcquisitionQueue{SchemaVersion: "1", Items: []documents.LegalAcquisitionQueueItem{{ID: "acq-001", PaperTitle: "OA paper", Source: "unpaywall", SourceURL: "https://example.org/paper.pdf", ExpectedLocalPath: filepath.Join(project, "documents", "open-access", "paper.pdf"), License: "CC-BY", OAStatus: "gold", Restricted: false, Shareable: true, ReviewerApprovalRequired: true}}})
	rec := httptest.NewRecorder()
	newAcquisitionQueueHandler(func() string { return project }).ServeHTTP(rec, httptest.NewRequest("GET", "/acquisition", nil))
	body := rec.Body.String()
	for _, want := range []string{"Legal full-text acquisition queue", "OA/license status", "https://example.org/paper.pdf", "Expected stored path", "restricted=false", "shareable=true", "explicit reviewer approval", "rforge --project"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestAcquisitionQueueHandlerDoesNotReadSymlinkedQueue(t *testing.T) {
	projectPath := t.TempDir()
	dataDir := filepath.Join(projectPath, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir project data: %v", err)
	}
	externalQueue := documents.LegalAcquisitionQueue{SchemaVersion: "1", Items: []documents.LegalAcquisitionQueueItem{{
		ID: "external-private-acquisition", PaperTitle: "External private paper", Source: "external-private-source",
		SourceURL: "https://private.invalid/external-paper.pdf", ExpectedLocalPath: "/external/private/paper.pdf", Restricted: true,
	}}}
	externalPath := filepath.Join(t.TempDir(), "legal-acquisition-queue.json")
	writeJSON(t, externalPath, externalQueue)
	queuePath := filepath.Join(dataDir, "legal-acquisition-queue.json")
	if err := os.Symlink(externalPath, queuePath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	rec := httptest.NewRecorder()
	newAcquisitionQueueHandler(func() string { return projectPath }).ServeHTTP(rec, httptest.NewRequest("GET", "/acquisition", nil))
	body := rec.Body.String()
	for _, private := range []string{"external-private-acquisition", "External private paper", "private.invalid"} {
		if strings.Contains(body, private) {
			t.Fatalf("acquisition queue disclosed %q from symlinked data: %s", private, body)
		}
	}
	if info, err := os.Lstat(queuePath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("acquisition queue symlink changed: info=%v err=%v", info, err)
	}
}
