package documents

import (
	"bytes"
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

func TestImportLocalFileDoesNotWriteThroughSymlinkedDestination(t *testing.T) {
	projectPath := t.TempDir()
	source := filepath.Join(t.TempDir(), "private.pdf")
	if err := os.WriteFile(source, []byte("%PDF-1.4 imported bytes"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	documentDir := filepath.Join(projectPath, "documents", "local")
	if err := os.MkdirAll(documentDir, 0o755); err != nil {
		t.Fatalf("create document directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.pdf")
	outsideBefore := []byte("outside local document must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside document: %v", err)
	}
	destination := filepath.Join(documentDir, filepath.Base(source))
	if err := os.Symlink(outsidePath, destination); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := ImportLocalFile(projectPath, source, "paper-1")
	if err == nil {
		t.Fatal("ImportLocalFile succeeded with a symlinked destination")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside document: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("ImportLocalFile wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(destination)
	if lstatErr != nil {
		t.Fatalf("lstat destination: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("ImportLocalFile replaced symlink despite rejecting destination: mode=%v", info.Mode())
	}
}
