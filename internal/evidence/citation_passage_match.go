package evidence

import (
	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type CitationLockMatch struct {
	Ref   string              `json:"ref"`
	Match claims.PassageMatch `json:"match"`
}

// CheckCitationLocks compares every quoted support with its stored passage.
// A review status or an ID link is not proof that the quoted text appears there.
// Missing or ambiguous passages are inconclusive, never silently successful.
func (s CitationLockedSuggestion) CheckCitationLocks(docs []parsing.ParsedDocument) []CitationLockMatch {
	out := make([]CitationLockMatch, 0, len(s.CitationLocks))
	for _, lock := range s.CitationLocks {
		result := CitationLockMatch{Ref: lock.Ref, Match: claims.PassageMatch{State: claims.StateInconclusive}}
		count := 0
		for _, doc := range docs {
			if s.PaperID == "" || doc.PaperID != s.PaperID {
				continue
			}
			for _, section := range doc.Sections {
				for _, passage := range section.Passages {
					if passage.ID == "" || (lock.Ref != passage.ID && lock.Ref != s.PaperID+":"+passage.ID) || (passage.PaperID != "" && passage.PaperID != s.PaperID) {
						continue
					}
					count++
					result.Match = claims.MatchPassage(passage.Text, lock.ExactText)
					if result.Match.State == claims.StatePassed && passage.Offset.Start >= 0 && passage.Offset.End-passage.Offset.Start == len(passage.Text) && doc.ParserName != "" && doc.ParserVersion != "" {
						result.Match.Offsets = &claims.TextOffsets{Start: passage.Offset.Start + result.Match.Start, End: passage.Offset.Start + result.Match.End, ParserName: doc.ParserName, ParserVersion: doc.ParserVersion}
					}
				}
			}
		}
		if count != 1 {
			result.Match = claims.PassageMatch{State: claims.StateInconclusive}
		}
		out = append(out, result)
	}
	return out
}
