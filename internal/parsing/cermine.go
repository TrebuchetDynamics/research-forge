package parsing

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

type CERMINEXMLParser struct{ Version string }

func (p CERMINEXMLParser) Parse(ctx context.Context, data []byte, options ParseOptions) (ParsedDocument, error) {
	_ = ctx
	var payload cermineArticle
	if err := xml.Unmarshal(data, &payload); err != nil {
		return ParsedDocument{}, err
	}
	version := p.Version
	if version == "" {
		version = "external-xml"
	}
	doc := ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(options.PaperID), ParserName: "cermine", ParserVersion: version, Title: compactParsingSpace(payload.Title), Abstract: compactParsingSpace(payload.Abstract)}
	for i, sec := range payload.Sections {
		title := compactParsingSpace(sec.Title)
		if title == "" {
			title = fmt.Sprintf("Section %d", i+1)
		}
		section := Section{ID: fmt.Sprintf("cermine-s%d", i+1), Title: title}
		for j, para := range sec.Paragraphs {
			text := compactParsingSpace(para)
			if text != "" {
				section.Passages = append(section.Passages, Passage{ID: fmt.Sprintf("%s-p%d", section.ID, j+1), PaperID: doc.PaperID, SectionID: section.ID, Text: text})
			}
		}
		doc.Sections = append(doc.Sections, section)
	}
	for _, ref := range payload.References {
		title := compactParsingSpace(ref.Title)
		doi := normalizeReferenceDOI(ref.DOI)
		raw := compactParsingSpace(ref.Raw)
		if title != "" || doi != "" || raw != "" {
			doc.References = append(doc.References, Reference{Title: title, DOI: doi, Raw: raw, Confidence: ref.Confidence})
		}
	}
	if doc.Title == "" {
		doc.Warnings = append(doc.Warnings, "missing title")
	}
	return EnrichParsedDocumentModel(doc), nil
}

type cermineArticle struct {
	Title      string             `xml:"title"`
	Abstract   string             `xml:"abstract"`
	Sections   []cermineSection   `xml:"section"`
	References []cermineReference `xml:"ref"`
}
type cermineSection struct {
	Title      string   `xml:"title,attr"`
	Paragraphs []string `xml:"p"`
}
type cermineReference struct {
	Title      string  `xml:"title"`
	DOI        string  `xml:"doi"`
	Raw        string  `xml:"raw"`
	Confidence float64 `xml:"confidence"`
}

type ParserFallbackPolicy struct {
	Parsers             []string `json:"parsers"`
	ReviewerRequired    bool     `json:"reviewerRequired"`
	AutoAcceptConflicts bool     `json:"autoAcceptConflicts"`
}
type ParserFallbackDecision struct {
	OrderedParsers         []string         `json:"orderedParsers"`
	ConflictReviewRequired bool             `json:"conflictReviewRequired"`
	NextAction             string           `json:"nextAction"`
	Comparison             ComparisonReport `json:"comparison"`
}

func DefaultParserFallbackPolicy() ParserFallbackPolicy {
	return ParserFallbackPolicy{Parsers: []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"}, ReviewerRequired: true, AutoAcceptConflicts: false}
}
func (p ParserFallbackPolicy) HasParser(name string) bool {
	name = canonicalParserName(name)
	for _, parser := range p.Parsers {
		if canonicalParserName(parser) == name {
			return true
		}
	}
	return false
}
func (p ParserFallbackPolicy) Plan(docs []ParsedDocument) ParserFallbackDecision {
	comparison := CompareParsedDocuments(docs)
	review := p.ReviewerRequired && (comparison.RecommendedUse == "review-required" || comparison.TitleMismatch || comparison.ReferenceDelta != 0 || comparison.PassageDelta != 0)
	action := "continue"
	if review {
		action = "open parser conflict review"
	}
	return ParserFallbackDecision{OrderedParsers: append([]string{}, p.Parsers...), ConflictReviewRequired: review, NextAction: action, Comparison: comparison}
}
