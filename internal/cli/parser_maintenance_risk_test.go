package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteParseMaintenanceRiskWritesScienceParseGate(t *testing.T) {
	out := filepath.Join(t.TempDir(), "parser-risk.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", t.TempDir(), "parse", "maintenance-risk", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	var report struct {
		Parsers []struct {
			ParserName     string `json:"parserName"`
			ReviewerGate   string `json:"reviewerGate"`
			EnableFallback bool   `json:"enableFallback"`
		} `json:"parsers"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("json: %v", err)
	}
	found := false
	for _, parser := range report.Parsers {
		if parser.ParserName == "science-parse" {
			found = true
			if parser.EnableFallback || parser.ReviewerGate != "maintenance-risk-review" {
				t.Fatalf("science-parse parser = %#v", parser)
			}
		}
	}
	if !found {
		t.Fatalf("science-parse missing: %#v", report)
	}
}
