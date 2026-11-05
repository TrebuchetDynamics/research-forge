package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/model"
	textutil "github.com/TrebuchetDynamics/research-forge/internal/parsing/text"
	"sort"
	"strings"
)

// S2ORCJSONParser adapts s2orc-doc2json-like JSON into model.ParsedDocument.
type S2ORCJSONParser struct{ Version string }

func (p S2ORCJSONParser) Parse(ctx context.Context, data []byte, options model.ParseOptions) (model.ParsedDocument, error) {
	_ = ctx
	var payload s2orcDocument
	if err := json.Unmarshal(data, &payload); err != nil {
		return model.ParsedDocument{}, err
	}
	version := p.Version
	if version == "" {
		version = "s2orc-json"
	}
	doc := model.ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(options.PaperID), ParserName: "s2orc-doc2json", ParserVersion: version, Title: textutil.CompactSpace(payload.Title), Abstract: textutil.CompactSpace(payload.Abstract)}
	for i, paragraph := range payload.BodyText {
		sectionTitle := textutil.CompactSpace(paragraph.Section)
		if sectionTitle == "" {
			sectionTitle = "Body"
		}
		sectionID := fmt.Sprintf("%s-s2orc-sec-%d", doc.PaperID, i+1)
		text := textutil.CompactSpace(paragraph.Text)
		section := model.Section{ID: sectionID, Title: sectionTitle}
		if text != "" {
			section.Passages = append(section.Passages, model.Passage{ID: fmt.Sprintf("%s-p-1", sectionID), PaperID: doc.PaperID, SectionID: sectionID, Text: text})
		}
		doc.Sections = append(doc.Sections, section)
	}
	// JSON object iteration is unordered; reference order is part of the
	// normalized text stream and must be reproducible across reparses.
	bibIDs := make([]string, 0, len(payload.BibEntries))
	for id := range payload.BibEntries {
		bibIDs = append(bibIDs, id)
	}
	sort.Strings(bibIDs)
	for _, id := range bibIDs {
		bib := payload.BibEntries[id]
		title := textutil.CompactSpace(bib.Title)
		doi := textutil.NormalizeReferenceDOI(bib.DOI)
		if title == "" && doi == "" {
			continue
		}
		doc.References = append(doc.References, model.Reference{Title: title, DOI: doi, Raw: textutil.CompactSpace(bib.RawText)})
	}
	if doc.Title == "" {
		doc.Warnings = append(doc.Warnings, "missing title")
	}
	return model.EnrichParsedDocumentModel(doc), nil
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
