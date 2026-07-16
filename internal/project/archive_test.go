package project

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// capWriter accepts only the first limit bytes, then fails. Used to force a
// write error during tar.Writer.Close's trailer flush, which only issues
// writes inside Close and is otherwise untestable via WriteHeader/Write.
type capWriter struct {
	buf     bytes.Buffer
	limit   int
	written int
}

func TestRestorePreservesExistingFileWhenArchiveEntryIsTruncated(t *testing.T) {
	var complete bytes.Buffer
	writer := tar.NewWriter(&complete)
	replacement := []byte("replacement")
	if err := writer.WriteHeader(&tar.Header{Name: "existing.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(replacement))}); err != nil {
		t.Fatalf("write archive header: %v", err)
	}
	if _, err := writer.Write(replacement); err != nil {
		t.Fatalf("write archive entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close complete archive: %v", err)
	}
	archivePath := filepath.Join(t.TempDir(), "truncated.tar")
	const partialEntrySize = 4
	if err := os.WriteFile(archivePath, complete.Bytes()[:512+partialEntrySize], 0o644); err != nil {
		t.Fatalf("write truncated archive: %v", err)
	}

	destination := t.TempDir()
	target := filepath.Join(destination, "existing.txt")
	before := []byte("keep existing\n")
	if err := os.WriteFile(target, before, 0o640); err != nil {
		t.Fatalf("write existing target: %v", err)
	}
	if err := Restore(archivePath, destination); err == nil {
		t.Fatal("Restore succeeded with a truncated archive entry")
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read existing target: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("existing target changed after failed restore: got %q, want %q", after, before)
	}
}

func TestRestorePreservesEarlierFilesWhenLaterArchiveEntryIsTruncated(t *testing.T) {
	var complete bytes.Buffer
	writer := tar.NewWriter(&complete)
	entries := []struct {
		name string
		body []byte
	}{
		{name: "first.txt", body: []byte("first replacement body")},
		{name: "second.txt", body: []byte("second replacement body")},
	}
	for _, entry := range entries {
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(entry.body))}); err != nil {
			t.Fatalf("write %s header: %v", entry.name, err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			t.Fatalf("write %s body: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close complete archive: %v", err)
	}
	secondBodyOffset := bytes.Index(complete.Bytes(), entries[1].body)
	if secondBodyOffset < 0 {
		t.Fatal("second archive body not found")
	}
	archivePath := filepath.Join(t.TempDir(), "later-truncated.tar")
	if err := os.WriteFile(archivePath, complete.Bytes()[:secondBodyOffset+4], 0o644); err != nil {
		t.Fatalf("write truncated archive: %v", err)
	}

	destination := t.TempDir()
	before := map[string][]byte{
		"first.txt":  []byte("keep first\n"),
		"second.txt": []byte("keep second\n"),
	}
	for name, data := range before {
		if err := os.WriteFile(filepath.Join(destination, name), data, 0o640); err != nil {
			t.Fatalf("write existing %s: %v", name, err)
		}
	}
	if err := Restore(archivePath, destination); err == nil {
		t.Fatal("Restore succeeded with a truncated later archive entry")
	}
	for name, want := range before {
		got, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil {
			t.Fatalf("read existing %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("existing %s changed after failed restore: got %q, want %q", name, got, want)
		}
	}
}

func TestRestorePreflightsLaterDestinationConflictBeforeChangingEarlierFile(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	entries := []struct {
		name string
		body []byte
	}{
		{name: "first.txt", body: []byte("first replacement")},
		{name: "z-blocked/second.txt", body: []byte("second replacement")},
	}
	for _, entry := range entries {
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(entry.body))}); err != nil {
			t.Fatalf("write %s header: %v", entry.name, err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			t.Fatalf("write %s body: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	archivePath := filepath.Join(t.TempDir(), "conflict.tar")
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	destination := t.TempDir()
	firstPath := filepath.Join(destination, "first.txt")
	firstBefore := []byte("keep first\n")
	if err := os.WriteFile(firstPath, firstBefore, 0o640); err != nil {
		t.Fatalf("write existing first file: %v", err)
	}
	blockerPath := filepath.Join(destination, "z-blocked")
	blockerBefore := []byte("keep blocker\n")
	if err := os.WriteFile(blockerPath, blockerBefore, 0o640); err != nil {
		t.Fatalf("write destination blocker: %v", err)
	}

	if err := Restore(archivePath, destination); err == nil {
		t.Fatal("Restore succeeded with a directory/file destination conflict")
	}
	for path, want := range map[string][]byte{firstPath: firstBefore, blockerPath: blockerBefore} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved destination %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("destination %s changed after preflight failure: got %q, want %q", path, got, want)
		}
	}
}

