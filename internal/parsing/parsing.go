package parsing

import "context"

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
	SchemaVersion string
	PaperID       string
	ParserName    string
	ParserVersion string
	Title         string
	Authors       []ParsedAuthor
	Abstract      string
	Sections      []Section
	References    []Reference
	Warnings      []string
}

// ParsedAuthor is a TEI-normalized author.
type ParsedAuthor struct {
	Given  string
	Family string
}

// Section is one parsed full-text section.
type Section struct {
	ID       string
	Title    string
	Passages []Passage
}

// Passage is one retrievable section passage.
type Passage struct {
	ID        string
	PaperID   string
	SectionID string
	Text      string
}

// Reference is one parsed bibliography reference.
type Reference struct {
	Title string
}
