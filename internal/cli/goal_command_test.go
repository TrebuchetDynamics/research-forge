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
	goalsPath := filepath.Join(projectPath, "goals.json")
	if err := os.Chmod(goalsPath, 0o600); err != nil {
		t.Fatalf("chmod goals.json: %v", err)
	}
	stdout.Reset()
	code = Execute([]string{
		"--project", projectPath, "goal", "set",
		"--metric", "counted_total",
		"--min", "10",
		"--name", "reach 10 papers",
	}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("second goal set exit code = %d", code)
	}
	data, err = os.ReadFile(goalsPath)
	if err != nil {
		t.Fatalf("read updated goals.json: %v", err)
	}
	goals = nil
	if err := json.Unmarshal(data, &goals); err != nil {
		t.Fatalf("updated goals.json is not valid JSON: %v", err)
	}
	if len(goals) != 2 {
		t.Fatalf("updated goals count = %d, want 2", len(goals))
	}
	info, err := os.Stat(goalsPath)
	if err != nil {
		t.Fatalf("stat goals.json: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("goals.json mode = %o, want 600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		t.Fatalf("read project directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "goals.json" {
		t.Fatalf("project directory entries = %#v, want only goals.json", entries)
	}
}

func TestGoalSetDoesNotWriteThroughSymlinkedGoalFile(t *testing.T) {
	projectPath := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside-goals.json")
	outsideBefore := []byte("[]\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside goals: %v", err)
	}
	goalsPath := filepath.Join(projectPath, "goals.json")
	if err := os.Symlink(outsidePath, goalsPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{
		"--project", projectPath, "--json", "goal", "set",
		"--metric", "counted_total",
		"--min", "5000",
		"--name", "reach 5000 papers",
	}, stdout, stderr)
	if code == 0 {
		t.Fatalf("goal set succeeded with a symlinked goals file: stdout=%s", stdout.String())
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside goals: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("goal set wrote through goals symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(goalsPath)
	if lstatErr != nil {
		t.Fatalf("lstat goals file: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("goal set replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestGoalAuditDoesNotReadThroughSymlinkedGoalFile(t *testing.T) {
	projectPath := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside-goals.json")
	outsideGoals := []byte(`[{"name":"external private goal","metric":"records","min":1}]`)
	if err := os.WriteFile(outsidePath, outsideGoals, 0o640); err != nil {
		t.Fatalf("write outside goals: %v", err)
	}
	goalsPath := filepath.Join(projectPath, "goals.json")
	if err := os.Symlink(outsidePath, goalsPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	ledgerPath := filepath.Join(projectPath, "results.jsonl")
	if err := os.WriteFile(ledgerPath, []byte("{}\n"), 0o640); err != nil {
		t.Fatalf("write goal audit ledger: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", projectPath, "--json", "goal", "audit", "--ledger", ledgerPath}, stdout, stderr)
	if code == 0 {
		t.Fatalf("goal audit read external goals through symlink: stdout=%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "external private goal") {
		t.Fatalf("goal audit exposed external goal: stdout=%s", stdout.String())
	}
	info, err := os.Lstat(goalsPath)
	if err != nil {
		t.Fatalf("lstat goals path: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("goal audit replaced symlink: mode=%v", info.Mode())
	}
}

func TestGoalSetDoesNotOverwriteMalformedGoalFile(t *testing.T) {
	projectPath := t.TempDir()
	goalsPath := filepath.Join(projectPath, "goals.json")
	malformed := []byte("{not valid goal JSON\n")
	if err := os.WriteFile(goalsPath, malformed, 0o600); err != nil {
		t.Fatalf("write malformed goals: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{
		"--project", projectPath, "--json", "goal", "set",
		"--metric", "counted_total",
		"--min", "5000",
		"--name", "reach 5000 papers",
	}, stdout, stderr)
	if code == 0 {
		t.Fatalf("goal set succeeded with malformed goals: stdout=%s", stdout.String())
	}
	got, readErr := os.ReadFile(goalsPath)
	if readErr != nil {
		t.Fatalf("read malformed goals after goal set: %v", readErr)
	}
	if !bytes.Equal(got, malformed) {
		t.Fatalf("goal set replaced malformed goals: got %q, want %q", got, malformed)
	}
	info, statErr := os.Stat(goalsPath)
	if statErr != nil {
		t.Fatalf("stat malformed goals: %v", statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("malformed goals mode = %o, want 600", info.Mode().Perm())
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

func TestGoalAuditCountsLargeLedgerRecord(t *testing.T) {
	projectPath := t.TempDir()
	if code := Execute([]string{
		"--project", projectPath, "goal", "set",
		"--metric", "counted_total",
		"--min", "1",
		"--name", "collect one record",
	}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("goal set exit code = %d", code)
	}
	ledgerPath := filepath.Join(projectPath, "results-deduped.jsonl")
	if err := os.WriteFile(ledgerPath, []byte(strings.Repeat("x", 128*1024)+"\n"), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "goal", "audit", "--ledger", ledgerPath}, stdout, stderr)
	if code != 0 {
		t.Fatalf("goal audit exit code = %d, stderr = %s", code, stderr)
	}
	if out := stdout.String(); !strings.Contains(out, "met") || !strings.Contains(out, "1 / 1") {
		t.Fatalf("stdout = %q, want met count", out)
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

func TestGoalAuditReportsMalformedGoalFile(t *testing.T) {
	projectPath := t.TempDir()
	goalsPath := filepath.Join(projectPath, "goals.json")
	malformed := []byte("{not valid goal JSON\n")
	if err := os.WriteFile(goalsPath, malformed, 0o600); err != nil {
		t.Fatalf("write malformed goals: %v", err)
	}
	ledgerPath := filepath.Join(projectPath, "results-deduped.jsonl")
	if err := os.WriteFile(ledgerPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{
		"--project", projectPath, "--json", "goal", "audit",
		"--ledger", ledgerPath,
	}, stdout, stderr)
	if code == 0 {
		t.Fatalf("goal audit succeeded with malformed goals: stdout=%s", stdout.String())
	}
	got, readErr := os.ReadFile(goalsPath)
	if readErr != nil {
		t.Fatalf("read malformed goals after audit: %v", readErr)
	}
	if !bytes.Equal(got, malformed) {
		t.Fatalf("goal audit changed malformed goals: got %q, want %q", got, malformed)
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
