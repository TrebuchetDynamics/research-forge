package parsing

import (
	"context"
	"testing"
)

func TestCERMINEAdapterSeamParsesXMLIntoParsedDocument(t *testing.T) {
	input := []byte(`<article><title>CERMINE paper</title><abstract>Abstract text.</abstract><section title="Results"><p>Result passage [1].</p></section><ref><title>Reference title</title><doi>10.1000/ref</doi><raw>Reference raw</raw></ref></article>`)
	doc, err := (CERMINEXMLParser{Version: "fixture"}).Parse(context.Background(), input, ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.ParserName != "cermine" || doc.ParserVersion != "fixture" || doc.Title != "CERMINE paper" || len(doc.Sections) != 1 || len(doc.References) != 1 {
		t.Fatalf("doc = %#v", doc)
	}
	if len(doc.CitationSpans) != 1 || doc.CitationSpans[0].ReferenceIndex != 0 || doc.ParserConfidence == 0 {
		t.Fatalf("enrichment missing: %#v", doc)
	}
}

func TestDefaultParserFallbackPolicyIncludesCERMINEAndRequiresReviewForConflicts(t *testing.T) {
	policy := DefaultParserFallbackPolicy()
	if !policy.HasParser("cermine") || !policy.ReviewerRequired || policy.AutoAcceptConflicts {
		t.Fatalf("policy = %#v", policy)
	}
	decision := policy.Plan([]ParsedDocument{{ParserName: "grobid", Title: "A"}, {ParserName: "cermine", Title: "B"}})
	if !decision.ConflictReviewRequired || decision.NextAction != "open parser conflict review" {
		t.Fatalf("decision = %#v", decision)
	}
}
