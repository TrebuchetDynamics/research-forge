package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteParseQualityWritesReviewerGatedReport(t *testing.T) {
	dir := t.TempDir()
	left := filepath.Join(dir, "grobid.json")
	right := filepath.Join(dir, "s2orc.json")
	out := filepath.Join(dir, "quality.json")
	writeJSONFixture(t, left, parsing.ParsedDocument{PaperID: "p1", ParserName: "grobid", Title: "A", ParserConfidence: 0.9, References: []parsing.Reference{{DOI: "10.1/ref", Confidence: 0.9}}})
	writeJSONFixture(t, right, parsing.ParsedDocument{PaperID: "p1", ParserName: "s2orc-doc2json", Title: "B", ParserConfidence: 0.5, Warnings: []string{"low confidence"}})
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "parse", "quality", "--parsed", left, "--parsed", right, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	var report parsing.ParserQualityReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("json: %v", err)
	}
	if !report.ReviewerRequired || report.AutoAcceptedFields || len(report.Conflicts) == 0 || !report.HasParser("grobid") || !report.HasParser("s2orc") {
		t.Fatalf("report = %#v", report)
	}
	if !strings.Contains(stdout.String(), "quality") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}
