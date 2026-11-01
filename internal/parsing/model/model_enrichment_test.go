package model

import "testing"

func TestEnrichParsedDocumentAddsStableOffsetsAnnotationsCitationSpansAndConfidence(t *testing.T) {
	doc := ParsedDocument{
		SchemaVersion: "1", PaperID: "paper-1", ParserName: "papermage", Title: "Title", Abstract: "Abstract",
		Sections:   []Section{{ID: "s1", Title: "Results", Passages: []Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Treatment improved outcomes [1]."}}}},
		References: []Reference{{Title: "Reference one", Confidence: 0.8}},
	}
	enriched := EnrichParsedDocumentModel(doc)
	if enriched.TitleOffset.End <= enriched.TitleOffset.Start || enriched.AbstractOffset.End <= enriched.AbstractOffset.Start {
		t.Fatalf("document offsets missing: %#v %#v", enriched.TitleOffset, enriched.AbstractOffset)
	}
	passage := enriched.Sections[0].Passages[0]
	if passage.Offset.End <= passage.Offset.Start || passage.ParserConfidence == 0 {
		t.Fatalf("passage offset/confidence missing: %#v", passage)
	}
	if len(enriched.LayeredAnnotations) == 0 || enriched.LayeredAnnotations[0].Layer != "section" {
		t.Fatalf("layered annotations missing: %#v", enriched.LayeredAnnotations)
	}
	if len(enriched.CitationSpans) != 1 || enriched.CitationSpans[0].Text != "[1]" || enriched.CitationSpans[0].ReferenceIndex != 0 {
		t.Fatalf("citation spans missing: %#v", enriched.CitationSpans)
	}
	if enriched.ParserConfidence <= 0 || enriched.ParserConfidence > 1 {
		t.Fatalf("parser confidence = %v", enriched.ParserConfidence)
	}
}
