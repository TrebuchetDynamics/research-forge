package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteReportBuildIncludesParsedPassageProvenance(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Trace Report"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	parsed := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", Text: "Passage text."}}}}})
	parsedPath := filepath.Join(project, "parsed", "paper-1.json")
	if err := os.MkdirAll(filepath.Dir(parsedPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeJSONForCLITest(t, parsedPath, parsed)
	out := filepath.Join(project, "report.md")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "report", "build", "--out", out, "--parsed", parsedPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("report build code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	body := string(data)
	for _, want := range []string{"## Passage provenance", "grobid", "0.8", "parsed/paper-1.json#p1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("report missing %q:\n%s", want, body)
		}
	}
}
