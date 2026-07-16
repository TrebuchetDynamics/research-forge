package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
	out := filepath.Join(project, "exports", "nested", "final.md")
	code = Execute([]string{"--json", "--project", project, "report", "final-export", "--in", input, "--panel", panelPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("final export code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("missing final export: %v", err)
	}
	if err := os.Chmod(out, 0o600); err != nil {
		t.Fatalf("chmod final export: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "report", "final-export", "--in", input, "--panel", panelPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("final re-export code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat final export: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("final export mode = %o, want 600", got)
	}
	entries, err := os.ReadDir(filepath.Dir(out))
	if err != nil {
		t.Fatalf("read final export directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("final export left transaction debris: %s", entry.Name())
		}
	}
}

func TestExecuteReportFinalExportDoesNotWriteThroughSymlinkedOutput(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Panel"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	input := filepath.Join(project, "report.md")
	if err := os.WriteFile(input, []byte("# Ready report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	readyTrace := report.CitationEvidenceTraceView{SchemaVersion: "1", Claims: []report.ClaimTraceView{{ClaimID: "ok", PaperID: "p1", ClaimText: "generated paragraph", ClaimStatus: evidence.StatusAccepted, EffectSizeRows: []analysis.InputRow{{PaperID: "p1"}}, AcceptedEvidence: []evidence.EvidenceItem{{PaperID: "p1", Status: evidence.StatusAccepted}}, Passages: []report.TracePassage{{PassageID: "p1"}}}}}
	panelPath := filepath.Join(project, "data", "claim-panel.json")
	writeJSONForCLITest(t, panelPath, report.BuildClaimTraceabilityPanel(readyTrace))
	outsidePath := filepath.Join(t.TempDir(), "outside.md")
	outsideBefore := []byte("outside final report must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside final report: %v", err)
	}
	outPath := filepath.Join(project, "final.md")
	if err := os.Symlink(outsidePath, outPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "report", "final-export", "--in", input, "--panel", panelPath, "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("final export succeeded with symlinked output: stdout=%s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside final report: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("final export wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(outPath)
	if err != nil {
		t.Fatalf("lstat final report: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("final export replaced output symlink despite rejecting it: mode=%v", info.Mode())
	}
}
