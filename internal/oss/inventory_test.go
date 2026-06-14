package oss

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInventoryManifestRequiresNotesAndGovernanceFields(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(filepath.Join(root, "zotero.md"), []byte("# Zotero\n"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.WriteFile(manifest, []byte(`{
  "schemaVersion": "1",
  "entries": [
    {
      "id": "zotero",
      "name": "Zotero",
      "area": "reference-management",
      "disposition": "pattern-reference",
      "licensePolicy": "study-only",
      "note": "zotero.md",
      "risk": "Do not copy translators or source code without license review.",
      "nextSlice": "CSL-JSON and Better BibTeX interoperability."
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	result, err := ValidateInventoryManifest(manifest)
	if err != nil {
		t.Fatalf("ValidateInventoryManifest returned error: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("issues = %#v, want none", result.Issues)
	}
	if result.EntryCount != 1 {
		t.Fatalf("EntryCount = %d, want 1", result.EntryCount)
	}
}

func TestValidateInventoryManifestReportsMissingRequiredFields(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"broken"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	result, err := ValidateInventoryManifest(manifest)
	if err != nil {
		t.Fatalf("ValidateInventoryManifest returned error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Fatalf("want validation issues, got none")
	}
	for _, want := range []string{"missing name", "missing area", "missing disposition", "missing note"} {
		if !result.Contains(want) {
			t.Fatalf("issues missing %q: %#v", want, result.Issues)
		}
	}
}
