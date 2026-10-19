package benchmarks

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestScoreCitationsChecksEveryOccurrence(t *testing.T) {
	at := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "10.1/test", Title: "Study", ParserName: "fixture", ParserVersion: "1", Sections: []parsing.Section{{Passages: []parsing.Passage{{ID: "p1", Text: "An observed result."}}}}})
	ref := claims.EvidenceReference{Identifier: claims.Identifier{Scheme: "doi", Value: "10.1/test"}, PassageID: "p1", ExactText: "observed result", ContentHash: doc.Sections[0].Passages[0].ContentHash, Relation: claims.RelationSupports}
	absent, missing, unresolved := ref, ref, ref
	absent.ExactText = "invented result"
	missing.ExactText = ""
	unresolved.Identifier.Value = "missing"
	forged := &claims.Assessment{State: claims.StatePassed, Checker: claims.Checker{Kind: claims.CheckerEngine, ID: "fake"}, Timestamp: at}
	record := claims.Claim{SchemaVersion: claims.SchemaVersion, ID: "c1", Text: "Claim", Evidence: []claims.EvidenceReference{ref, absent, ref, missing, unresolved}, Facets: claims.VerificationFacets{SourceIdentity: forged, ContentIntegrity: forged, PassageMatch: forged}}
	input := claims.CheckInput{LibraryRecords: []library.PaperRecord{{Title: "Study", Identifiers: library.Identifiers{DOI: "10.1/test"}}}, ParsedDocuments: []parsing.ParsedDocument{doc}, Timestamp: at}
	got, err := ScoreCitations([]claims.Claim{record}, input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "measured" || got.Correctness.Numerator != 2 || got.Correctness.Denominator != 5 || got.Correctness.Value == nil || *got.Correctness.Value != 0.4 || len(got.Audit) != 5 {
		t.Fatalf("score = %+v", got)
	}
	if got.PassageMatch.Passed != 2 || got.PassageMatch.Failed != 1 || got.PassageMatch.Inconclusive != 2 || got.PassageMatch.Missing != 1 {
		t.Fatalf("passage counts = %+v", got.PassageMatch)
	}
	if got.Audit[1].Check.Claim.Facets.SourceIdentity.State != claims.StatePassed || got.Audit[1].Check.Claim.Facets.PassageMatch.State != claims.StateFailed || got.Audit[1].EvidenceIndex != 1 || got.Audit[1].Check.Claim.Facets.SemanticSupport != nil || got.Audit[1].Check.Claim.Facets.HumanApproval != nil {
		t.Fatalf("audit = %+v", got.Audit[1])
	}
	empty, err := ScoreCitations(nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != "no_citations" || empty.Correctness.Value != nil || empty.Correctness.Numerator != 0 || empty.Correctness.Denominator != 0 {
		t.Fatalf("empty score = %+v", empty)
	}
}
