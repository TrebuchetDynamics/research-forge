package report

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestBuildCitationEvidenceTraceViewLinksClaimsToFullProvenanceChain(t *testing.T) {
	claim := evidence.CitationLockedSuggestion{ID: "claim-1", PaperID: "paper-1", SuggestedText: "Treatment improved outcomes", Status: evidence.StatusAccepted, CitationLocks: []evidence.CitationLockedSupport{{Ref: "passage-1", ExactText: "Treatment improved outcomes."}}}
	item := evidence.EvidenceItem{PaperID: "paper-1", Values: map[string]string{"effect": "1"}, Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "passage-1"}, Status: evidence.StatusAccepted}
	run := analysis.AnalysisRun{ID: "run1", InputRows: []analysis.InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 0.1}}}
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "passage-1", PaperID: "paper-1", Text: "Treatment improved outcomes."}}}}})
	record := library.PaperRecord{Title: "Paper", Identifiers: library.Identifiers{DOI: "paper-1", ZoteroItemKey: "ZOT-1"}, SourceRefs: []library.SourceRef{{Source: "openalex", RawPayloadRef: "raw/openalex.json", Metadata: map[string]string{"request": "GET /works", "response": "200"}}}}
	view := BuildCitationEvidenceTraceView(CitationEvidenceTraceInput{Claims: []evidence.CitationLockedSuggestion{claim}, EvidenceItems: []evidence.EvidenceItem{item}, AnalysisRun: run, ParsedDocuments: []parsing.ParsedDocument{doc}, LibraryRecords: []library.PaperRecord{record}, PDFBaseURL: "/papers"})
	if view.SchemaVersion != "1" || len(view.Claims) != 1 {
		t.Fatalf("view = %#v", view)
	}
	row := view.Claims[0]
	if row.ClaimID != "claim-1" || len(row.EffectSizeRows) != 1 || len(row.AcceptedEvidence) != 1 || len(row.Passages) != 1 || row.Passages[0].ParserName != "grobid" || row.PDFViewURL != "/papers/paper-1/pdf#passage-1" {
		t.Fatalf("row chain = %#v", row)
	}
	if len(row.ReferenceManagerItems) != 1 || row.ReferenceManagerItems[0] != "zotero:ZOT-1" || len(row.SourceAPIRecords) != 1 || row.RawRequestResponse["request"] == "" {
		t.Fatalf("row source chain = %#v", row)
	}
}
