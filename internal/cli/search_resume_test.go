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

func TestSearchResumeRunsFailedQueries(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":1,"offset":0,"data":[{"paperId":"abc","title":"Fractal IFS","year":2020,"authors":[{"name":"Smith"}],"abstract":"test"}]}`))
	}))
	defer server.Close()

	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)

	prevOut := t.TempDir()
	writeSearchFailures(t, prevOut, `{"source":"semantic-scholar","query":"fractal IFS","error":"rate limited"}
{"source":"semantic-scholar","query":"chaos attractors","error":"timeout"}
`)

	newOut := t.TempDir()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"search", "resume", "--failures", filepath.Join(prevOut, "failures.jsonl"), "--out", newOut}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if requests < 2 {
		t.Errorf("requests = %d, want >= 2 (one per failed query)", requests)
	}
	if !strings.Contains(stdout.String(), "resumed") && !strings.Contains(stdout.String(), "2") {
		t.Errorf("stdout = %q, want mention of resumed queries", stdout.String())
	}
}

func TestSearchResumeRequiresFailuresFile(t *testing.T) {
	code := Execute([]string{"search", "resume", "--out", t.TempDir()}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestSearchResumeRequiresOutDir(t *testing.T) {
	dir := t.TempDir()
	writeSearchFailures(t, dir, `{"source":"semantic-scholar","query":"test","error":"timeout"}`)
	code := Execute([]string{"search", "resume", "--failures", filepath.Join(dir, "failures.jsonl")}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestSearchResumeEmptyFailuresFileSucceeds(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "failures.jsonl"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	code := Execute([]string{"search", "resume", "--failures", filepath.Join(dir, "failures.jsonl"), "--out", t.TempDir()}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Errorf("exit code = %d, want 0 for empty failures", code)
	}
}
