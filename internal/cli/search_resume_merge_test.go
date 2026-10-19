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

// TestSearchResumePreservesPriorResultsSameDir fixes the data-loss trap in the
// legacy resume command: pointing --out at the original topic dir used to
// REPLACE results.jsonl with only the retried records, discarding the
// original sweep's accumulation. It must append instead.
func TestSearchResumePreservesPriorResultsSameDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-R","doi":"https://doi.org/10.0000/retried","title":"Retried fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	dir := t.TempDir()
	priorResults := `{"Title":"Prior fixture","Identifiers":{"DOI":"10.0000/prior","OpenAlexID":"W-P"},"Year":2025,"SourceRefs":[{"Source":"openalex"}]}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte(priorResults), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "failures.jsonl"), []byte(`{"source":"openalex","query":"prior fixture query","error":"timeout"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "raw"), 0o755); err != nil {
		t.Fatal(err)
	}
	existingRaw := filepath.Join(dir, "raw", "search-openalex-001-unrelated-query.txt")
	if err := os.WriteFile(existingRaw, []byte("10.0000/unrelated\tUnrelated Query\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"search", "resume", "--failures", filepath.Join(dir, "failures.jsonl"), "--out", dir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}
	results, err := os.ReadFile(filepath.Join(dir, "results.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(results), "10.0000/prior") || !strings.Contains(string(results), "10.0000/retried") {
		t.Fatalf("same-dir resume must preserve prior results and append retried records; got %s", results)
	}
	rawData, err := os.ReadFile(existingRaw)
	if err != nil || !strings.Contains(string(rawData), "10.0000/unrelated") {
		t.Fatalf("unrelated raw file must survive same-dir resume; got %q err=%v", rawData, err)
	}
	// The still-failing record is replaced by an empty failures file only when
	// the retry succeeds — here it succeeds, so failures.jsonl must be empty.
	failures, err := os.ReadFile(filepath.Join(dir, "failures.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(failures)) != "" {
		t.Fatalf("successful retry must clear failures.jsonl; got %s", failures)
	}
}

// TestSearchResumeReusesOriginalRawName guards the numbering collision: the
// legacy command renumbered retried raw files by pending index (001, 002…),
// overwriting a different query's raw record in the same dir. A retry must
// reuse the raw file the original sweep wrote for that exact (source, query).
func TestSearchResumeReusesOriginalRawName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-5","doi":"https://doi.org/10.0000/query5","title":"Query five fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "raw"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The original sweep numbered "chaos attractors" as query 5.
	originalRaw := filepath.Join(dir, "raw", "search-openalex-005-chaos-attractors.txt")
	if err := os.WriteFile(originalRaw, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	// A different query legitimately owns index 001.
	firstQueryRaw := filepath.Join(dir, "raw", "search-openalex-001-first-query.txt")
	if err := os.WriteFile(firstQueryRaw, []byte("10.0000/first\tFirst Query\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "failures.jsonl"), []byte(`{"source":"openalex","query":"chaos attractors","error":"timeout"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"search", "resume", "--failures", filepath.Join(dir, "failures.jsonl"), "--out", dir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}
	retriedRaw, err := os.ReadFile(originalRaw)
	if err != nil {
		t.Fatalf("retry must reuse the original raw file name: %v", err)
	}
	if !strings.Contains(string(retriedRaw), "10.0000/query5") {
		t.Fatalf("reused raw file must hold the retried results; got %q", retriedRaw)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw", "search-openalex-001-chaos-attractors.txt")); !os.IsNotExist(err) {
		t.Fatalf("retry must not renumber into another query's raw file: %v", err)
	}
	firstData, err := os.ReadFile(firstQueryRaw)
	if err != nil || !strings.Contains(string(firstData), "10.0000/first") {
		t.Fatalf("first query's raw file must be untouched; got %q err=%v", firstData, err)
	}
}

// TestSearchResumeUpdatesCheckpoint keeps the legacy command consistent with
// the incremental checkpoint when one exists: successful retries must move the
// pair from failures to completed.
func TestSearchResumeUpdatesCheckpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-C","doi":"https://doi.org/10.0000/checkpoint","title":"Checkpoint fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "failures.jsonl"), []byte(`{"source":"openalex","query":"chaos attractors","error":"timeout"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := searchBatchState{
		SchemaVersion: "1",
		Queries:       []string{"chaos attractors"},
		Sources:       []string{"openalex"},
		Limit:         25,
		Completed:     []searchBatchStatePair{},
		Failures:      []searchBatchFailure{{Source: "openalex", Query: "chaos attractors", Error: "timeout"}},
	}
	if err := writeSearchBatchState(dir, state); err != nil {
		t.Fatal(err)
	}

	code := Execute([]string{"search", "resume", "--failures", filepath.Join(dir, "failures.jsonl"), "--out", dir}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	after := readBatchState(t, dir)
	if len(after.Completed) != 1 || len(after.Failures) != 0 {
		t.Fatalf("checkpoint after retry: completed=%d failures=%d, want 1/0", len(after.Completed), len(after.Failures))
	}
	if after.Completed[0].Query != "chaos attractors" || after.Completed[0].Records != 1 {
		t.Fatalf("checkpoint completed entry = %+v", after.Completed[0])
	}
}

// TestSearchResumeStillFailingRecordsFreshError proves a retry that keeps
// failing replaces the stale error text instead of silently vanishing.
func TestSearchResumeStillFailingRecordsFreshError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "still broken", http.StatusNotFound)
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "failures.jsonl"), []byte(`{"source":"openalex","query":"chaos attractors","error":"timeout"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"search", "resume", "--failures", filepath.Join(dir, "failures.jsonl"), "--out", dir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr)
	}
	failures := readSearchBatchFailures(filepath.Join(dir, "failures.jsonl"))
	if len(failures) != 1 {
		t.Fatalf("failures after still-failing retry = %+v, want 1 entry", failures)
	}
	if failures[0].Error == "timeout" || !strings.Contains(failures[0].Error, "404") {
		t.Fatalf("failure must carry the fresh error, got %q", failures[0].Error)
	}
	if !strings.Contains(stdout.String(), "still failed") {
		t.Fatalf("stdout must report the still-failing count: %s", stdout)
	}
}
