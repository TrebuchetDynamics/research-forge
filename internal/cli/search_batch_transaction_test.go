package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteSearchBatchPreservesPriorArtifactsWhenOutputInitializationFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-TXN","doi":"https://doi.org/10.0000/txn","title":"Transactional fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	resultsPath := filepath.Join(out, "results.jsonl")
	priorResults := []byte("{\"sentinel\":\"prior results\"}\n")
	if err := os.WriteFile(resultsPath, priorResults, 0o640); err != nil {
		t.Fatalf("write prior results: %v", err)
	}
	if err := os.Mkdir(filepath.Join(out, "failures.jsonl"), 0o750); err != nil {
		t.Fatalf("create conflicting failures output: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"search", "batch", "--query", "fixture query", "--sources", "openalex", "--out", out}, stdout, stderr)
	if code == 0 {
		t.Fatalf("Execute succeeded with a directory at failures.jsonl; stdout=%s", stdout.String())
	}
	results, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read prior results after failure: %v", err)
	}
	if !bytes.Equal(results, priorResults) {
		t.Fatalf("results after failure = %q, want %q", results, priorResults)
	}
	if info, err := os.Stat(filepath.Join(out, "failures.jsonl")); err != nil || !info.IsDir() {
		t.Fatalf("conflicting failures output changed after failure: info=%v err=%v", info, err)
	}
	if _, err := os.Stat(filepath.Join(out, "raw")); !os.IsNotExist(err) {
		t.Fatalf("failed batch left a new raw directory: %v", err)
	}
}

func TestExecuteSearchBatchReplacesArtifactsAndPreservesUnrelatedFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-TXN","doi":"https://doi.org/10.0000/txn","title":"Transactional fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	priorResults := []byte("{\"sentinel\":\"prior results\"}\n")
	if err := os.WriteFile(filepath.Join(out, "results.jsonl"), priorResults, 0o640); err != nil {
		t.Fatalf("write prior results: %v", err)
	}
	unrelated := []byte("keep this file\n")
	if err := os.WriteFile(filepath.Join(out, "notes.txt"), unrelated, 0o640); err != nil {
		t.Fatalf("write unrelated output file: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"search", "batch", "--query", "fixture query", "--sources", "openalex", "--out", out}, stdout, stderr)
	if code != 0 {
		t.Fatalf("Execute code = %d; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	results, err := os.ReadFile(filepath.Join(out, "results.jsonl"))
	if err != nil {
		t.Fatalf("read replacement results: %v", err)
	}
	if bytes.Equal(results, priorResults) || !bytes.Contains(results, []byte("Transactional fixture")) {
		t.Fatalf("replacement results = %q", results)
	}
	gotUnrelated, err := os.ReadFile(filepath.Join(out, "notes.txt"))
	if err != nil {
		t.Fatalf("read unrelated output file: %v", err)
	}
	if !bytes.Equal(gotUnrelated, unrelated) {
		t.Fatalf("unrelated output file = %q, want %q", gotUnrelated, unrelated)
	}
}

func TestExecuteSearchBatchRestoresAllArtifactsWhenCommitFails(t *testing.T) {
	out := t.TempDir()
	priorResults := []byte("{\"sentinel\":\"prior results\"}\n")
	priorManifest := []byte("{\"sentinel\":\"prior manifest\"}\n")
	type linkedArtifact struct {
		outputPath  string
		outsidePath string
		data        []byte
		before      os.FileInfo
	}
	linkArtifact := func(name string, data []byte) linkedArtifact {
		t.Helper()
		outsidePath := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(outsidePath, data, 0o640); err != nil {
			t.Fatalf("write outside %s: %v", name, err)
		}
		outputPath := filepath.Join(out, name)
		if err := os.Link(outsidePath, outputPath); err != nil {
			t.Skipf("hard links are unavailable: %v", err)
		}
		fixedTime := time.Unix(1_600_000_000, 0)
		if err := os.Chtimes(outsidePath, fixedTime, fixedTime); err != nil {
			t.Fatalf("set outside %s timestamps: %v", name, err)
		}
		before, err := os.Stat(outsidePath)
		if err != nil {
			t.Fatalf("stat outside %s before Execute: %v", name, err)
		}
		return linkedArtifact{outputPath: outputPath, outsidePath: outsidePath, data: data, before: before}
	}
	artifacts := []linkedArtifact{
		linkArtifact("results.jsonl", priorResults),
		linkArtifact("manifest.json", priorManifest),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := os.Mkdir(filepath.Join(out, "results.md"), 0o750); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-TXN","doi":"https://doi.org/10.0000/txn","title":"Transactional fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"search", "batch", "--query", "fixture query", "--sources", "openalex", "--out", out}, stdout, stderr)
	if code == 0 {
		t.Fatalf("Execute succeeded despite a commit-time output conflict; stdout=%s", stdout.String())
	}
	for _, artifact := range artifacts {
		restored, err := os.ReadFile(artifact.outputPath)
		if err != nil {
			t.Fatalf("read restored %s: %v", artifact.outputPath, err)
		}
		if !bytes.Equal(restored, artifact.data) {
			t.Fatalf("restored %s = %q, want %q", artifact.outputPath, restored, artifact.data)
		}
		outside, err := os.ReadFile(artifact.outsidePath)
		if err != nil {
			t.Fatalf("read outside %s: %v", artifact.outsidePath, err)
		}
		if !bytes.Equal(outside, artifact.data) {
			t.Fatalf("outside %s = %q, want %q", artifact.outsidePath, outside, artifact.data)
		}
		after, err := os.Stat(artifact.outsidePath)
		if err != nil {
			t.Fatalf("stat outside %s after Execute: %v", artifact.outsidePath, err)
		}
		if !after.ModTime().Equal(artifact.before.ModTime()) {
			t.Fatalf("outside %s mtime changed: got %s, want %s", artifact.outsidePath, after.ModTime(), artifact.before.ModTime())
		}
	}
	for _, relativePath := range []string{"results-deduped.jsonl", "failures.jsonl", "results.md", "raw"} {
		if _, err := os.Stat(filepath.Join(out, relativePath)); !os.IsNotExist(err) {
			t.Errorf("failed commit left %s: %v", relativePath, err)
		}
	}
}
