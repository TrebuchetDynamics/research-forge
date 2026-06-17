package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/report"
)

func TestExecuteReportTraceWritesCitationEvidenceTraceView(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Trace"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "data", "evidence.items.json"), []evidence.EvidenceItem{{PaperID: "paper-1", Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "passage-1"}, Status: evidence.StatusAccepted}})
	claimsPath := filepath.Join(project, "data", "claims.json")
	writeJSONForCLITest(t, claimsPath, evidence.CitationLockedSuggestionQueue{SchemaVersion: "1", PaperID: "paper-1", Suggestions: []evidence.CitationLockedSuggestion{{ID: "claim-1", PaperID: "paper-1", Status: evidence.StatusAccepted, CitationLocks: []evidence.CitationLockedSupport{{Ref: "passage-1", ExactText: "quoted"}}}}})
	analysisPath := filepath.Join(project, "analysis", "run1.json")
	if err := os.MkdirAll(filepath.Dir(analysisPath), 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	writeJSONForCLITest(t, analysisPath, analysis.AnalysisRun{ID: "run1", InputRows: []analysis.InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}})
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeJSONForCLITest(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{Passages: []parsing.Passage{{ID: "passage-1", PaperID: "paper-1", Text: "quoted"}}}}})
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: "Paper", Identifiers: library.Identifiers{DOI: "paper-1", ZoteroItemKey: "ZOT-1"}}); err != nil {
		t.Fatalf("seed library: %v", err)
	}
	out := filepath.Join(project, "data", "trace.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "report", "trace", "--claims", claimsPath, "--analysis", analysisPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("trace code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var view report.CitationEvidenceTraceView
	if err := readJSONFile(out, &view); err != nil {
		t.Fatalf("read trace: %v", err)
	}
	if len(view.Claims) != 1 || len(view.Claims[0].EffectSizeRows) != 1 || len(view.Claims[0].AcceptedEvidence) != 1 || len(view.Claims[0].Passages) != 1 || view.Claims[0].Passages[0].ParserName != "grobid" || view.Claims[0].ReferenceManagerItems[0] != "zotero:ZOT-1" {
		t.Fatalf("view = %#v", view)
	}
}
