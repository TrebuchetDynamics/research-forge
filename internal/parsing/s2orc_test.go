package parsing

import (
	"context"
	"testing"
)

func TestS2ORCJSONParserAdaptsDocument(t *testing.T) {
	fixture := []byte(`{
		"title":"S2ORC Fixture",
		"abstract":"Structured abstract.",
		"body_text":[{"section":"Introduction","text":"Intro passage."},{"section":"Methods","text":"Methods passage."}],
		"bib_entries":{"BIBREF0":{"title":"Referenced Work","doi":"https://doi.org/10.1000/REF","raw_text":"Referenced Work raw"}}
	}`)

	doc, err := (S2ORCJSONParser{Version: "fixture"}).Parse(context.Background(), fixture, ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if doc.ParserName != "s2orc-doc2json" || doc.ParserVersion != "fixture" || doc.Title != "S2ORC Fixture" || doc.Abstract != "Structured abstract." {
		t.Fatalf("doc = %#v", doc)
	}
	if len(doc.Sections) != 2 || doc.Sections[0].Title != "Introduction" || doc.Sections[0].Passages[0].Text != "Intro passage." {
		t.Fatalf("sections = %#v", doc.Sections)
	}
	if len(doc.References) != 1 || doc.References[0].DOI != "10.1000/ref" || doc.References[0].Raw != "Referenced Work raw" {
		t.Fatalf("references = %#v", doc.References)
	}
}

func TestS2ORCJSONParserWarnsOnMissingTitle(t *testing.T) {
	doc, err := (S2ORCJSONParser{}).Parse(context.Background(), []byte(`{"body_text":[]}`), ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(doc.Warnings) != 1 || doc.Warnings[0] != "missing title" {
		t.Fatalf("warnings = %#v", doc.Warnings)
	}
}
