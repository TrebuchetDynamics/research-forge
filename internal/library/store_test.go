package library

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenStoreDoesNotFollowDanglingSymlink(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "library.json")
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	if err := os.Symlink(outsidePath, storePath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := OpenStore(storePath)
	if err == nil {
		t.Fatal("OpenStore succeeded with a dangling store symlink")
	}
	if _, statErr := os.Stat(outsidePath); !os.IsNotExist(statErr) {
		t.Fatalf("outside path stat error = %v, want not exist", statErr)
	}
	info, lstatErr := os.Lstat(storePath)
	if lstatErr != nil {
		t.Fatalf("lstat store path: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("OpenStore replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestStoreReplaceAllDoesNotWriteThroughSymlinkedPath(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "library.json")
	store, err := OpenStore(storePath)
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	outsideBefore := []byte("outside library must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside library: %v", err)
	}
	if err := os.Remove(storePath); err != nil {
		t.Fatalf("remove store path: %v", err)
	}
	if err := os.Symlink(outsidePath, storePath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if err := store.ReplaceAll(nil); err == nil {
		t.Fatal("ReplaceAll succeeded with a symlinked store path")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside library: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("ReplaceAll wrote through store symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(storePath)
	if lstatErr != nil {
		t.Fatalf("lstat store path: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("ReplaceAll replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestStoreReplaceAllPreservesPermissionsAndCleansStagingFiles(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "library.json")
	if err := os.WriteFile(storePath, []byte("[]\n"), 0o600); err != nil {
		t.Fatalf("write prior store: %v", err)
	}
	store, err := OpenStore(storePath)
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	if err := store.ReplaceAll([]PaperRecord{}); err != nil {
		t.Fatalf("ReplaceAll returned error: %v", err)
	}
	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("stat store: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store mode = %o, want 600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read store directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(storePath) {
		t.Fatalf("store directory entries = %#v, want only %s", entries, filepath.Base(storePath))
	}
}

func TestStoreListDoesNotReadThroughSymlinkedPath(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "library.json")
	store, err := OpenStore(storePath)
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outsidePath, []byte("[{\"Title\":\"outside private record\"}]\n"), 0o640); err != nil {
		t.Fatalf("write outside library: %v", err)
	}
	if err := os.Remove(storePath); err != nil {
		t.Fatalf("remove store path: %v", err)
	}
	if err := os.Symlink(outsidePath, storePath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if records, err := store.List(); err == nil {
		t.Fatalf("List succeeded through a store symlink: records=%#v", records)
	}
	info, lstatErr := os.Lstat(storePath)
	if lstatErr != nil {
		t.Fatalf("lstat store path: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("List replaced store symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestStoreCreateUpdateListAndSearchPaperRecords(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	record, err := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis catalyst review", Identifiers: Identifiers{DOI: "10.1000/example"}, Year: 2026})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	record.Abstract = "Updated abstract"
	if err := store.Update(record); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].Abstract != "Updated abstract" {
		t.Fatalf("items = %#v", items)
	}
	results, err := store.Search("catalyst")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 || results[0].Identifiers.DOI != "10.1000/example" {
		t.Fatalf("results = %#v", results)
	}
}
