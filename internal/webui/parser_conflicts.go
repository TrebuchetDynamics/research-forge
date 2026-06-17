package webui

import (
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type ParserConflictReviewState struct {
	Parsers             []string
	ParserLabels        map[string]string
	Fields              []ParserArbitrationFieldRow
	Controls            []string
	ReviewerRequired    bool
	AutoAcceptConflicts bool
	CLIEquivalent       string
}

type ParserArbitrationFieldRow struct {
	Field      string
	ParserName string
	Confidence string
	RawText    string
	Offsets    string
	Warnings   string
}

func BuildParserConflictReviewState() ParserConflictReviewState {
	p := parsing.DefaultParserFallbackPolicy()
	parsers := []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"}
	labels := map[string]string{"grobid": "GROBID", "s2orc-doc2json": "S2ORC-style JSON", "papermage": "PaperMage", "cermine": "CERMINE", "science-parse": "Science Parse-style metadata", "anystyle": "Anystyle"}
	fields := []ParserArbitrationFieldRow{}
	for _, parser := range parsers {
		fields = append(fields, ParserArbitrationFieldRow{Field: "title", ParserName: labels[parser], Confidence: "per-field score", RawText: "raw text", Offsets: "source offsets", Warnings: "parser warnings"})
	}
	return ParserConflictReviewState{Parsers: parsers, ParserLabels: labels, Fields: fields, Controls: []string{"accept", "correct", "reject"}, ReviewerRequired: p.ReviewerRequired, AutoAcceptConflicts: p.AutoAcceptConflicts, CLIEquivalent: "rforge parse arbitrate --parsed <grobid.json> --parsed <s2orc.json> --parsed <papermage.json> --parsed <cermine.json> --parsed <science-parse.json> --parsed <anystyle.json> --out data/parser-arbitration.json"}
}
func (s ParserConflictReviewState) HasParser(name string) bool {
	name = strings.ToLower(name)
	for _, parser := range s.Parsers {
		if strings.ToLower(parser) == name {
			return true
		}
	}
	return false
}
func NewParserConflictReviewHandler(state ParserConflictReviewState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = parserConflictTemplate.Execute(w, state)
	})
}
