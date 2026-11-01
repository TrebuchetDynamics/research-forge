package adapters

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/model"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/quality"
	textutil "github.com/TrebuchetDynamics/research-forge/internal/parsing/text"
	"strings"
)

type CERMINEXMLParser struct{ Version string }

func (p CERMINEXMLParser) Parse(ctx context.Context, data []byte, options model.ParseOptions) (model.ParsedDocument, error) {
	_ = ctx
	var payload cermineArticle
	if err := xml.Unmarshal(data, &payload); err != nil {
		return model.ParsedDocument{}, err
	}
	version := p.Version
	if version == "" {
		version = "external-xml"
	}
	doc := model.ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(options.PaperID), ParserName: "cermine", ParserVersion: version, Title: textutil.CompactSpace(payload.Title), Abstract: textutil.CompactSpace(payload.Abstract)}
	for i, sec := range payload.Sections {
		title := textutil.CompactSpace(sec.Title)
		if title == "" {
			title = fmt.Sprintf("model.Section %d", i+1)
		}
		section := model.Section{ID: fmt.Sprintf("cermine-s%d", i+1), Title: title}
		for j, para := range sec.Paragraphs {
			text := textutil.CompactSpace(para)
			if text != "" {
				section.Passages = append(section.Passages, model.Passage{ID: fmt.Sprintf("%s-p%d", section.ID, j+1), PaperID: doc.PaperID, SectionID: section.ID, Text: text})
			}
		}
		doc.Sections = append(doc.Sections, section)
	}
	for _, ref := range payload.References {
		title := textutil.CompactSpace(ref.Title)
		doi := textutil.NormalizeReferenceDOI(ref.DOI)
		raw := textutil.CompactSpace(ref.Raw)
		if title != "" || doi != "" || raw != "" {
			doc.References = append(doc.References, model.Reference{Title: title, DOI: doi, Raw: raw, Confidence: ref.Confidence})
		}
	}
	if doc.Title == "" {
		doc.Warnings = append(doc.Warnings, "missing title")
	}
	return model.EnrichParsedDocumentModel(doc), nil
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
	OrderedParsers         []string                 `json:"orderedParsers"`
	ConflictReviewRequired bool                     `json:"conflictReviewRequired"`
	NextAction             string                   `json:"nextAction"`
	Comparison             quality.ComparisonReport `json:"comparison"`
}

func DefaultParserFallbackPolicy() ParserFallbackPolicy {
	return ParserFallbackPolicy{Parsers: []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"}, ReviewerRequired: true, AutoAcceptConflicts: false}
}
func (p ParserFallbackPolicy) HasParser(name string) bool {
	name = quality.CanonicalParserName(name)
	for _, parser := range p.Parsers {
		if quality.CanonicalParserName(parser) == name {
			return true
		}
	}
	return false
}
func (p ParserFallbackPolicy) Plan(docs []model.ParsedDocument) ParserFallbackDecision {
	comparison := quality.CompareParsedDocuments(docs)
	review := p.ReviewerRequired && (comparison.RecommendedUse == "review-required" || comparison.TitleMismatch || comparison.ReferenceDelta != 0 || comparison.PassageDelta != 0)
	action := "continue"
	if review {
		action = "open parser conflict review"
	}
	return ParserFallbackDecision{OrderedParsers: append([]string{}, p.Parsers...), ConflictReviewRequired: review, NextAction: action, Comparison: comparison}
}
