package sources

import (
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
	cache := NewResponseCache(filepath.Join(t.TempDir(), "cache"))
	first, err := cache.Write("openalex", "artificial photosynthesis", []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("first Write returned error: %v", err)
	}
	second, err := cache.Write("openalex", "artificial photosynthesis", []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("second Write returned error: %v", err)
	}
	if first != second {
		t.Fatalf("refs differ: first=%q second=%q", first, second)
	}
}
