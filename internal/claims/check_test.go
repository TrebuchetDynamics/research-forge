package claims

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// Identity, hashes and quote text must be checked independently, never inferred
// from one another or from the submitted facet assessments.
func TestCheckDeterministicFacets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Claim, *CheckInput)
		want   [3]State
	}{
		{"exact", nil, [3]State{StatePassed, StatePassed, StatePassed}},
		{"normalized", func(c *Claim, _ *CheckInput) { c.Evidence[0].ExactText = "Cafe\u0301  result" }, [3]State{StatePassed, StatePassed, StatePassed}},
		{"absent quote", func(c *Claim, _ *CheckInput) { c.Evidence[0].ExactText = "invented result" }, [3]State{StatePassed, StatePassed, StateFailed}},
		{"wrong expected hash", func(c *Claim, _ *CheckInput) { c.Evidence[0].ContentHash = "sha256:" + strings.Repeat("0", 64) }, [3]State{StatePassed, StateFailed, StatePassed}},
		{"missing expected hash", func(c *Claim, _ *CheckInput) { c.Evidence[0].ContentHash = "" }, [3]State{StatePassed, StateInconclusive, StatePassed}},
		{"missing stored hash", func(_ *Claim, i *CheckInput) { i.ParsedDocuments[0].Sections[0].Passages[0].ContentHash = "" }, [3]State{StatePassed, StateInconclusive, StatePassed}},
		{"changed bytes", func(_ *Claim, i *CheckInput) { i.ParsedDocuments[0].Sections[0].Passages[0].Text = "A changed result." }, [3]State{StatePassed, StateFailed, StateFailed}},
		{"metadata mismatch", func(c *Claim, _ *CheckInput) { c.Evidence[0].SourceMetadata.Title = "Other study" }, [3]State{StateFailed, StatePassed, StatePassed}},
		{"metadata unavailable", func(_ *Claim, i *CheckInput) { i.LibraryRecords[0].Title = "" }, [3]State{StateInconclusive, StatePassed, StatePassed}},
		{"no comparable metadata", func(c *Claim, i *CheckInput) { c.Evidence[0].SourceMetadata = nil; i.ParsedDocuments[0].Title = "" }, [3]State{StateInconclusive, StatePassed, StatePassed}},
		{"unresolved source", func(c *Claim, _ *CheckInput) { c.Evidence[0].Identifier.Value = "missing" }, [3]State{StateFailed, StateInconclusive, StateInconclusive}},
		{"empty normalized ID", func(c *Claim, i *CheckInput) {
			c.Evidence[0].Identifier.Value = "doi:"
			i.LibraryRecords[0].Identifiers.DOI = ""
		}, [3]State{StateFailed, StateInconclusive, StateInconclusive}},
		{"passage offsets", func(c *Claim, i *CheckInput) {
			p := i.ParsedDocuments[0].Sections[0].Passages[0]
			c.Evidence[0].Offsets = &TextOffsets{Start: p.Offset.Start, End: p.Offset.End, ParserName: "fixture", ParserVersion: "1"}
		}, [3]State{StatePassed, StatePassed, StatePassed}},
		{"stale parser offsets", func(c *Claim, _ *CheckInput) {
			c.Evidence[0].Offsets = &TextOffsets{Start: 0, End: 1, ParserName: "fixture", ParserVersion: "old"}
		}, [3]State{StatePassed, StatePassed, StateFailed}},
		{"missing passage", func(c *Claim, _ *CheckInput) { c.Evidence[0].PassageID = "missing" }, [3]State{StatePassed, StateInconclusive, StateInconclusive}},
		{"ambiguous passage", func(_ *Claim, i *CheckInput) { i.ParsedDocuments = append(i.ParsedDocuments, i.ParsedDocuments[0]) }, [3]State{StatePassed, StateInconclusive, StateInconclusive}},
		{"all references required", func(c *Claim, _ *CheckInput) {
			extra := c.Evidence[0]
			extra.ExactText = ""
			c.Evidence = append(c.Evidence, extra)
		}, [3]State{StatePassed, StatePassed, StateInconclusive}},
		{"empty evidence", func(c *Claim, _ *CheckInput) { c.Evidence = nil }, [3]State{StateInconclusive, StateInconclusive, StateInconclusive}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
			doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "10.1/test", Title: "Study", ParserName: "fixture", ParserVersion: "1", Sections: []parsing.Section{{Passages: []parsing.Passage{{ID: "p1", Text: "A Café result."}}}}})
			semantic, err := NewSemanticSupport(Assessment{State: StateFailed, Checker: Checker{Kind: CheckerAgent, ID: "pi", Model: "model-1"}, Timestamp: at})
			if err != nil {
				t.Fatal(err)
			}
			human, err := NewHumanApproval(Assessment{State: StateFailed, Checker: Checker{Kind: CheckerHuman, ID: "ada"}, Timestamp: at})
			if err != nil {
				t.Fatal(err)
			}
			record := Claim{SchemaVersion: SchemaVersion, ID: "c1", Text: "An assertion", Evidence: []EvidenceReference{{Identifier: Identifier{Scheme: "doi", Value: "10.1/test"}, PassageID: "p1", ExactText: "Café result", ContentHash: doc.Sections[0].Passages[0].ContentHash, Relation: RelationSupports, SourceMetadata: &SourceMetadata{Title: "Study"}}}, Facets: VerificationFacets{SemanticSupport: &semantic, HumanApproval: &human}}
			input := CheckInput{LibraryRecords: []library.PaperRecord{{Title: "Study", Identifiers: library.Identifiers{DOI: "10.1/test"}}}, ParsedDocuments: []parsing.ParsedDocument{doc}, Timestamp: at}
			if tc.change != nil {
				tc.change(&record, &input)
			}
			result, err := Check(record, input)
			if err != nil {
				t.Fatal(err)
			}
			f := result.Claim.Facets
			if got := [3]State{f.SourceIdentity.State, f.ContentIntegrity.State, f.PassageMatch.State}; got != tc.want {
				t.Fatalf("facets = %v, want %v", got, tc.want)
			}
			if f.SemanticSupport != record.Facets.SemanticSupport || f.HumanApproval != record.Facets.HumanApproval || f.SourceIdentity.Checker.Kind != CheckerEngine || !f.SourceIdentity.Timestamp.Equal(at) {
				t.Fatalf("unexpected attribution: %+v", f)
			}
			if tc.want[2] == StatePassed {
				got := result.Evidence[0].PassageMatch
				if got.Offsets == nil || got.Offsets.Start != 10 || got.Offsets.End != 22 || got.Offsets.ParserName != "fixture" {
					t.Fatalf("match offsets = %+v", got)
				}
			}
		})
	}
}

