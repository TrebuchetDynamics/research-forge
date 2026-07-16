package filetxn

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReplaceFromReaderPreservesExistingFileWhenReadFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	before := []byte("before\n")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	copyErr := errors.New("simulated read failure")
	source := io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(copyErr))

	err := ReplaceFromReader(path, source, 0o644)
	if !errors.Is(err, copyErr) {
		t.Fatalf("ReplaceFromReader error = %v, want read failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read preserved target: %v", readErr)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("target changed after failed replacement: got %q, want %q", after, before)
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("stat preserved target: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("preserved target mode = %o, want 600", got)
	}
	assertNoTransactionDebris(t, dir)
}

func TestReplaceFromReaderDoesNotWriteThroughSymlinkedParent(t *testing.T) {
	outsideDir := t.TempDir()
	parent := filepath.Join(t.TempDir(), "output")
	if err := os.Symlink(outsideDir, parent); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	path := filepath.Join(parent, "target.txt")

	if err := ReplaceFromReader(path, strings.NewReader("data"), 0o644); err == nil {
		t.Fatal("ReplaceFromReader followed symlinked parent directory")
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "target.txt")); !os.IsNotExist(err) {
		t.Fatalf("target created through symlinked parent: %v", err)
	}
	assertNoTransactionDebris(t, outsideDir)
}

func TestReplaceAllRestoresEarlierFilesWhenLaterTargetCannotBeStaged(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.txt")
	firstBefore := []byte("first before\n")
	if err := os.WriteFile(firstPath, firstBefore, 0o600); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	secondPath := filepath.Join(dir, "missing", "second.txt")

	err := ReplaceAll([]Output{
		{Path: firstPath, Data: []byte("first after\n"), Mode: 0o644},
		{Path: secondPath, Data: []byte("second after\n"), Mode: 0o644},
	})
	if err == nil {
		t.Fatal("ReplaceAll succeeded when the second target could not be staged")
	}
	firstAfter, readErr := os.ReadFile(firstPath)
	if readErr != nil {
		t.Fatalf("read restored first file: %v", readErr)
	}
	if !bytes.Equal(firstAfter, firstBefore) {
		t.Fatalf("first file was not restored: got %q, want %q", firstAfter, firstBefore)
	}
	info, statErr := os.Stat(firstPath)
	if statErr != nil {
		t.Fatalf("stat restored first file: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored first mode = %o, want 600", got)
	}
	if _, statErr := os.Stat(secondPath); !os.IsNotExist(statErr) {
		t.Fatalf("second target exists after rollback: err=%v", statErr)
	}
	assertNoTransactionDebris(t, dir)
}

func TestReplaceAllValidatesEveryTargetBeforeChangingAny(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.txt")
	firstBefore := []byte("first before\n")
	if err := os.WriteFile(firstPath, firstBefore, 0o600); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.txt")
	outsideBefore := []byte("outside before\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	secondPath := filepath.Join(dir, "second.txt")
	if err := os.Symlink(outsidePath, secondPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	err := ReplaceAll([]Output{
		{Path: firstPath, Data: []byte("first after\n"), Mode: 0o644},
		{Path: secondPath, Data: []byte("second after\n"), Mode: 0o644},
	})
	if err == nil {
		t.Fatal("ReplaceAll succeeded with a symlinked second target")
	}
	firstAfter, readErr := os.ReadFile(firstPath)
	if readErr != nil {
		t.Fatalf("read first file: %v", readErr)
	}
	if !bytes.Equal(firstAfter, firstBefore) {
		t.Fatalf("first file changed before target validation completed: got %q, want %q", firstAfter, firstBefore)
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside file: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("ReplaceAll wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
}

func TestReplaceAllThenRestoresExistingAndNewFilesWhenCommitFails(t *testing.T) {
	dir := t.TempDir()
	existingPath := filepath.Join(dir, "existing.txt")
	existingBefore := []byte("existing before\n")
	if err := os.WriteFile(existingPath, existingBefore, 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	newPath := filepath.Join(dir, "new.txt")
	commitErr := errors.New("commit failed")

	err := ReplaceAllThen([]Output{
		{Path: existingPath, Data: []byte("existing after\n"), Mode: 0o644},
		{Path: newPath, Data: []byte("new after\n"), Mode: 0o640},
	}, func() error { return commitErr })
	if !errors.Is(err, commitErr) {
		t.Fatalf("ReplaceAllThen error = %v, want commit error", err)
	}
	existingAfter, readErr := os.ReadFile(existingPath)
	if readErr != nil {
		t.Fatalf("read restored existing file: %v", readErr)
	}
	if !bytes.Equal(existingAfter, existingBefore) {
		t.Fatalf("existing file was not restored: got %q, want %q", existingAfter, existingBefore)
	}
	info, statErr := os.Stat(existingPath)
	if statErr != nil {
		t.Fatalf("stat restored existing file: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored existing mode = %o, want 600", got)
	}
	if _, statErr := os.Stat(newPath); !os.IsNotExist(statErr) {
		t.Fatalf("new file remains after commit rollback: err=%v", statErr)
	}
	assertNoTransactionDebris(t, dir)
}

func assertNoTransactionDebris(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read transaction directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("transaction debris remains: %s", entry.Name())
		}
	}
}
