package report

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestBuildPassageProvenanceFromParsedDocuments(t *testing.T) {
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", Text: "Quoted passage."}}}}})
	links := BuildPassageProvenanceFromParsedDocuments([]parsing.ParsedDocument{doc}, "parsed/paper-1.json")
	if len(links) != 1 || links[0].PaperID != "paper-1" || links[0].PassageID != "p1" || links[0].ParserName != "grobid" || links[0].ParserVersion != "0.8" || links[0].SourceOffsetEnd <= links[0].SourceOffsetStart || links[0].SourceRef != "parsed/paper-1.json#p1" {
		t.Fatalf("links = %#v", links)
	}
}
