package project

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreRejectsSymlinkEntries(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "malicious.tar")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	writer := tar.NewWriter(file)
	if err := writer.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "rforge.project.toml", Mode: 0o777}); err != nil {
		t.Fatalf("write symlink header: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}

	if err := Restore(archivePath, t.TempDir()); err == nil {
		t.Fatalf("Restore returned nil error for symlink entry")
	}
}

func TestArchiveAndRestoreProject(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(projectPath, CreateOptions{Title: "Demo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	archivePath := filepath.Join(t.TempDir(), "demo.rforge.tar")
	if err := Archive(projectPath, archivePath); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	restorePath := filepath.Join(t.TempDir(), "restored")
	if err := Restore(archivePath, restorePath); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	proj, err := Inspect(restorePath)
	if err != nil {
		t.Fatalf("Inspect restored: %v", err)
	}
	if proj.Title != "Demo" {
		t.Fatalf("restored title = %q", proj.Title)
	}
}
