package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteSearchResumePreservesPriorArtifactsWhenOutputInitializationFails(t *testing.T) {
	inputDir := t.TempDir()
	inputPath := filepath.Join(inputDir, "failures.jsonl")
	if err := os.WriteFile(inputPath, []byte("{\"source\":\"semantic-scholar\",\"query\":\"fixture query\",\"error\":\"timeout\"}\n"), 0o644); err != nil {
		t.Fatalf("write resume input: %v", err)
	}
	out := t.TempDir()
	resultsPath := filepath.Join(out, "results.jsonl")
	priorResults := []byte("{\"sentinel\":\"prior resume results\"}\n")
	if err := os.WriteFile(resultsPath, priorResults, 0o640); err != nil {
		t.Fatalf("write prior resume results: %v", err)
	}
	if err := os.Mkdir(filepath.Join(out, "failures.jsonl"), 0o750); err != nil {
		t.Fatalf("create conflicting resume failures output: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"search", "resume", "--failures", inputPath, "--out", out}, stdout, stderr)
	if code == 0 {
		t.Fatalf("Execute succeeded with a directory at failures.jsonl; stdout=%s", stdout.String())
	}
	results, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read prior resume results after failure: %v", err)
	}
	if !bytes.Equal(results, priorResults) {
		t.Fatalf("resume results after failure = %q, want %q", results, priorResults)
	}
	if info, err := os.Stat(filepath.Join(out, "failures.jsonl")); err != nil || !info.IsDir() {
		t.Fatalf("conflicting resume failures output changed: info=%v err=%v", info, err)
	}
	if _, err := os.Stat(filepath.Join(out, "raw")); !os.IsNotExist(err) {
		t.Fatalf("failed resume left a new raw directory: %v", err)
	}
}

func TestExecuteSearchResumeRestoresAllArtifactsWhenCommitFails(t *testing.T) {
	inputDir := t.TempDir()
	inputPath := filepath.Join(inputDir, "failures.jsonl")
	if err := os.WriteFile(inputPath, []byte("{\"source\":\"semantic-scholar\",\"query\":\"fixture query\",\"error\":\"timeout\"}\n"), 0o644); err != nil {
		t.Fatalf("write resume input: %v", err)
	}
	out := t.TempDir()
	resultsPath := filepath.Join(out, "results.jsonl")
	failuresPath := filepath.Join(out, "failures.jsonl")
	priorResults := []byte("{\"sentinel\":\"prior resume results\"}\n")
	priorFailures := []byte("{\"sentinel\":\"prior resume failures\"}\n")
	if err := os.WriteFile(resultsPath, priorResults, 0o640); err != nil {
		t.Fatalf("write prior resume results: %v", err)
	}
	if err := os.WriteFile(failuresPath, priorFailures, 0o640); err != nil {
		t.Fatalf("write prior resume failures: %v", err)
	}
	rawConflict := filepath.Join(out, "raw", "search-semantic-scholar-001-fixture-query.txt")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := os.MkdirAll(rawConflict, 0o750); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"total":1,"offset":0,"data":[{"paperId":"txn","title":"Transactional fixture","year":2026,"authors":[{"name":"Fixture"}],"abstract":"test"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"search", "resume", "--failures", inputPath, "--out", out}, stdout, stderr)
	if code == 0 {
		t.Fatalf("Execute succeeded despite a commit-time output conflict; stdout=%s", stdout.String())
	}
	for path, want := range map[string][]byte{resultsPath: priorResults, failuresPath: priorFailures} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read restored resume artifact %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("restored resume artifact %s = %q, want %q", path, got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "raw")); !os.IsNotExist(err) {
		t.Fatalf("failed resume commit left raw artifacts: %v", err)
	}
}
