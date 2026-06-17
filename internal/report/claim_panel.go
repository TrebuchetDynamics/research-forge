package report

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

type ClaimTraceStatus string

const (
	ClaimTraceReady       ClaimTraceStatus = "ready"
	ClaimTraceUnresolved  ClaimTraceStatus = "unresolved"
	ClaimTraceWeakSupport ClaimTraceStatus = "weak_support"
)

type ClaimTraceabilityPanel struct {
	SchemaVersion    string               `json:"schemaVersion"`
	Rows             []ClaimTracePanelRow `json:"rows"`
	BlockFinalExport bool                 `json:"blockFinalExport"`
	Blockers         []string             `json:"blockers,omitempty"`
}

type ClaimTracePanelRow struct {
	ClaimID string           `json:"claimId"`
	PaperID string           `json:"paperId"`
	Kind    string           `json:"kind"`
	Status  ClaimTraceStatus `json:"status"`
	Reason  string           `json:"reason"`
	Trace   ClaimTraceView   `json:"trace"`
}

func BuildClaimTraceabilityPanel(view CitationEvidenceTraceView) ClaimTraceabilityPanel {
	panel := ClaimTraceabilityPanel{SchemaVersion: "1"}
	for _, trace := range view.Claims {
		row := ClaimTracePanelRow{ClaimID: trace.ClaimID, PaperID: trace.PaperID, Kind: generatedOutputKind(trace.ClaimText), Status: ClaimTraceReady, Reason: "claim has accepted evidence, source passage, and analysis trace", Trace: trace}
		if trace.ClaimID == "" || trace.ClaimText == "" || trace.ClaimStatus != evidence.StatusAccepted {
			row.Status = ClaimTraceUnresolved
			row.Reason = "generated output is unresolved, unreviewed, or missing claim text"
		} else if len(trace.AcceptedEvidence) == 0 || len(trace.Passages) == 0 {
			row.Status = ClaimTraceWeakSupport
			row.Reason = "generated output lacks accepted evidence or source passage support"
		} else if len(trace.EffectSizeRows) == 0 {
			row.Status = ClaimTraceWeakSupport
			row.Reason = "generated output is not linked to downstream effect-size rows"
		}
		if row.Status != ClaimTraceReady {
			panel.BlockFinalExport = true
			panel.Blockers = append(panel.Blockers, row.ClaimID+": "+row.Reason)
		}
		panel.Rows = append(panel.Rows, row)
	}
	return panel
}

func GuardFinalReportExport(panel ClaimTraceabilityPanel) error {
	if panel.BlockFinalExport {
		return fmt.Errorf("final export blocked by %d unresolved or weak claim trace(s)", len(panel.Blockers))
	}
	for _, row := range panel.Rows {
		if row.Status != ClaimTraceReady {
			return fmt.Errorf("final export blocked by claim %s", row.ClaimID)
		}
	}
	return nil
}

func generatedOutputKind(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "table") {
		return "table"
	}
	if strings.Contains(lower, "figure") {
		return "figure"
	}
	return "paragraph"
}
