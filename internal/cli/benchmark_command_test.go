package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteBenchmarkCrossToolWritesReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "cross-tool.json")
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
}
