package sources

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestResponseCacheWritesAndReadsRawSourcePayload(t *testing.T) {
	cache := NewResponseCache(filepath.Join(t.TempDir(), "cache"))
	payload := []byte(`{"results":[{"id":"W1"}]}`)

	ref, err := cache.Write("openalex", "artificial photosynthesis", payload)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if ref == "" {
		t.Fatalf("cache ref is empty")
	}
	read, err := cache.Read(ref)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if string(read) != string(payload) {
		t.Fatalf("payload = %s, want %s", read, payload)
	}
}

func TestResponseCacheUsesStableRefsForSameSourceAndQuery(t *testing.T) {
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	cache := NewResponseCache(cacheRoot)
	first, err := cache.Write("openalex", "artificial photosynthesis", []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("first Write returned error: %v", err)
	}
	entryPath := filepath.Join(cacheRoot, filepath.FromSlash(first))
	if err := os.Chmod(entryPath, 0o600); err != nil {
		t.Fatalf("chmod cache entry: %v", err)
	}
	secondPayload := []byte(`{"a":2}`)
	second, err := cache.Write("openalex", "artificial photosynthesis", secondPayload)
	if err != nil {
		t.Fatalf("second Write returned error: %v", err)
	}
	if first != second {
		t.Fatalf("refs differ: first=%q second=%q", first, second)
	}
	read, err := cache.Read(second)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if !bytes.Equal(read, secondPayload) {
		t.Fatalf("payload = %s, want %s", read, secondPayload)
	}
	info, err := os.Stat(entryPath)
	if err != nil {
		t.Fatalf("stat cache entry: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cache entry mode = %o, want 600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(entryPath))
	if err != nil {
		t.Fatalf("read cache entry directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(entryPath) {
		t.Fatalf("cache entry directory entries = %#v, want only %s", entries, filepath.Base(entryPath))
	}
}

func TestResponseCacheWriteDoesNotWriteThroughSymlinkedEntry(t *testing.T) {
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	cache := NewResponseCache(cacheRoot)
	ref, err := cache.Write("openalex", "artificial photosynthesis", []byte(`{"first":true}`))
	if err != nil {
		t.Fatalf("first Write returned error: %v", err)
	}
	entryPath := filepath.Join(cacheRoot, filepath.FromSlash(ref))
	if err := os.Remove(entryPath); err != nil {
		t.Fatalf("remove cache entry: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	outsideBefore := []byte("outside cache payload must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside cache payload: %v", err)
	}
	if err := os.Symlink(outsidePath, entryPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if _, err := cache.Write("openalex", "artificial photosynthesis", []byte(`{"second":true}`)); err == nil {
		t.Fatal("Write succeeded with a symlinked cache entry")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside cache payload: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("Write followed cache symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(entryPath)
	if lstatErr != nil {
		t.Fatalf("lstat cache entry: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Write replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestResponseCacheReadDoesNotReadThroughSymlinkedEntry(t *testing.T) {
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	cache := NewResponseCache(cacheRoot)
	ref, err := cache.Write("openalex", "artificial photosynthesis", []byte(`{"cached":true}`))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	entryPath := filepath.Join(cacheRoot, filepath.FromSlash(ref))
	if err := os.Remove(entryPath); err != nil {
		t.Fatalf("remove cache entry: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outsidePath, []byte(`{"outside":"private"}`), 0o640); err != nil {
		t.Fatalf("write outside cache payload: %v", err)
	}
	if err := os.Symlink(outsidePath, entryPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if payload, err := cache.Read(ref); err == nil {
		t.Fatalf("Read succeeded through a cache symlink: payload=%s", payload)
	}
	info, lstatErr := os.Lstat(entryPath)
	if lstatErr != nil {
		t.Fatalf("lstat cache entry: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Read replaced cache symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestResponseCacheReadRejectsEmptyRoot(t *testing.T) {
	workingDir := t.TempDir()
	priorDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(priorDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	if err := os.WriteFile("entry.json", []byte(`{"outside":"working-directory"}`), 0o640); err != nil {
		t.Fatalf("write working-directory file: %v", err)
	}

	cache := NewResponseCache("")
	if payload, err := cache.Read("entry.json"); err == nil {
		t.Fatalf("Read succeeded with an empty cache root: payload=%s", payload)
	}
}
