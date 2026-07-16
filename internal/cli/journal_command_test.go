package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestJournalReadHandlesLargeAppendedEntry(t *testing.T) {
	projectPath := t.TempDir()
	entry := strings.Repeat("large journal entry ", 8192)

	if code := Execute([]string{"--project", projectPath, "journal", "append", "--entry", entry}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("append exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--project", projectPath, "journal", "read"}, stdout, stderr); code != 0 {
		t.Fatalf("read exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout.String(), strings.TrimSpace(entry)) {
		t.Fatalf("journal read omitted the large entry")
	}
}

func TestJournalAppendDoesNotWriteThroughSymlinkedJournal(t *testing.T) {
	projectPath := t.TempDir()
	journalDir := filepath.Join(projectPath, "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	outsideBefore := []byte("outside journal must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside journal: %v", err)
	}
	journalPath := filepath.Join(journalDir, "entries.jsonl")
	if err := os.Symlink(outsidePath, journalPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", projectPath, "--json", "journal", "append", "--entry", "private observation"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("journal append succeeded with a symlinked journal: stdout=%s", stdout.String())
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside journal: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("journal append wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(journalPath)
	if lstatErr != nil {
		t.Fatalf("lstat journal: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("journal append replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestJournalAppendDoesNotWriteThroughHardLinkedJournal(t *testing.T) {
	projectPath := t.TempDir()
	journalDir := filepath.Join(projectPath, "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	outsideBefore := []byte("{\"timestamp\":\"2026-07-01T00:00:00Z\",\"entry\":\"outside observation\"}\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside journal: %v", err)
	}
	journalPath := filepath.Join(journalDir, "entries.jsonl")
	if err := os.Link(outsidePath, journalPath); err != nil {
		t.Skipf("hard links are unavailable: %v", err)
	}
	fixedTime := time.Unix(1_600_000_000, 0)
	if err := os.Chtimes(outsidePath, fixedTime, fixedTime); err != nil {
		t.Fatalf("set outside journal timestamps: %v", err)
	}
	before, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatalf("stat outside journal before append: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", projectPath, "journal", "append", "--entry", "private observation"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("journal append code = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside journal: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("outside journal changed: got %q, want %q", outsideAfter, outsideBefore)
	}
	after, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatalf("stat outside journal after append: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("outside journal mtime changed: got %s, want %s", after.ModTime(), before.ModTime())
	}
	journalBytes, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("read project journal: %v", err)
	}
	if !bytes.Contains(journalBytes, []byte("outside observation")) || !bytes.Contains(journalBytes, []byte("private observation")) {
		t.Fatalf("project journal does not contain both entries: %s", journalBytes)
	}
}

func TestJournalAppendRejectsSymlinkedJournalDirectory(t *testing.T) {
	projectPath := t.TempDir()
	outsideDir := t.TempDir()
	journalDir := filepath.Join(projectPath, "journal")
	if err := os.Symlink(outsideDir, journalDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", projectPath, "--json", "journal", "append", "--entry", "private observation"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("journal append succeeded through a symlinked directory: stdout=%s", stdout.String())
	}
	files, err := os.ReadDir(outsideDir)
	if err != nil {
		t.Fatalf("read outside directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("journal append wrote through symlinked directory: entries=%v", files)
	}
	info, err := os.Lstat(journalDir)
	if err != nil {
		t.Fatalf("lstat journal directory: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("journal append replaced directory symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestJournalReadShowsEntries(t *testing.T) {
	projectPath := t.TempDir()
	journalPath := filepath.Join(projectPath, "journal", "entries.jsonl")

	if code := Execute([]string{"--project", projectPath, "journal", "append", "--entry", "first observation"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("first append exit code = %d", code)
	}
	if err := os.Chmod(journalPath, 0o600); err != nil {
		t.Fatalf("chmod journal: %v", err)
	}
	if code := Execute([]string{"--project", projectPath, "journal", "append", "--entry", "second observation"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("second append exit code = %d", code)
	}
	info, err := os.Stat(journalPath)
	if err != nil {
		t.Fatalf("stat journal: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("journal mode = %o, want 600", got)
	}
	files, err := os.ReadDir(filepath.Dir(journalPath))
	if err != nil {
		t.Fatalf("read journal directory: %v", err)
	}
	if len(files) != 1 || files[0].Name() != "entries.jsonl" {
		t.Fatalf("journal directory entries = %v, want only entries.jsonl", files)
	}

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

func TestJournalReadDoesNotReadThroughSymlinkedJournal(t *testing.T) {
	projectPath := t.TempDir()
	journalDir := filepath.Join(projectPath, "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.WriteFile(outsidePath, []byte("{\"timestamp\":\"2026-01-01T00:00:00Z\",\"entry\":\"outside private observation\"}\n"), 0o640); err != nil {
		t.Fatalf("write outside journal: %v", err)
	}
	journalPath := filepath.Join(journalDir, "entries.jsonl")
	if err := os.Symlink(outsidePath, journalPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", projectPath, "--json", "journal", "read"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("journal read succeeded through a symlink: stdout=%s", stdout.String())
	}
	info, lstatErr := os.Lstat(journalPath)
	if lstatErr != nil {
		t.Fatalf("lstat journal: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("journal read replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestJournalReadReportsMalformedEntry(t *testing.T) {
	projectPath := t.TempDir()
	journalDir := filepath.Join(projectPath, "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal directory: %v", err)
	}
	journalPath := filepath.Join(journalDir, "entries.jsonl")
	malformed := []byte("{\"timestamp\":\"2026-01-01T00:00:00Z\",\"entry\":\"valid\"}\n{not valid JSON}\n")
	if err := os.WriteFile(journalPath, malformed, 0o600); err != nil {
		t.Fatalf("write malformed journal: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", projectPath, "--json", "journal", "read"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("journal read succeeded with a malformed entry: stdout=%s", stdout.String())
	}
	got, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("read malformed journal after command: %v", readErr)
	}
	if !bytes.Equal(got, malformed) {
		t.Fatalf("journal read changed malformed journal: got %q, want %q", got, malformed)
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
