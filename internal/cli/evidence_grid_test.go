package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteEvidenceGridWritesExtractionGrid(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Evidence Grid"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	item := evidence.EvidenceItem{PaperID: "paper-1", SchemaName: "outcomes", Values: map[string]string{"mean_treatment": "10"}, Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "passage-1"}, Status: evidence.StatusAccepted, History: []evidence.CorrectionEvent{{Status: evidence.StatusAccepted, Reviewer: "ada", Note: "checked"}}}
	writeJSONForCLITest(t, filepath.Join(project, "data", "evidence.items.json"), []evidence.EvidenceItem{item})
	parsed := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "passage-1", PaperID: "paper-1", Text: "Treatment mean was 10."}}}}})
	parsedPath := filepath.Join(project, "parsed", "paper-1.json")
	if err := os.MkdirAll(filepath.Dir(parsedPath), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeJSONForCLITest(t, parsedPath, parsed)
	analysisPath := filepath.Join(project, "analysis", "run1.json")
	if err := os.MkdirAll(filepath.Dir(analysisPath), 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	writeJSONForCLITest(t, analysisPath, analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}})
	out := filepath.Join(project, "data", "evidence-grid.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "grid", "--parsed", parsedPath, "--analysis", analysisPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("grid code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var grid evidence.ExtractionGrid
	if err := readJSONFile(out, &grid); err != nil {
		t.Fatalf("read grid: %v", err)
	}
	if len(grid.Rows) != 1 || grid.Rows[0].ParserName != "grobid" || !grid.Rows[0].DownstreamAnalysisIncluded || grid.Rows[0].PDFViewURL != "/papers/paper-1/pdf#passage-1" {
		t.Fatalf("grid = %#v", grid)
	}
}

func TestExecuteEvidenceGridRejectsMalformedEvidenceStore(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Evidence Grid"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.WriteFile(evidenceItemsPath(project), []byte(`[{"PaperID":`), 0o644); err != nil {
		t.Fatalf("write malformed evidence: %v", err)
	}
	out := filepath.Join(project, "data", "evidence-grid.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "grid", "--out", out}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("grid code=%d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"evidence_read_failed"`) {
		t.Fatalf("grid did not report evidence read failure: %s", stdout.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("grid wrote output after evidence read failure: %v", err)
	}
}
