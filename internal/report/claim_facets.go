package report

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// BuildProjectCitationEvidenceTraceView uses the latest ledger facets only when
// they describe exactly the included text and citation evidence. A stale binding
// is an error requiring reassessment, not an excuse to drop a negative facet.
func BuildProjectCitationEvidenceTraceView(project string, input CitationEvidenceTraceInput) (CitationEvidenceTraceView, error) {
	records, err := claims.List(project)
	if err != nil {
		return CitationEvidenceTraceView{}, err
	}
	view := BuildCitationEvidenceTraceView(input)
	for i := range input.Claims {
		for _, record := range records {
			if record.ID != input.Claims[i].ID {
				continue
			}
			if !sameClaimEvidence(input.Claims[i], record, input.LibraryRecords) {
				return CitationEvidenceTraceView{}, fmt.Errorf("claim %s ledger facets refer to different text or evidence; reassessment required", record.ID)
			}
			view.Claims[i].Facets = record.Facets
		}
	}
	return view, nil
}

func sameClaimEvidence(suggestion evidence.CitationLockedSuggestion, record claims.Claim, sources []library.PaperRecord) bool {
	if suggestion.SuggestedText != record.Text || len(suggestion.CitationLocks) != len(record.Evidence) {
		return false
	}
	used := make([]bool, len(record.Evidence))
	for _, lock := range suggestion.CitationLocks {
		found := false
		for i, ref := range record.Evidence {
			if used[i] || ref.Relation != claims.RelationSupports || lock.ExactText != ref.ExactText || strings.TrimPrefix(lock.Ref, suggestion.PaperID+":") != ref.PassageID || !claims.ReferenceTargetsPaper(ref, suggestion.PaperID, sources) {
				continue
			}
			used[i], found = true, true
			break
		}
		if !found {
			return false
		}
	}
	return true
}
