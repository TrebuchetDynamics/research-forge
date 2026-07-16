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

// TestExecuteSearchResumeDryRunListsPendingFailuresWithoutNetwork is the
// regression test for the resume trust gap observed across real rforge usage:
// 39 uncleared failures.jsonl files mean users do not run resume because they
// cannot preview what it would retry. --dry-run must list pending source/query
// pairs and count them without any network call and without writing outputs.
func TestExecuteSearchResumeDryRunListsPendingFailuresWithoutNetwork(t *testing.T) {
	dir := t.TempDir()
	failuresPath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(failuresPath, []byte(
		`{"source":"semantic-scholar","query":"limit order book prediction","error":"source HTTP status 429: rate limited"}
{"source":"openalex","query":"ifs attractor","error":"timeout after 10s"}
`), 0o600); err != nil {
		t.Fatalf("write failures: %v", err)
	}
	out := t.TempDir()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// A server that fails any request proves --dry-run makes no network call:
	// if it did, the test would hang or the server would record a hit.
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		hits++
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)

	code := Execute([]string{"search", "resume", "--failures", failuresPath, "--out", out, "--dry-run"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	if hits != 0 {
		t.Fatalf("--dry-run made %d network request(s); must make none", hits)
	}
	if _, err := os.Stat(filepath.Join(out, "results.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("--dry-run must not write results.jsonl; stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "failures.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("--dry-run must not write failures.jsonl; stat err = %v", err)
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "semantic-scholar") || !strings.Contains(outStr, "limit order book prediction") {
		t.Fatalf("dry-run output must list the pending semantic-scholar query: %s", outStr)
	}
	if !strings.Contains(outStr, "openalex") || !strings.Contains(outStr, "ifs attractor") {
		t.Fatalf("dry-run output must list the pending openalex query: %s", outStr)
	}
}

func TestExecuteSearchResumeJSONDryRunReportsPending(t *testing.T) {
	dir := t.TempDir()
	failuresPath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(failuresPath, []byte(
		`{"source":"semantic-scholar","query":"q1","error":"429"}`+"\n"+
			`{"source":"openalex","query":"q2","error":"timeout"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write failures: %v", err)
	}
	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "resume", "--failures", failuresPath, "--out", t.TempDir(), "--dry-run"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	if int(data["pending"].(float64)) != 2 {
		t.Fatalf("pending = %v, want 2", data["pending"])
	}
	pending, ok := data["failures"].([]any)
	if !ok || len(pending) != 2 {
		t.Fatalf("failures = %#v, want 2 entries", data["failures"])
	}
}

// TestExecuteSearchResumeClearsRecoveredFailures locks the core resume promise:
// a failure that recovers on retry must NOT reappear in the output
// failures.jsonl, while a failure that still fails must. This is the
// "clears failures.jsonl" behavior that makes resume trustworthy.
func TestExecuteSearchResumeClearsRecoveredFailures(t *testing.T) {
	dir := t.TempDir()
	failuresPath := filepath.Join(dir, "failures.jsonl")
	if err := os.WriteFile(failuresPath, []byte(
		`{"source":"openalex","query":"recovered query","error":"timeout"}`+"\n"+
			`{"source":"semantic-scholar","query":"still failing","error":"429"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write failures: %v", err)
	}
	out := t.TempDir()
	// openalex recovers (returns a record); semantic-scholar still 429s.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "semantic-scholar") || strings.HasPrefix(r.URL.Path, "/graph") {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W1","doi":"https://doi.org/10.0000/r","title":"Recovered","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY", "")

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "resume", "--failures", failuresPath, "--out", out}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}

	newFailures, err := os.ReadFile(filepath.Join(out, "failures.jsonl"))
	if err != nil {
		t.Fatalf("read output failures.jsonl: %v", err)
	}
	if strings.Contains(string(newFailures), "recovered query") {
		t.Fatalf("recovered failure must be cleared from failures.jsonl; got %s", newFailures)
	}
	if !strings.Contains(string(newFailures), "still failing") {
		t.Fatalf("still-failing query must remain in failures.jsonl; got %s", newFailures)
	}
	results, err := os.ReadFile(filepath.Join(out, "results.jsonl"))
	if err != nil {
		t.Fatalf("read results.jsonl: %v", err)
	}
	if !strings.Contains(string(results), "Recovered") {
		t.Fatalf("recovered query must write its record to results.jsonl; got %s", results)
	}
}
