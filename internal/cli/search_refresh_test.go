package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePriorTopic writes a manifest.json and results-deduped.jsonl representing
// a prior search run, so search refresh can compute a DOI delta against it.
func writePriorTopic(t *testing.T, dir string, queries, sources []string, dois []string) {
	t.Helper()
	manifest := map[string]any{
		"schemaVersion": "1",
		"createdAt":     "2026-07-01T00:00:00Z",
		"queries":       queries,
		"sources":       sources,
		"limit":         20,
		"results":       len(dois),
		"deduped":       len(dois),
		"failures":      0,
	}
	mb, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	var b strings.Builder
	for _, doi := range dois {
		rec, _ := json.Marshal(map[string]any{
			"Title":       "Prior paper " + doi,
			"Identifiers": map[string]string{"DOI": doi},
		})
		b.Write(rec)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "results-deduped.jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write results-deduped: %v", err)
	}
}

// TestExecuteSearchRefreshDryRunReportsPriorManifestWithoutNetwork verifies
// that --dry-run reads the stored manifest and reports what would be re-queried
// without making any network calls or writing outputs. This is the trust gate:
// users hand-version dirs (-v2/-wave) because they cannot preview a refresh.
func TestExecuteSearchRefreshDryRunReportsPriorManifestWithoutNetwork(t *testing.T) {
	dir := t.TempDir()
	writePriorTopic(t, dir, []string{"limit order book prediction", "market microstructure"}, []string{"openalex"}, []string{"10.0000/prior-a", "10.0000/prior-b"})

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { hits++ }))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "refresh", "--dir", dir, "--dry-run"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}
	if hits != 0 {
		t.Fatalf("--dry-run made %d network request(s); must make none", hits)
	}
	out := stdout.String()
	if !strings.Contains(out, "limit order book prediction") || !strings.Contains(out, "market microstructure") {
		t.Fatalf("dry-run must list the stored queries: %s", out)
	}
	if !strings.Contains(out, "openalex") {
		t.Fatalf("dry-run must list the stored sources: %s", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("dry-run must report the prior record count: %s", out)
	}
}

// TestExecuteSearchRefreshReportsDOIDelta verifies that refresh re-runs the
// stored queries and reports new / unchanged / gone DOIs versus the prior run,
// replacing the manual -v2/-wave directory hand-versioning pattern.
func TestExecuteSearchRefreshReportsDOIDelta(t *testing.T) {
	dir := t.TempDir()
	// Prior run had a and b. Refresh returns b and c (a is gone, c is new).
	writePriorTopic(t, dir, []string{"market microstructure"}, []string{"openalex"}, []string{"10.0000/a", "10.0000/b"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/Wb","doi":"https://doi.org/10.0000/b","title":"B"},{"id":"https://openalex.org/Wc","doi":"https://doi.org/10.0000/c","title":"C"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "refresh", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	newDOIs, _ := data["new"].([]any)
	goneDOIs, _ := data["gone"].([]any)
	unchanged, _ := data["unchanged"].(float64)
	if len(newDOIs) != 1 || newDOIs[0] != "10.0000/c" {
		t.Fatalf("new = %#v, want [10.0000/c]", newDOIs)
	}
	if len(goneDOIs) != 1 || goneDOIs[0] != "10.0000/a" {
		t.Fatalf("gone = %#v, want [10.0000/a]", goneDOIs)
	}
	if int(unchanged) != 1 {
		t.Fatalf("unchanged = %v, want 1", unchanged)
	}
	// Refresh must update results-deduped.jsonl with the new run.
	updated, err := os.ReadFile(filepath.Join(dir, "results-deduped.jsonl"))
	if err != nil {
		t.Fatalf("read updated results-deduped: %v", err)
	}
	if !strings.Contains(string(updated), "10.0000/c") {
		t.Fatalf("results-deduped.jsonl must contain the new DOI after refresh; got %s", updated)
	}
	if strings.Contains(string(updated), "10.0000/a") {
		t.Fatalf("results-deduped.jsonl must drop the gone DOI after refresh; got %s", updated)
	}
}

// TestExecuteSearchRefreshFailsWithoutPriorManifest verifies refresh refuses to
// run on a directory with no manifest.json, so it never silently re-queries an
// unknown topic.
func TestExecuteSearchRefreshFailsWithoutPriorManifest(t *testing.T) {
	dir := t.TempDir()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"search", "refresh", "--dir", dir}, stdout, stderr)
	if code == 0 {
		t.Fatalf("refresh must fail without manifest.json; stdout = %s", stdout.String())
	}
}
