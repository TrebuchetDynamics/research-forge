package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/report"
)

func TestExecuteReportClaimPanelBlocksFinalExport(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Panel"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	trace := report.CitationEvidenceTraceView{SchemaVersion: "1", Claims: []report.ClaimTraceView{
		{ClaimID: "weak", PaperID: "p1", ClaimText: "generated paragraph", ClaimStatus: evidence.StatusAccepted},
	}}
	tracePath := filepath.Join(project, "data", "trace.json")
	writeJSONForCLITest(t, tracePath, trace)
	panelPath := filepath.Join(project, "data", "claim-panel.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "report", "claim-panel", "--trace", tracePath, "--out", panelPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("claim-panel code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var panel report.ClaimTraceabilityPanel
	if err := readJSONFile(panelPath, &panel); err != nil {
		t.Fatalf("read panel: %v", err)
	}
	if !panel.BlockFinalExport || panel.Rows[0].Status != report.ClaimTraceWeakSupport {
		t.Fatalf("panel = %#v", panel)
	}
	input := filepath.Join(project, "report.md")
	if err := os.WriteFile(input, []byte("# Report"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "report", "final-export", "--in", input, "--panel", panelPath, "--out", filepath.Join(project, "final.md")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected final export block, stdout=%s", stdout.String())
	}
	readyTrace := report.CitationEvidenceTraceView{SchemaVersion: "1", Claims: []report.ClaimTraceView{{ClaimID: "ok", PaperID: "p1", ClaimText: "generated paragraph", ClaimStatus: evidence.StatusAccepted, EffectSizeRows: []analysis.InputRow{{PaperID: "p1"}}, AcceptedEvidence: []evidence.EvidenceItem{{PaperID: "p1", Status: evidence.StatusAccepted}}, Passages: []report.TracePassage{{PassageID: "p1"}}}}}
	readyPanel := report.BuildClaimTraceabilityPanel(readyTrace)
	writeJSONForCLITest(t, panelPath, readyPanel)
	stdout.Reset()
	stderr.Reset()
	out := filepath.Join(project, "final.md")
	code = Execute([]string{"--json", "--project", project, "report", "final-export", "--in", input, "--panel", panelPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("final export code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("missing final export: %v", err)
	}
}