func TestRestoreRollsBackEarlierFilesWhenLaterInstallFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read mode-000 staged files")
	}
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	entries := []struct {
		name string
		body []byte
		mode int64
	}{
		{name: "first.txt", body: []byte("first replacement"), mode: 0o644},
		{name: "z-unreadable.txt", body: []byte("second replacement"), mode: 0},
	}
	for _, entry := range entries {
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Typeflag: tar.TypeReg, Mode: entry.mode, Size: int64(len(entry.body))}); err != nil {
			t.Fatalf("write %s header: %v", entry.name, err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			t.Fatalf("write %s body: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	archivePath := filepath.Join(t.TempDir(), "install-failure.tar")
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	destination := t.TempDir()
	firstPath := filepath.Join(destination, "first.txt")
	firstBefore := []byte("keep first\n")
	if err := os.WriteFile(firstPath, firstBefore, 0o640); err != nil {
		t.Fatalf("write existing first file: %v", err)
	}
	if err := Restore(archivePath, destination); err == nil {
		t.Fatal("Restore succeeded despite an unreadable later staged file")
	}
	firstAfter, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read restored first file: %v", err)
	}
	if !bytes.Equal(firstAfter, firstBefore) {
		t.Fatalf("first file was not rolled back: got %q, want %q", firstAfter, firstBefore)
	}
	if _, err := os.Stat(filepath.Join(destination, "z-unreadable.txt")); !os.IsNotExist(err) {
		t.Fatalf("failed restore left later target: %v", err)
	}
}

func (c *capWriter) Write(p []byte) (int, error) {
	if c.written+len(p) > c.limit {
		return 0, errors.New("simulated write failure")
	}
	n, err := c.buf.Write(p)
	c.written += n
	return n, err
}

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

func TestArchiveToPropagatesTrailerWriteError(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(projectPath, CreateOptions{Title: "Demo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var full bytes.Buffer
	if err := archiveTo(&full, projectPath); err != nil {
		t.Fatalf("archiveTo (uncapped): %v", err)
	}

	capped := &capWriter{limit: full.Len() - 1}
	if err := archiveTo(capped, projectPath); err == nil {
		t.Fatalf("archiveTo returned nil error despite a failing write during the tar trailer flush")
	}
}

func TestArchivePreservesExistingOutputWhenProjectReadFails(t *testing.T) {
	outputDir := t.TempDir()
	archivePath := filepath.Join(outputDir, "review.rforge.tar")
	priorArchive := []byte("existing valid archive bytes\n")
	if err := os.WriteFile(archivePath, priorArchive, 0o640); err != nil {
		t.Fatalf("write prior archive: %v", err)
	}

	if err := Archive(filepath.Join(t.TempDir(), "missing-project"), archivePath); err == nil {
		t.Fatal("Archive returned nil error for a missing project")
	}
	got, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read prior archive after failure: %v", err)
	}
	if !bytes.Equal(got, priorArchive) {
		t.Fatalf("archive after failure = %q, want %q", got, priorArchive)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read archive output directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(archivePath) {
		t.Fatalf("archive output entries = %#v, want only %s", entries, filepath.Base(archivePath))
	}
}

func TestArchiveAndRestoreProject(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(projectPath, CreateOptions{Title: "Demo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	outputDir := t.TempDir()
	archivePath := filepath.Join(outputDir, "demo.rforge.tar")
	if err := os.WriteFile(archivePath, []byte("prior archive\n"), 0o640); err != nil {
		t.Fatalf("write prior archive: %v", err)
	}
	if err := Archive(projectPath, archivePath); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read archive output directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(archivePath) {
		t.Fatalf("archive output entries = %#v, want only %s", entries, filepath.Base(archivePath))
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
