package webui

import (
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type ParserConflictReviewState struct {
	Parsers             []string
	ReviewerRequired    bool
	AutoAcceptConflicts bool
	CLIEquivalent       string
}

func BuildParserConflictReviewState() ParserConflictReviewState {
	p := parsing.DefaultParserFallbackPolicy()
	return ParserConflictReviewState{Parsers: p.Parsers, ReviewerRequired: p.ReviewerRequired, AutoAcceptConflicts: p.AutoAcceptConflicts, CLIEquivalent: "rforge parse quality --parsed <grobid.json> --parsed <cermine.json> --out data/parser-quality.json"}
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
