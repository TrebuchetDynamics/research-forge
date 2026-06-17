package parsing

import (
	"context"
	"regexp"
	"strings"
)

// ParserAdapter parses a source document into a structured ParsedDocument.
type ParserAdapter interface {
	Parse(context.Context, []byte, ParseOptions) (ParsedDocument, error)
}

// ParseOptions configures one parse request.
type ParseOptions struct {
	PaperID string
}

// ParsedDocument is normalized structured paper content.
type ParsedDocument struct {
	SchemaVersion         string
	PaperID               string
	ParserName            string
	ParserVersion         string
	Title                 string
	TitleOffset           TextOffset
	Authors               []ParsedAuthor
	Abstract              string
	AbstractOffset        TextOffset
	Sections              []Section
	References            []Reference
	Warnings              []string
	ParserConfidence      float64
	LayeredAnnotations    []LayeredAnnotation
	CitationSpans         []CitationSpan
	ReconciliationOutputs []ParserReconciliationOutput
}

// ParsedAuthor is a TEI-normalized author.
type ParsedAuthor struct {
	Given  string
	Family string
}

// Section is one parsed full-text section.
type Section struct {
	ID               string
	Title            string
	Offset           TextOffset
	ParserConfidence float64
	Passages         []Passage
}

// Passage is one retrievable section passage.
type Passage struct {
	ID                 string
	PaperID            string
	SectionID          string
	Text               string
	Offset             TextOffset
	ParserConfidence   float64
	LayeredAnnotations []LayeredAnnotation
	CitationSpans      []CitationSpan
}

// Reference is one parsed bibliography reference.
type Reference struct {
	Title      string
	DOI        string
	Raw        string
	Confidence float64
	Offset     TextOffset
}

// LayeredAnnotation records parser layer output such as section, paragraph, reference, or table annotations.
type LayeredAnnotation struct {
	ID         string     `json:"id"`
	Layer      string     `json:"layer"`
	Label      string     `json:"label,omitempty"`
	Text       string     `json:"text,omitempty"`
	Offset     TextOffset `json:"offset"`
	ParserName string     `json:"parserName,omitempty"`
	Confidence float64    `json:"confidence,omitempty"`
}

// CitationSpan records a citation mention and its stable offset in parsed text.
type CitationSpan struct {
	ID             string     `json:"id"`
	PassageID      string     `json:"passageId,omitempty"`
	Text           string     `json:"text"`
	Offset         TextOffset `json:"offset"`
	ReferenceIndex int        `json:"referenceIndex"`
	Confidence     float64    `json:"confidence,omitempty"`
}

// ParserReconciliationOutput records field-level multi-parser reconciliation.
type ParserReconciliationOutput struct {
	Field          string                    `json:"field"`
	AcceptedParser string                    `json:"acceptedParser"`
	AcceptedValue  string                    `json:"acceptedValue"`
	Reason         string                    `json:"reason"`
	Alternatives   []ParserReconciliationAlt `json:"alternatives,omitempty"`
}

type ParserReconciliationAlt struct {
	ParserName string     `json:"parserName"`
	Value      string     `json:"value"`
	Offset     TextOffset `json:"offset"`
	Warnings   []string   `json:"warnings,omitempty"`
}

// EnrichParsedDocumentModel fills deterministic local offsets, layer annotations, citation spans, and parser confidence.
func EnrichParsedDocumentModel(doc ParsedDocument) ParsedDocument {
	cursor := 0
	doc.TitleOffset = nextOffset(&cursor, doc.Title)
	doc.AbstractOffset = nextOffset(&cursor, doc.Abstract)
	doc.ParserConfidence = parserConfidence(doc.Warnings, doc.Title, doc.Abstract)
	doc.LayeredAnnotations = nil
	doc.CitationSpans = nil
	for i := range doc.Sections {
		section := &doc.Sections[i]
		section.Offset = nextOffset(&cursor, section.Title)
		section.ParserConfidence = doc.ParserConfidence
		if strings.TrimSpace(section.Title) != "" {
			doc.LayeredAnnotations = append(doc.LayeredAnnotations, LayeredAnnotation{ID: section.ID + "-section", Layer: "section", Label: section.Title, Text: section.Title, Offset: section.Offset, ParserName: doc.ParserName, Confidence: section.ParserConfidence})
		}
		for j := range section.Passages {
			passage := &section.Passages[j]
			passage.Offset = nextOffset(&cursor, passage.Text)
			passage.ParserConfidence = doc.ParserConfidence
			annotation := LayeredAnnotation{ID: passage.ID + "-paragraph", Layer: "paragraph", Text: passage.Text, Offset: passage.Offset, ParserName: doc.ParserName, Confidence: passage.ParserConfidence}
			passage.LayeredAnnotations = appendUniqueAnnotation(passage.LayeredAnnotations, annotation)
			doc.LayeredAnnotations = append(doc.LayeredAnnotations, annotation)
			spans := citationSpansForPassage(*passage, len(doc.References))
			passage.CitationSpans = append(passage.CitationSpans, spans...)
			doc.CitationSpans = append(doc.CitationSpans, spans...)
		}
	}
	for i := range doc.References {
		text := firstNonEmptyParsing(doc.References[i].Title, doc.References[i].Raw, doc.References[i].DOI)
		doc.References[i].Offset = nextOffset(&cursor, text)
		if doc.References[i].Confidence == 0 {
			doc.References[i].Confidence = doc.ParserConfidence
		}
	}
	return doc
}

func nextOffset(cursor *int, text string) TextOffset {
	start := *cursor
	end := start + len(text)
	*cursor = end + 1
	return TextOffset{Start: start, End: end}
}

func parserConfidence(warnings []string, values ...string) float64 {
	score := 1.0 - 0.1*float64(len(warnings))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			score -= 0.1
		}
	}
	if score < 0.1 {
		return 0.1
	}
	if score > 1 {
		return 1
	}
	return score
}

func appendUniqueAnnotation(existing []LayeredAnnotation, annotation LayeredAnnotation) []LayeredAnnotation {
	for _, item := range existing {
		if item.ID == annotation.ID {
			return existing
		}
	}
	return append(existing, annotation)
}

var bracketCitationPattern = regexp.MustCompile(`\[(\d+)\]`)

func citationSpansForPassage(passage Passage, referenceCount int) []CitationSpan {
	matches := bracketCitationPattern.FindAllStringSubmatchIndex(passage.Text, -1)
	spans := []CitationSpan{}
	for i, match := range matches {
		refIndex := parseCitationIndex(passage.Text[match[2]:match[3]])
		if refIndex < 0 || (referenceCount > 0 && refIndex >= referenceCount) {
			continue
		}
		spans = append(spans, CitationSpan{ID: passage.ID + "-cite-" + intString(i+1), PassageID: passage.ID, Text: passage.Text[match[0]:match[1]], Offset: TextOffset{Start: passage.Offset.Start + match[0], End: passage.Offset.Start + match[1]}, ReferenceIndex: refIndex, Confidence: passage.ParserConfidence})
	}
	return spans
}

func parseCitationIndex(value string) int {
	idx := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return -1
		}
		idx = idx*10 + int(r-'0')
	}
	return idx - 1
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func firstNonEmptyParsing(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
