package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchStatsShowsFailedQueries(t *testing.T) {
	dir := t.TempDir()
	writeSearchFailures(t, dir, `{"source":"semantic-scholar","query":"fractal chaos","error":"rate limited: retry after 60s"}
{"source":"openalex","query":"IFS attractor","error":"timeout after 10s"}
`)

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "semantic-scholar") {
		t.Errorf("stdout missing source 'semantic-scholar':\n%s", out)
	}
	if !strings.Contains(out, "fractal chaos") {
		t.Errorf("stdout missing query 'fractal chaos':\n%s", out)
	}
	if !strings.Contains(out, "rate limited") {
		t.Errorf("stdout missing error text 'rate limited':\n%s", out)
	}
	if !strings.Contains(out, "IFS attractor") {
		t.Errorf("stdout missing query 'IFS attractor':\n%s", out)
	}
}

func TestSearchStatsJSONIncludesFailures(t *testing.T) {
	dir := t.TempDir()
	writeSearchFailures(t, dir, `{"source":"semantic-scholar","query":"fractal","error":"rate limited"}
`)

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	failures, ok := data["failures"]
	if !ok {
		t.Fatalf("JSON data missing 'failures' key; data = %v", data)
	}
	list, _ := failures.([]any)
	if len(list) != 1 {
		t.Fatalf("failures len = %d, want 1", len(list))
	}
}

func TestSearchStatsNoFailuresFileShowsNone(t *testing.T) {
	dir := t.TempDir()
	// No failures.jsonl — command should succeed and not print a failures section.
	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if strings.Contains(stdout.String(), "failed") {
		t.Errorf("stdout should not mention failures when failures.jsonl absent:\n%s", stdout.String())
	}
}

func writeSearchFailures(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "failures.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatalf("write failures.jsonl: %v", err)
	}
}
