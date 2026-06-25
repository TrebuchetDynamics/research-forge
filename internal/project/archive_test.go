package project

import (
	"archive/tar"
	"io"
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

func TestRestoreRejectsExistingSymlinkParent(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "malicious-parent.tar")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	writer := tar.NewWriter(file)
	if err := writer.WriteHeader(&tar.Header{Name: "linked/file.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("secret"))}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := writer.Write([]byte("secret")); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	destination := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(destination, "linked")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := Restore(archivePath, destination); err == nil {
		t.Fatalf("Restore accepted existing symlink parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "file.txt")); !os.IsNotExist(err) {
		t.Fatalf("restore wrote through symlink parent: %v", err)
	}
}

func TestArchiveSkipsSymlinkedProjectFiles(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(projectPath, CreateOptions{Title: "Demo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(projectPath, "data", "secret-link.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	archivePath := filepath.Join(t.TempDir(), "demo.rforge.tar")
	if err := Archive(projectPath, archivePath); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	tr := tar.NewReader(file)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == "data/secret-link.txt" {
			t.Fatalf("archive included symlink entry: %#v", hdr)
		}
	}
}

func TestArchiveRejectsSymlinkOutputPath(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(projectPath, CreateOptions{Title: "Demo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside.tar")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "archive.tar")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := Archive(projectPath, link); err == nil {
		t.Fatalf("Archive accepted symlink output path")
	}
	data, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("archive overwrote symlink target: %q", data)
	}
}

func TestArchiveRejectsOutputInsideProject(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(projectPath, CreateOptions{Title: "Demo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := Archive(projectPath, filepath.Join(projectPath, "demo.rforge.tar")); err == nil {
		t.Fatalf("Archive accepted output inside project")
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
