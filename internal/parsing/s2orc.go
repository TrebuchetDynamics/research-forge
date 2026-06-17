package parsing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// S2ORCJSONParser adapts s2orc-doc2json-like JSON into ParsedDocument.
type S2ORCJSONParser struct{ Version string }

func (p S2ORCJSONParser) Parse(ctx context.Context, data []byte, options ParseOptions) (ParsedDocument, error) {
	_ = ctx
	var payload s2orcDocument
	if err := json.Unmarshal(data, &payload); err != nil {
		return ParsedDocument{}, err
	}
	version := p.Version
	if version == "" {
		version = "s2orc-json"
	}
	doc := ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(options.PaperID), ParserName: "s2orc-doc2json", ParserVersion: version, Title: compactText(payload.Title), Abstract: compactText(payload.Abstract)}
	for i, paragraph := range payload.BodyText {
		sectionTitle := compactText(paragraph.Section)
		if sectionTitle == "" {
			sectionTitle = "Body"
		}
		sectionID := fmt.Sprintf("%s-s2orc-sec-%d", doc.PaperID, i+1)
		text := compactText(paragraph.Text)
		section := Section{ID: sectionID, Title: sectionTitle}
		if text != "" {
			section.Passages = append(section.Passages, Passage{ID: fmt.Sprintf("%s-p-1", sectionID), PaperID: doc.PaperID, SectionID: sectionID, Text: text})
		}
		doc.Sections = append(doc.Sections, section)
	}
	for _, bib := range payload.BibEntries {
		title := compactText(bib.Title)
		doi := normalizeReferenceDOI(bib.DOI)
		if title == "" && doi == "" {
			continue
		}
		doc.References = append(doc.References, Reference{Title: title, DOI: doi, Raw: compactText(bib.RawText)})
	}
	if doc.Title == "" {
		doc.Warnings = append(doc.Warnings, "missing title")
	}
	return EnrichParsedDocumentModel(doc), nil
}

type s2orcDocument struct {
	Title      string                   `json:"title"`
	Abstract   string                   `json:"abstract"`
	BodyText   []s2orcParagraph         `json:"body_text"`
	BibEntries map[string]s2orcBibEntry `json:"bib_entries"`
}

type s2orcParagraph struct {
	Section string `json:"section"`
	Text    string `json:"text"`
}

type s2orcBibEntry struct {
	Title   string `json:"title"`
	DOI     string `json:"doi"`
	RawText string `json:"raw_text"`
}
