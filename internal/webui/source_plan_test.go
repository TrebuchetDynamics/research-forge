package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSourcePlanningHandlerRendersCockpit(t *testing.T) {
	req := httptest.NewRequest("GET", "/sources?question=Do+catalysts+improve+hydrogen+evolution%3F&type=pico&population=hydrogen+evolution&intervention=catalysts&outcome=efficiency", nil)
	rec := httptest.NewRecorder()
	NewSourcePlanningHandler().ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"Source planning cockpit", "OpenAlex", "Semantic Scholar", "NASA ADS", "DOAJ", "CORE", "Zotero", "JabRef", "Local files", "Reviewer approval required", "rforge protocol plan-sources"} {
		if !strings.Contains(body, want) {
			t.Fatalf("source planning cockpit missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesSourcesRoute(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/sources?question=parser+comparison")
	if status != 200 {
		t.Fatalf("GET /sources status = %d", status)
	}
	if !strings.Contains(body, "Source planning cockpit") {
		t.Fatalf("/sources missing cockpit: %s", body)
	}
}
