package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParserConflictReviewUIShowsCERMINEFallbackPolicy(t *testing.T) {
	state := BuildParserConflictReviewState()
	for _, parser := range []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"} {
		if !state.HasParser(parser) {
			t.Fatalf("missing parser %s in %#v", parser, state)
		}
	}
	if !state.ReviewerRequired || state.AutoAcceptConflicts || len(state.Fields) == 0 || len(state.Controls) != 3 {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewParserConflictReviewHandler(state))
	for _, want := range []string{"Parser conflict review", "GROBID", "S2ORC-style JSON", "PaperMage", "CERMINE", "Science Parse-style metadata", "Anystyle", "field-by-field", "confidence", "raw text", "offsets", "warnings", "accept", "correct", "reject", "reviewer required", "No conflicting fields are auto-accepted", "rforge parse arbitrate"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesParserConflictReview(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body := httpGetBody(t, ts.URL+"/parsing")
	if !strings.Contains(body, "Parser conflict review") || !strings.Contains(body, "CERMINE") {
		t.Fatalf("/parsing missing parser review: %s", body)
	}
}
