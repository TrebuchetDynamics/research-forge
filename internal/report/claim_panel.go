package report

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
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
		row := ClaimTracePanelRow{ClaimID: trace.ClaimID, PaperID: trace.PaperID, Kind: generatedOutputKind(trace.ClaimText), Status: ClaimTraceReady, Reason: "trace links present; unchecked facets remain unchecked; no truth judgment", Trace: trace}
		if reason := facetBlocker(trace); reason != "" {
			row.Status = ClaimTraceWeakSupport
			row.Reason = reason
		} else if trace.ClaimID == "" || trace.ClaimText == "" || trace.ClaimStatus != evidence.StatusAccepted {
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
	for _, row := range panel.Rows {
		if reason := facetBlocker(row.Trace); reason != "" {
			return fmt.Errorf("final export blocked by claim %s: %s", row.ClaimID, reason)
		}
		if row.Status != ClaimTraceReady {
			return fmt.Errorf("final export blocked by claim %s", row.ClaimID)
		}
	}
	if panel.BlockFinalExport {
		return fmt.Errorf("final export blocked by %d unresolved or weak claim trace(s)", len(panel.Blockers))
	}
	return nil
}

// Export readiness is a workflow guard, not a combined verification facet.
// Legacy nil facets stay unchecked; review status never upgrades them to passed.
func facetBlocker(trace ClaimTraceView) string {
	facets := trace.Facets
	type namedFacet struct {
		name  string
		value *claims.Assessment
	}
	checks := []namedFacet{
		{"sourceIdentity", facets.SourceIdentity},
		{"contentIntegrity", facets.ContentIntegrity},
		{"passageMatch", facets.PassageMatch},
	}
	if facets.SemanticSupport != nil {
		value := facets.SemanticSupport.Assessment()
		checks = append(checks, namedFacet{"semanticSupport (agent_reported)", &value})
	}
	if facets.HumanApproval != nil {
		value := facets.HumanApproval.Assessment()
		checks = append(checks, namedFacet{"humanApproval", &value})
	}
	for _, check := range checks {
		if check.value != nil && check.value.State != claims.StatePassed {
			return fmt.Sprintf("%s facet is %s", check.name, check.value.State)
		}
	}
	for _, match := range trace.CitationMatches {
		if match.Match.State != claims.StatePassed {
			return fmt.Sprintf("passageMatch for %s is %s", match.Ref, match.Match.State)
		}
	}
	return ""
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
