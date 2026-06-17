package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteParseArbitrateWritesDecisionReport(t *testing.T) {
	project := t.TempDir()
	left := filepath.Join(project, "left.json")
	right := filepath.Join(project, "right.json")
	out := filepath.Join(project, "arbitration.json")
	writeParsedFixture(t, left, parsing.ParsedDocument{SchemaVersion: "1", PaperID: "paper-1", ParserName: "grobid", Title: "Title", Abstract: "Abstract", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", Text: "text"}}}}})
	writeParsedFixture(t, right, parsing.ParsedDocument{SchemaVersion: "1", PaperID: "paper-1", ParserName: "s2orc", Title: "Title", Warnings: []string{"missing abstract"}})
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "parse", "arbitrate", "--left", left, "--right", right, "--out", out, "--accept", "grobid", "--reason", "best field coverage", "--reviewer", "reviewer-a"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	var report parsing.ParserArbitrationReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(data))
	}
	if report.Decision.AcceptedParser != "grobid" || report.Decision.Reason != "best field coverage" || len(report.FieldScores["abstract"]) != 2 || len(report.Comparisons) == 0 || len(report.WarningComparison) != 2 {
		t.Fatalf("report = %#v", report)
	}

	multiOut := filepath.Join(project, "multi-arbitration.json")
	paperMage := filepath.Join(project, "papermage.json")
	writeParsedFixture(t, paperMage, parsing.ParsedDocument{SchemaVersion: "1", PaperID: "paper-1", ParserName: "papermage", Title: "Different", Abstract: "Layered"})
	code = Execute([]string{"--json", "--project", project, "parse", "arbitrate", "--parsed", left, "--parsed", right, "--parsed", paperMage, "--out", multiOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("multi code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var multi parsing.ParserArbitrationReport
	if err := readJSONFile(multiOut, &multi); err != nil {
		t.Fatalf("read multi: %v", err)
	}
	if len(multi.FieldScores["title"]) != 3 || len(multi.ConflictReviewQueue) == 0 {
		t.Fatalf("multi report = %#v", multi)
	}
}

func writeParsedFixture(t *testing.T, path string, doc parsing.ParsedDocument) {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
