package reviewpkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestCreateBlocksReferenceManagerPrivateFieldsUntilPrivacyApproved(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	records := []library.PaperRecord{{Title: "Private Zotero", SourceRefs: []library.SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"attachment_files": "paper.pdf", "note": "private note", "annotations": "highlight"}}}}}
	writeJSONForPackageTest(t, filepath.Join(project, "data", "library.json"), records)
	_, err := Create(project, filepath.Join(t.TempDir(), "pkg"), Options{})
	if err == nil || !strings.Contains(err.Error(), "privacy/licensing review approval required") {
		t.Fatalf("expected privacy gate error, got %v", err)
	}
	review := documents.ApprovePrivacyLicensing(documents.ReviewPrivacyLicensing(documents.PrivacyLicensingReviewInput{Records: records}), "reviewer", "redacted")
	writeJSONForPackageTest(t, filepath.Join(project, "data", "privacy-licensing-review.json"), review)
	if _, err := Create(project, filepath.Join(t.TempDir(), "pkg-approved"), Options{}); err != nil {
		t.Fatalf("approved package blocked: %v", err)
	}
}

func writeJSONForPackageTest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
