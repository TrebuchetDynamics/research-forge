package parsing

import (
	"context"
	"testing"
)

type fakeAnyStyleRunner struct{ output []byte }

func (f fakeAnyStyleRunner) Run(context.Context, []byte) ([]byte, error) { return f.output, nil }

func TestAnyStyleReferenceParserAdaptsJSONReferences(t *testing.T) {
	parser := AnyStyleReferenceParser{Runner: fakeAnyStyleRunner{output: []byte(`[{"title":"Parsed reference","doi":"https://doi.org/10.1000/REF","raw":"Raw parsed reference","confidence":0.91},{"raw":"Fallback title"}]`)}, Version: "fixture"}

	doc, err := parser.ParseReferences(context.Background(), "paper-1", []byte("refs"))
	if err != nil {
		t.Fatalf("ParseReferences returned error: %v", err)
	}
	if doc.ParserName != "anystyle" || doc.ParserVersion != "fixture" || doc.PaperID != "paper-1" || len(doc.References) != 2 {
		t.Fatalf("doc = %#v", doc)
	}
	if doc.References[0].Title != "Parsed reference" || doc.References[0].DOI != "10.1000/ref" || doc.References[0].Raw != "Raw parsed reference" || doc.References[0].Confidence != 0.91 {
		t.Fatalf("first reference = %#v", doc.References[0])
	}
	if doc.References[1].Title != "Fallback title" {
		t.Fatalf("second reference = %#v", doc.References[1])
	}
}

func TestAnyStyleReferenceParserWarnsWhenNoReferencesParsed(t *testing.T) {
	parser := AnyStyleReferenceParser{Runner: fakeAnyStyleRunner{output: []byte(`[{"title":""}]`)}}
	doc, err := parser.ParseReferences(context.Background(), "paper-1", []byte("refs"))
	if err != nil {
		t.Fatalf("ParseReferences returned error: %v", err)
	}
	if len(doc.References) != 0 || len(doc.Warnings) != 1 {
		t.Fatalf("doc = %#v", doc)
	}
}
