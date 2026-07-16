package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
	code := Execute([]string{"--json", "--project", project, "evidence", "gaps", "--question", "Does treatment reduce hospitalization?", "--screened-in", "p1", "--screened-in", "p2", "--parsed-paper", "p1", "--outcome", "hospitalization", "--comparator", "standard care", "--full-text", "p1", "--claims", claimsPath, "--analysis", analysisPath, "--out", out}, &stdout, &stderr)
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
	if !hasEvidenceGapCode(report, "screened_in_missing_evidence") || !hasEvidenceGapCode(report, "screened_in_missing_parsed_passages") || !hasEvidenceGapCode(report, "question_term_missing_evidence") {
		t.Fatalf("report missing cross-check gaps = %#v", report.Gaps)
	}
}

func TestExecuteEvidenceGapsRejectsMalformedEvidenceStore(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Gaps"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.WriteFile(evidenceItemsPath(project), []byte(`[{"PaperID":`), 0o644); err != nil {
		t.Fatalf("write malformed evidence: %v", err)
	}
	out := filepath.Join(project, "data", "evidence-gaps.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "gaps", "--out", out}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("gaps code=%d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"evidence_read_failed"`) {
		t.Fatalf("gaps did not report evidence read failure: %s", stdout.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("gaps wrote report after evidence read failure: %v", err)
	}
}

func hasEvidenceGapCode(report evidence.EvidenceGapReport, code string) bool {
	for _, gap := range report.Gaps {
		if gap.Code == code {
			return true
		}
	}
	return false
}
