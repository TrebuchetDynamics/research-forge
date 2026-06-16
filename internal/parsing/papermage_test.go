package parsing

import (
	"context"
	"testing"
)

func TestPaperMageJSONParserMapsLayersToParsedDocument(t *testing.T) {
	input := []byte(`{"metadata":{"title":"Layered paper"},"layers":{"sections":[{"text":"Introduction"}],"paragraphs":[{"section":"Introduction","text":"First paragraph."}],"bibliography":[{"title":"Reference title","doi":"10.1000/ref"}]},"warnings":["low confidence bibliography"]}`)
	doc, err := (PaperMageJSONParser{}).Parse(context.Background(), input, ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if doc.ParserName != "papermage" || doc.Title != "Layered paper" || len(doc.Sections) != 1 || len(doc.Sections[0].Passages) != 1 || len(doc.References) != 1 {
		t.Fatalf("doc = %#v", doc)
	}
	if doc.References[0].DOI != "10.1000/ref" || doc.Warnings[0] != "low confidence bibliography" {
		t.Fatalf("references/warnings = %#v %#v", doc.References, doc.Warnings)
	}
}
