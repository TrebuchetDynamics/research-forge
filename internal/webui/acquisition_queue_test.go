package webui

import (
	"net/http/httptest"
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
