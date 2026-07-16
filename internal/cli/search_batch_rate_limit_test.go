package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecuteSearchBatchWritesNoRawFileOnRateLimit is the regression test for
// the dominant error category observed across real rforge usage: a 429 from
// Semantic Scholar must be recorded in failures.jsonl and must NOT leave an
// empty search-<source>-*.txt file in raw/, which previously polluted stats
// and looked like a zero-hit query.
func TestExecuteSearchBatchWritesNoRawFileOnRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY", "")

	out := t.TempDir()
	queries := filepath.Join(t.TempDir(), "queries.txt")
	if err := os.WriteFile(queries, []byte("limit order book prediction\n"), 0o600); err != nil {
		t.Fatalf("write queries: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"search", "batch", "--queries", queries, "--sources", "semantic-scholar", "--out", out, "--continue-on-error"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}

	rawDir := filepath.Join(out, "raw")
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		// raw dir may be absent; that is the strongest form of "no empty file".
		if !os.IsNotExist(err) {
			t.Fatalf("read raw dir: %v", err)
		}
		entries = nil
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "search-semantic-scholar-") {
			info, statErr := entry.Info()
			if statErr != nil {
				t.Fatalf("stat raw file %s: %v", name, statErr)
			}
			if info.Size() == 0 {
				t.Fatalf("rate-limited search left an empty raw file %s; 429 must record in failures.jsonl, not raw/", name)
			}
			t.Fatalf("rate-limited search wrote an unexpected raw file %s (size %d); 429 must skip raw/", name, info.Size())
		}
	}

	failures, err := os.ReadFile(filepath.Join(out, "failures.jsonl"))
	if err != nil {
		t.Fatalf("read failures.jsonl: %v", err)
	}
	if !strings.Contains(string(failures), "semantic-scholar") || !strings.Contains(string(failures), "rate limited") {
		t.Fatalf("failures.jsonl must record the semantic-scholar 429; got %s", failures)
	}

	// manifest must report the failure count so the run is auditable.
	manifest, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	if !strings.Contains(string(manifest), "semantic-scholar") || !strings.Contains(string(manifest), "failures") {
		t.Fatalf("manifest must record the semantic-scholar source and failures field; got %s", manifest)
	}
	if !strings.Contains(string(manifest), `"failures": 1`) && !strings.Contains(string(manifest), `"failures":1`) {
		t.Fatalf("manifest must report 1 failure; got %s", manifest)
	}
}
