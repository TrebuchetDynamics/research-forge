package oss

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckInventoryDriftFlagsNoteHeadingMetadataAndUnreferencedNotes(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("# Different\n\nArea: parser\nDisposition: avoid\nRepository: other/repo\nLicense policy: old\nNext slice: old slice\n"), 0o644); err != nil {
		t.Fatalf("write alpha note: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra.md"), []byte("# Extra\n"), 0o644); err != nil {
		t.Fatalf("write extra note: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"schemaVersion":"1","entries":[{"id":"alpha","name":"Alpha","repository":"owner/repo","area":"scholarly-graph-source","disposition":"adapter-only","licensePolicy":"adapter only","note":"alpha.md","risk":"drift","nextSlice":"new slice"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	result, err := CheckInventoryDrift(manifestPath)
	if err != nil {
		t.Fatalf("CheckInventoryDrift returned error: %v", err)
	}
	for _, want := range []string{
		"alpha: note heading \"Different\" does not match manifest name \"Alpha\"",
		"alpha: note area \"parser\" does not match manifest area \"scholarly-graph-source\"",
		"alpha: note disposition \"avoid\" does not match manifest disposition \"adapter-only\"",
		"alpha: note repository \"other/repo\" does not match manifest repository \"owner/repo\"",
		"alpha: note license policy does not match manifest licensePolicy",
		"alpha: note next slice does not match manifest nextSlice",
		"extra.md: note is not referenced by manifest",
	} {
		if !result.Contains(want) {
			t.Fatalf("drift result missing %q: %#v", want, result.Issues)
		}
	}
	if result.EntryCount != 1 || result.NoteCount != 2 {
		t.Fatalf("counts = %+v", result)
	}
}
