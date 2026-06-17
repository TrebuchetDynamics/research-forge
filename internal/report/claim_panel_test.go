package report

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestClaimTraceabilityPanelBlocksUnresolvedWeakGeneratedOutputs(t *testing.T) {
	panel := BuildClaimTraceabilityPanel(CitationEvidenceTraceView{SchemaVersion: "1", Claims: []ClaimTraceView{
		{ClaimID: "ok", PaperID: "p1", ClaimText: "supported paragraph", ClaimStatus: evidence.StatusAccepted, EffectSizeRows: []analysis.InputRow{{PaperID: "p1"}}, AcceptedEvidence: []evidence.EvidenceItem{{PaperID: "p1", Status: evidence.StatusAccepted}}, Passages: []TracePassage{{PassageID: "p1-passage"}}},
		{ClaimID: "unresolved", PaperID: "p2", ClaimText: "generated table without review"},
		{ClaimID: "weak", PaperID: "p3", ClaimText: "generated figure with weak support", ClaimStatus: evidence.StatusAccepted, AcceptedEvidence: []evidence.EvidenceItem{{PaperID: "p3", Status: evidence.StatusAccepted}}},
	}})
	if panel.SchemaVersion != "1" || len(panel.Rows) != 3 || panel.BlockFinalExport != true {
		t.Fatalf("panel = %#v", panel)
	}
	if panel.Rows[0].Status != ClaimTraceReady || panel.Rows[1].Status != ClaimTraceUnresolved || panel.Rows[2].Status != ClaimTraceWeakSupport {
		t.Fatalf("rows = %#v", panel.Rows)
	}
	if err := GuardFinalReportExport(panel); err == nil {
		t.Fatalf("expected final export blocker")
	}
	ready := BuildClaimTraceabilityPanel(CitationEvidenceTraceView{Claims: []ClaimTraceView{panel.Rows[0].Trace}})
	if err := GuardFinalReportExport(ready); err != nil {
		t.Fatalf("ready panel blocked: %v", err)
	}
}
