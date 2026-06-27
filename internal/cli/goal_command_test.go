package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoalSetCreatesGoalFile(t *testing.T) {
	projectPath := t.TempDir()

	stdout := new(bytes.Buffer)
	code := Execute([]string{
		"--project", projectPath, "goal", "set",
		"--metric", "counted_total",
		"--min", "5000",
		"--name", "reach 5000 papers",
	}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "goal set") {
		t.Errorf("stdout = %q, want 'goal set'", stdout.String())
	}
	data, err := os.ReadFile(filepath.Join(projectPath, "goals.json"))
	if err != nil {
		t.Fatalf("goals.json not created: %v", err)
	}
	var goals []any
	if err := json.Unmarshal(data, &goals); err != nil {
		t.Fatalf("goals.json is not valid JSON: %v", err)
	}
	if len(goals) == 0 {
		t.Fatal("goals.json is empty")
	}
}

func TestGoalAuditReportsProgress(t *testing.T) {
	projectPath := t.TempDir()

	// Set a goal first
	Execute([]string{
		"--project", projectPath, "goal", "set",
		"--metric", "counted_total",
		"--min", "10",
		"--name", "collect 10 papers",
	}, new(bytes.Buffer), new(bytes.Buffer))

	// Write a ledger file with 15 papers
	ledgerPath := filepath.Join(projectPath, "results-deduped.jsonl")
	content := ""
	for i := 0; i < 15; i++ {
		content += `{"doi":"10.1/` + string(rune('a'+i)) + `","title":"Paper"}` + "\n"
	}
	if err := os.WriteFile(ledgerPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	code := Execute([]string{
		"--project", projectPath, "goal", "audit",
		"--ledger", ledgerPath,
	}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "15") && !strings.Contains(out, "met") {
		t.Errorf("stdout = %q, want count or met status", out)
	}
}

func TestGoalAuditJSONOutput(t *testing.T) {
	projectPath := t.TempDir()

	Execute([]string{
		"--project", projectPath, "goal", "set",
		"--metric", "counted_total",
		"--min", "5",
		"--name", "collect 5 papers",
	}, new(bytes.Buffer), new(bytes.Buffer))

	ledgerPath := filepath.Join(projectPath, "results-deduped.jsonl")
	content := ""
	for i := 0; i < 3; i++ {
		content += `{"doi":"10.1/x","title":"P"}` + "\n"
	}
	os.WriteFile(ledgerPath, []byte(content), 0o644)

	stdout := new(bytes.Buffer)
	code := Execute([]string{
		"--project", projectPath, "--json", "goal", "audit",
		"--ledger", ledgerPath,
	}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if _, ok := data["met"]; !ok {
		t.Errorf("JSON data missing 'met' key: %v", data)
	}
	if _, ok := data["count"]; !ok {
		t.Errorf("JSON data missing 'count' key: %v", data)
	}
}

func TestGoalSetRequiresProject(t *testing.T) {
	code := Execute([]string{"goal", "set", "--metric", "counted_total", "--min", "5", "--name", "x"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestGoalAuditRequiresLedger(t *testing.T) {
	projectPath := t.TempDir()
	code := Execute([]string{"--project", projectPath, "goal", "audit"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}
