package benchmarks

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
)

// CitationRatio is null-valued for 0/0, never a perfect score for no citations.
type CitationRatio struct {
	Numerator   int      `json:"numerator"`
	Denominator int      `json:"denominator"`
	Value       *float64 `json:"value"`
}

// CitationFacetCounts keeps unavailable data visible. Missing is the subset of
// occurrences lacking required reference fields, not a fourth assessment state;
// unavailable stored evidence also contributes to Inconclusive.
type CitationFacetCounts struct {
	Passed       int `json:"passed"`
	Failed       int `json:"failed"`
	Inconclusive int `json:"inconclusive"`
	Missing      int `json:"missing"`
}

type CitationAudit struct {
	ClaimIndex    int                `json:"claimIndex"`
	EvidenceIndex int                `json:"evidenceIndex"`
	Check         claims.CheckResult `json:"check"`
}

type CitationScore struct {
	Status           string              `json:"citation_status"`
	Correctness      CitationRatio       `json:"citation_correctness"`
	SourceIdentity   CitationFacetCounts `json:"sourceIdentity"`
	ContentIntegrity CitationFacetCounts `json:"contentIntegrity"`
	PassageMatch     CitationFacetCounts `json:"passageMatch"`
	// Audit belongs in the private scoring artifact, not an agent-visible gold set.
	Audit []CitationAudit `json:"audit"`
}

// ScoreCitations recomputes every claim-to-source citation occurrence against
// the caller's locked evidence corpus. It never trusts supplied facet states,
// deduplicates citations, or infers semantic support/human approval. The metric's
// conjunction is not an overall claim facet or a scientific truth judgment.
func ScoreCitations(records []claims.Claim, input claims.CheckInput) (CitationScore, error) {
	if input.Timestamp.IsZero() {
		return CitationScore{}, fmt.Errorf("citation scoring requires an explicit timestamp")
	}
	out := CitationScore{Status: "no_citations", Audit: []CitationAudit{}}
	for claimIndex, record := range records {
		for evidenceIndex, ref := range record.Evidence {
			occurrence := record
			occurrence.Evidence = []claims.EvidenceReference{ref}
			checked, err := claims.Check(occurrence, input)
			if err != nil {
				return CitationScore{}, err
			}
			out.Audit = append(out.Audit, CitationAudit{ClaimIndex: claimIndex, EvidenceIndex: evidenceIndex, Check: checked})
			f := checked.Claim.Facets
			out.SourceIdentity.add(f.SourceIdentity.State, strings.TrimSpace(ref.Identifier.Scheme) == "" || strings.TrimSpace(ref.Identifier.Value) == "")
			out.ContentIntegrity.add(f.ContentIntegrity.State, ref.ContentHash == "")
			out.PassageMatch.add(f.PassageMatch.State, strings.TrimSpace(ref.PassageID) == "" || strings.TrimSpace(ref.ExactText) == "")
			out.Correctness.Denominator++
			if f.SourceIdentity.State == claims.StatePassed && f.ContentIntegrity.State == claims.StatePassed && f.PassageMatch.State == claims.StatePassed {
				out.Correctness.Numerator++
			}
		}
	}
	if out.Correctness.Denominator > 0 {
		out.Status = "measured"
		value := float64(out.Correctness.Numerator) / float64(out.Correctness.Denominator)
		out.Correctness.Value = &value
	}
	return out, nil
}

func (counts *CitationFacetCounts) add(state claims.State, missing bool) {
	if missing {
		counts.Missing++
	}
	switch state {
	case claims.StatePassed:
		counts.Passed++
	case claims.StateFailed:
		counts.Failed++
	default:
		counts.Inconclusive++
	}
}
