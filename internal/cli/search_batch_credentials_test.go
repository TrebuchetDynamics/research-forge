package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchBatchRejectsMissingProjectBeforeCheckingCredentials(t *testing.T) {
	t.Setenv("RFORGE_LENS_TOKEN", "")
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)

	code := Execute([]string{
		"--json", "search", "batch", "--query", "test", "--sources", "lens",
		"--out", filepath.Join(t.TempDir(), "out"), "--fetch-pdfs",
	}, stdout, stderr)
	if code != 2 || !strings.Contains(stdout.String(), "missing_project") {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
}

func TestSearchBatchRejectsUnknownSourceBeforeCreatingOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)

	code := Execute([]string{
		"--json", "search", "batch", "--query", "first", "--query", "second",
		"--sources", "not-a-source", "--out", out, "--continue-on-error",
	}, stdout, stderr)
	if code != 2 || !strings.Contains(stdout.String(), "unknown_source") {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output created for invalid source: %v", err)
	}
}

func TestSearchBatchRecordsMissingCredentialOncePerSource(t *testing.T) {
	for _, env := range []string{"RFORGE_LENS_TOKEN", "RFORGE_ADS_TOKEN", "RFORGE_DIMENSIONS_TOKEN"} {
		t.Setenv(env, "")
	}
	for _, env := range []string{"RFORGE_LENS_URL", "RFORGE_ADS_URL", "RFORGE_DIMENSIONS_URL"} {
		t.Setenv(env, "http://127.0.0.1:1")
	}

	out := t.TempDir()
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{
		"search", "batch",
		"--query", "first query",
		"--query", "second query",
		"--sources", "lens,nasa-ads,dimensions",
		"--out", out,
		"--continue-on-error",
		"--stats",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(out, "failures.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.FieldsFunc(strings.TrimSpace(string(data)), func(r rune) bool { return r == '\n' })
	if len(lines) != 3 {
		t.Fatalf("failure lines = %d, want one per source:\n%s", len(lines), data)
	}
	for _, line := range lines {
		var issue struct {
			Skipped bool `json:"skipped"`
		}
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			t.Fatal(err)
		}
		if !issue.Skipped {
			t.Fatalf("configuration issue is not marked skipped: %s", line)
		}
	}
	for _, env := range []string{"RFORGE_LENS_TOKEN", "RFORGE_ADS_TOKEN", "RFORGE_DIMENSIONS_TOKEN"} {
		if !strings.Contains(string(data), env) {
			t.Fatalf("failures do not name %s:\n%s", env, data)
		}
	}

	manifestData, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Failures int `json:"failures"`
		Skipped  int `json:"skipped"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Failures != 0 || manifest.Skipped != 3 {
		t.Fatalf("manifest failures/skipped = %d/%d, want 0/3", manifest.Failures, manifest.Skipped)
	}
	stats, err := os.ReadFile(filepath.Join(out, "search-stats.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stats), "Failures: 0") || !strings.Contains(string(stats), "Skipped sources: 3") {
		t.Fatalf("stats do not distinguish skips:\n%s", stats)
	}
}
