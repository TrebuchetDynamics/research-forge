package documents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportLocalFileCopiesIntoProjectWithLocalOnlyStatus(t *testing.T) {
	projectPath := t.TempDir()
	source := filepath.Join(t.TempDir(), "private.pdf")
	if err := os.WriteFile(source, []byte("%PDF-1.4 local fixture"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	asset, err := ImportLocalFile(projectPath, source, "paper-1")
	if err != nil {
		t.Fatalf("ImportLocalFile returned error: %v", err)
	}
	if !asset.LocalOnly || asset.OAStatus != "local-only" || asset.AcquisitionSource != "manual-local" || asset.MIMEType != "application/pdf" {
		t.Fatalf("asset = %#v", asset)
	}
	if _, err := os.Stat(asset.LocalPath); err != nil {
		t.Fatalf("missing copied asset: %v", err)
	}
	if filepath.Dir(asset.LocalPath) != filepath.Join(projectPath, "documents", "local") {
		t.Fatalf("LocalPath = %q", asset.LocalPath)
	}
}
