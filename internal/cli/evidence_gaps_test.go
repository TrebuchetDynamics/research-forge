package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestExecuteEvidenceGapsWritesGapReport(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Gaps"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "data", "evidence.items.json"), []evidence.EvidenceItem{{PaperID: "p1", Values: map[string]string{"outcome": "mortality", "comparator": "placebo"}, Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "p1:p1"}, Status: evidence.StatusAccepted}})
	claimsPath := filepath.Join(project, "data", "citation-locked.json")
	writeJSONForCLITest(t, claimsPath, evidence.CitationLockedSuggestionQueue{SchemaVersion: "1", PaperID: "p1", Suggestions: []evidence.CitationLockedSuggestion{{ID: "claim-1", PaperID: "p1", Status: evidence.StatusSuggested}}})
	analysisPath := filepath.Join(project, "analysis.json")
	writeJSONForCLITest(t, analysisPath, analysis.AnalysisRun{SchemaVersion: "1", ID: "run1"})
	out := filepath.Join(project, "data", "evidence-gaps.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "gaps", "--outcome", "hospitalization", "--comparator", "standard care", "--full-text", "p1", "--claims", claimsPath, "--analysis", analysisPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("gaps code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report evidence.EvidenceGapReport
	if err := readJSONFile(out, &report); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if report.ReadyForAnalysis || len(report.Gaps) == 0 {
		t.Fatalf("report = %#v", report)
	}
}
