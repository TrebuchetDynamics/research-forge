package evidence

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestCitationLocksCheckStoredText(t *testing.T) {
	docs := []parsing.ParsedDocument{parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper", ParserName: "fixture", ParserVersion: "1", Sections: []parsing.Section{{Passages: []parsing.Passage{{ID: "p1", Text: "A Café result."}}}}})}
	for _, tc := range []struct {
		paper, ref, quote string
		state             claims.State
	}{
		{"paper", "p1", "Café result", claims.StatePassed},
		{"paper", "paper:p1", "Cafe\u0301  result", claims.StatePassed},
		{"paper", "p1", "made up", claims.StateFailed},
		{"other", "p1", "Café result", claims.StateInconclusive},
	} {
		s := CitationLockedSuggestion{PaperID: tc.paper, CitationLocks: []CitationLockedSupport{{Ref: tc.ref, ExactText: tc.quote}}}
		got := s.CheckCitationLocks(docs)
		if len(got) != 1 || got[0].Ref != tc.ref || got[0].Match.State != tc.state {
			t.Fatalf("lock %+v = %+v", tc, got)
		}
		if tc.state == claims.StatePassed && (got[0].Match.Offsets == nil || got[0].Match.Offsets.Start != 5 || got[0].Match.Offsets.End != 17) {
			t.Fatalf("offsets = %+v", got[0].Match)
		}
	}
}
