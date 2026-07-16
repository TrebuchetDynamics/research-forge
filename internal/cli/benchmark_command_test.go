package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteBenchmarkCrossToolWritesReport(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "cross-tool.json")
	if err := os.WriteFile(out, []byte("prior benchmark\n"), 0o600); err != nil {
		t.Fatalf("write prior benchmark: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "benchmark", "cross-tool", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	var report struct {
		Metrics []struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("report json: %v", err)
	}
	if len(report.Metrics) != 7 {
		t.Fatalf("metrics = %#v", report.Metrics)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("cross-tool")) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat out: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("output mode = %o, want 600", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("benchmark left transaction debris: %s", entry.Name())
		}
	}
}

func TestExecuteBenchmarkCrossToolDoesNotWriteThroughSymlinkedOutput(t *testing.T) {
	dir := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	outsideBefore := []byte("outside benchmark must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside benchmark: %v", err)
	}
	out := filepath.Join(dir, "cross-tool.json")
	if err := os.Symlink(outsidePath, out); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "benchmark", "cross-tool", "--out", out}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("benchmark succeeded with symlinked output: stdout=%s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside benchmark: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("benchmark wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(out)
	if err != nil {
		t.Fatalf("lstat benchmark output: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("benchmark replaced output symlink despite rejecting it: mode=%v", info.Mode())
	}
}
