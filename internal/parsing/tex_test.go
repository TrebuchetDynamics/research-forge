package parsing

import (
	"context"
	"testing"
)

func TestTeXParserExtractsTitleAbstractAndSections(t *testing.T) {
	source := `\title{A TeX Study}
\begin{abstract}We study TeX parsing.\end{abstract}
\section{Introduction}
This is the introduction with \emph{markup}.
\section{Methods}
We parse source text.`

	doc, err := (TeXParser{Version: "test"}).Parse(context.Background(), []byte(source), ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if doc.ParserName != "tex" || doc.ParserVersion != "test" || doc.Title != "A TeX Study" || doc.Abstract != "We study TeX parsing." {
		t.Fatalf("doc = %#v", doc)
	}
	if len(doc.Sections) != 2 || doc.Sections[0].Title != "Introduction" || len(doc.Sections[0].Passages) != 1 {
		t.Fatalf("sections = %#v", doc.Sections)
	}
	if doc.Sections[0].Passages[0].Text != "This is the introduction with markup." {
		t.Fatalf("passage = %q", doc.Sections[0].Passages[0].Text)
	}
}

func TestTeXParserWarnsOnMissingTitle(t *testing.T) {
	doc, err := (TeXParser{}).Parse(context.Background(), []byte(`\section{Only}`), ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(doc.Warnings) != 1 || doc.Warnings[0] != "missing title" {
		t.Fatalf("warnings = %#v", doc.Warnings)
	}
}
