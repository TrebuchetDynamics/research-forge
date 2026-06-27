package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJournalAppendCreatesEntry(t *testing.T) {
	projectPath := t.TempDir()

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "journal", "append", "--entry", "tested Barnsley fern with 10k points"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "entry recorded") {
		t.Errorf("stdout = %q, want 'entry recorded'", stdout.String())
	}
}

func TestJournalReadShowsEntries(t *testing.T) {
	projectPath := t.TempDir()

	Execute([]string{"--project", projectPath, "journal", "append", "--entry", "first observation"}, new(bytes.Buffer), new(bytes.Buffer))
	Execute([]string{"--project", projectPath, "journal", "append", "--entry", "second observation"}, new(bytes.Buffer), new(bytes.Buffer))

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "journal", "read"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "first observation") {
		t.Errorf("stdout missing first entry: %s", out)
	}
	if !strings.Contains(out, "second observation") {
		t.Errorf("stdout missing second entry: %s", out)
	}
}

func TestJournalAppendJSONOutput(t *testing.T) {
	projectPath := t.TempDir()

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "--json", "journal", "append", "--entry", "noted"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["action"] != "journal.append" {
		t.Errorf("data.action = %v, want 'journal.append'", data["action"])
	}
}

func TestJournalAppendRequiresEntry(t *testing.T) {
	projectPath := t.TempDir()

	code := Execute([]string{"--project", projectPath, "journal", "append"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestJournalAppendRequiresProject(t *testing.T) {
	code := Execute([]string{"journal", "append", "--entry", "hello"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestJournalReadEmptyProjectReturnsNothing(t *testing.T) {
	projectPath := t.TempDir()

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "journal", "read"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
}