func TestCheckRequiresExplicitTimestamp(t *testing.T) {
	if _, err := Check(Claim{}, CheckInput{}); err == nil {
		t.Fatal("check invented or omitted its timestamp")
	}
}

func TestMatchPassageOriginalByteOffsets(t *testing.T) {
	for _, tc := range []struct {
		name, text, quote, method string
		start, end                int
		state                     State
	}{
		{"exact first", "é hello hello", "hello", "exact", 3, 8, StatePassed},
		{"canonical and whitespace", "x Cafe\u0301\u00a0\n result!", "Café result", "normalized", 2, 18, StatePassed},
		{"hangul", "x \u1100\u1161!", "가", "normalized", 2, 8, StatePassed},
		{"canonical ordering", "a\u0315\u0300!", "à\u0315", "normalized", 0, 5, StatePassed},
		{"canonical singleton", "Å!", "Å", "normalized", 0, 2, StatePassed},
		{"accent is significant", "Café", "Cafe", "", 0, 0, StateFailed},
		{"partial Hangul expansion", "가", "ᄀ", "", 0, 0, StateFailed},
		{"partial Bengali expansion", "ো", "ে", "", 0, 0, StateFailed},
		{"missing", "observed", "invented", "", 0, 0, StateFailed},
		{"empty quote", "observed", " \n", "", 0, 0, StateInconclusive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchPassage(tc.text, tc.quote)
			if got.State != tc.state || got.Method != tc.method || (tc.state == StatePassed && (got.Start != tc.start || got.End != tc.end)) {
				t.Fatalf("match = %+v, want %+v", got, tc)
			}
		})
	}
}
