package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchBatchPubMedErrorRedaction(t *testing.T) {
	const key = "synthetic+pubmed/key=with spaces&percent%"
	const query = "pubmed failure fixture"
	server := newJobsTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entrez/eutils/esearch.fcgi" || r.URL.Query().Get("api_key") != key || r.URL.Query().Get("term") != query {
			t.Error("batch request lost endpoint, query or configured API key")
		}
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack fixture connection: %v", err)
			return
		}
		_ = connection.Close() // Deterministic EOF without an external request or a sleep.
	}))
	t.Setenv("RFORGE_PUBMED_URL", server.URL)
	t.Setenv("RFORGE_PUBMED_API_KEY", key)
	t.Setenv("RFORGE_PUBMED_TOOL", "research-forge-tests")
	t.Setenv("RFORGE_PUBMED_EMAIL", "dev@example.org")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0")

	out := t.TempDir()
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"search", "batch", "--query", query, "--sources", "pubmed", "--out", out, "--continue-on-error"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}

	artifacts := map[string][]byte{}
	for _, name := range []string{"failures.jsonl", "batch-state.json", "results.md"} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		artifacts[name] = data
		for _, secret := range []string{key, url.QueryEscape(key)} {
			if bytes.Contains(data, []byte(secret)) {
				t.Errorf("%s exposes the configured PubMed key: %s", name, data)
			}
		}
		if !bytes.Contains(data, []byte("pubmed")) || !bytes.Contains(data, []byte(query)) || !bytes.Contains(data, []byte("EOF")) {
			t.Errorf("%s lost source/query/cause context: %s", name, data)
		}
	}
	var failure searchBatchFailure
	if err := json.Unmarshal(bytes.TrimSpace(artifacts["failures.jsonl"]), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Source != "pubmed" || failure.Query != query || failure.Skipped {
		t.Fatalf("failure identity = %#v", failure)
	}
	var state searchBatchState
	if err := json.Unmarshal(artifacts["batch-state.json"], &state); err != nil {
		t.Fatal(err)
	}
	if state.RequestsUsed != 1 || len(state.Failures) != 1 || state.Failures[0] != failure || len(state.Completed) != 0 {
		t.Fatalf("checkpoint lost failed-pair accounting: %#v", state)
	}
	manifestData, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest searchBatchManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Failures != 1 || manifest.Skipped != 0 || manifest.RequestsUsed != 1 || manifest.Results != 0 {
		t.Fatalf("manifest lost failed-pair accounting: %#v", manifest)
	}
	entries, err := os.ReadDir(filepath.Join(out, "raw"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed request wrote raw results: %v, %v", entries, err)
	}
}
