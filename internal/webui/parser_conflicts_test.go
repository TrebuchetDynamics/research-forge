package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParserConflictReviewUIShowsCERMINEFallbackPolicy(t *testing.T) {
	state := BuildParserConflictReviewState()
	if !state.HasParser("cermine") || !state.ReviewerRequired || state.AutoAcceptConflicts {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewParserConflictReviewHandler(state))
	for _, want := range []string{"Parser conflict review", "CERMINE", "reviewer required", "No conflicting fields are auto-accepted", "rforge parse quality"} {
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
