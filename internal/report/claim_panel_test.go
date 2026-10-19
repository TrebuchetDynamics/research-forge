package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

// Ignoring a recorded negative facet must never permit final export.
func TestFinalExportReadsSeparateFacets(t *testing.T) {
	for _, facet := range []string{"sourceIdentity", "contentIntegrity", "passageMatch", "semanticSupport", "humanApproval"} {
		for _, state := range []string{"passed", "failed", "inconclusive"} {
			t.Run(facet+"/"+state, func(t *testing.T) {
				kind := "engine"
				if facet == "semanticSupport" {
					kind = "agent"
				} else if facet == "humanApproval" {
					kind = "human"
				}
				var trace ClaimTraceView
				data := fmt.Sprintf(`{"claimId":"c1","claimText":"Claim","claimStatus":"accepted","acceptedEvidence":[{}],"passages":[{}],"effectSizeRows":[{}],"facets":{%q:{"state":%q,"checker":{"kind":%q,"id":"checker","model":"fixture"},"timestamp":"2026-10-19T12:00:00Z"}}}`, facet, state, kind)
				if err := json.Unmarshal([]byte(data), &trace); err != nil {
					t.Fatal(err)
				}
				panel := BuildClaimTraceabilityPanel(CitationEvidenceTraceView{Claims: []ClaimTraceView{trace}})
				err := GuardFinalReportExport(panel)
				blocked := state != "passed"
				if (err != nil) != blocked || (blocked && !strings.Contains(strings.Join(panel.Blockers, " "), facet)) {
					t.Fatalf("export ignored %s: panel=%+v err=%v", facet, panel, err)
				}
				// The guard also reads facets when handed a prebuilt ready panel.
				panel.BlockFinalExport = false
				panel.Rows[0].Status = ClaimTraceReady
				if err := GuardFinalReportExport(panel); (err != nil) != blocked {
					t.Fatalf("ready status overrode %s: %v", facet, err)
				}
			})
		}
	}
}

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
