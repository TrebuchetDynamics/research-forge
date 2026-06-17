package parsing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// TeXParser extracts a lightweight ParsedDocument from arXiv TeX source.
type TeXParser struct{ Version string }

func (p TeXParser) Parse(ctx context.Context, data []byte, options ParseOptions) (ParsedDocument, error) {
	_ = ctx
	text := string(data)
	if strings.TrimSpace(text) == "" {
		return ParsedDocument{}, fmt.Errorf("tex source is empty")
	}
	version := p.Version
	if version == "" {
		version = "builtin"
	}
	doc := ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(options.PaperID), ParserName: "tex", ParserVersion: version}
	doc.Title = texCommandValue(text, "title")
	doc.Abstract = texEnvironmentValue(text, "abstract")
	sections := texSections(text)
	for i, section := range sections {
		sectionID := fmt.Sprintf("%s-tex-sec-%d", doc.PaperID, i+1)
		body := compactText(stripTeX(section.Body))
		parsedSection := Section{ID: sectionID, Title: section.Title}
		if body != "" {
			parsedSection.Passages = append(parsedSection.Passages, Passage{ID: fmt.Sprintf("%s-p-1", sectionID), PaperID: doc.PaperID, SectionID: sectionID, Text: body})
		}
		doc.Sections = append(doc.Sections, parsedSection)
	}
	if doc.Title == "" {
		doc.Warnings = append(doc.Warnings, "missing title")
	}
	return EnrichParsedDocumentModel(doc), nil
}

type texSection struct {
	Title string
	Body  string
}

func texCommandValue(text, command string) string {
	re := regexp.MustCompile(`(?s)\\` + regexp.QuoteMeta(command) + `\{(.*?)\}`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return compactText(stripTeX(match[1]))
}

func texEnvironmentValue(text, env string) string {
	re := regexp.MustCompile(`(?s)\\begin\{` + regexp.QuoteMeta(env) + `\}(.*?)\\end\{` + regexp.QuoteMeta(env) + `\}`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return compactText(stripTeX(match[1]))
}

func texSections(text string) []texSection {
	re := regexp.MustCompile(`(?s)\\section\{(.*?)\}`)
	matches := re.FindAllStringSubmatchIndex(text, -1)
	sections := []texSection{}
	for i, match := range matches {
		title := compactText(stripTeX(text[match[2]:match[3]]))
		bodyStart := match[1]
		bodyEnd := len(text)
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		}
		sections = append(sections, texSection{Title: title, Body: text[bodyStart:bodyEnd]})
	}
	return sections
}

func stripTeX(value string) string {
	value = regexp.MustCompile(`(?s)%.*?\n`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`\\[a-zA-Z]+\*?(\[[^\]]*\])?(\{([^{}]*)\})?`).ReplaceAllString(value, "$3")
	value = strings.NewReplacer("~", " ", "\\&", "&", "\\%", "%", "\\_", "_", "$", " ").Replace(value)
	return value
}
