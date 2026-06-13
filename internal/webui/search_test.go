package webui

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

func TestSearchHandlerRendersHTMXSearchScreen(t *testing.T) {
	state := ui.NewSearchFormState([]string{"openalex", "arxiv", "crossref"})
	state.Query = "artificial photosynthesis"

	req := httptest.NewRequest("GET", "/search", nil)
	rec := httptest.NewRecorder()

	NewSearchHandler(state).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Search papers",
		"hx-get=\"/search/results\"",
		"name=\"query\"",
		"value=\"artificial photosynthesis\"",
		"openalex",
		"arxiv",
		"crossref",
		"Loading",
		"No results yet",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("search screen missing %q:\n%s", want, body)
		}
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q", got)
	}
}
