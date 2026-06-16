package parsing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PaperMageJSONParser adapts externally generated PaperMage-style layer JSON into ParsedDocument.
type PaperMageJSONParser struct{}

// Parse maps a conservative PaperMage layer export into ResearchForge's parsed-document model.
func (p PaperMageJSONParser) Parse(ctx context.Context, data []byte, options ParseOptions) (ParsedDocument, error) {
	_ = ctx
	var payload paperMagePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return ParsedDocument{}, err
	}
	doc := ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(options.PaperID), ParserName: "papermage", ParserVersion: "external-json", Title: compactParsingSpace(payload.Metadata.Title), Warnings: nonEmptyWarnings(payload.Warnings)}
	sectionIndex := map[string]int{}
	for i, section := range payload.Layers.Sections {
		title := compactParsingSpace(section.Text)
		if title == "" {
			title = fmt.Sprintf("Section %d", i+1)
		}
		id := fmt.Sprintf("s%d", len(doc.Sections)+1)
		sectionIndex[strings.ToLower(title)] = len(doc.Sections)
		doc.Sections = append(doc.Sections, Section{ID: id, Title: title})
	}
	for _, paragraph := range payload.Layers.Paragraphs {
		text := compactParsingSpace(paragraph.Text)
		if text == "" {
			continue
		}
		sectionName := compactParsingSpace(paragraph.Section)
		sectionPos, ok := sectionIndex[strings.ToLower(sectionName)]
		if !ok {
			sectionPos = len(doc.Sections)
			id := fmt.Sprintf("s%d", len(doc.Sections)+1)
			if sectionName == "" {
				sectionName = "Body"
			}
			sectionIndex[strings.ToLower(sectionName)] = sectionPos
			doc.Sections = append(doc.Sections, Section{ID: id, Title: sectionName})
		}
		section := &doc.Sections[sectionPos]
		passageID := fmt.Sprintf("%s-p%d", section.ID, len(section.Passages)+1)
		section.Passages = append(section.Passages, Passage{ID: passageID, PaperID: doc.PaperID, SectionID: section.ID, Text: text})
	}
	for _, ref := range payload.Layers.Bibliography {
		title := compactParsingSpace(ref.Title)
		doi := strings.TrimSpace(ref.DOI)
		raw := compactParsingSpace(ref.Raw)
		if title == "" && doi == "" && raw == "" {
			continue
		}
		doc.References = append(doc.References, Reference{Title: title, DOI: doi, Raw: raw, Confidence: ref.Confidence})
	}
	return doc, nil
}

type paperMagePayload struct {
	Metadata struct {
		Title string `json:"title"`
	} `json:"metadata"`
	Layers struct {
		Sections     []paperMageSection   `json:"sections"`
		Paragraphs   []paperMageParagraph `json:"paragraphs"`
		Bibliography []paperMageReference `json:"bibliography"`
	} `json:"layers"`
	Warnings []string `json:"warnings"`
}

type paperMageSection struct {
	Text string `json:"text"`
}

type paperMageParagraph struct {
	Section string `json:"section"`
	Text    string `json:"text"`
}

type paperMageReference struct {
	Title      string  `json:"title"`
	DOI        string  `json:"doi"`
	Raw        string  `json:"raw"`
	Confidence float64 `json:"confidence"`
}

func compactParsingSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func nonEmptyWarnings(warnings []string) []string {
	out := []string{}
	for _, warning := range warnings {
		if trimmed := compactParsingSpace(warning); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
