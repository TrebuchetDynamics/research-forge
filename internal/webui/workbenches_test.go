package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkbenchesRouteCoversMetaAnalysisSpineHTMXWorkbenches(t *testing.T) {
	state := BuildWorkbenchIndexState()
	want := []string{"source planning", "import/dedupe", "legal acquisition", "parser arbitration", "retrieval tuning", "screening", "evidence extraction", "meta-analysis", "report traceability", "research map", "connector health", "reproducibility/export"}
	for _, label := range want {
		if !state.Has(label) {
			t.Fatalf("missing workbench %q in %#v", label, state.Workbenches)
		}
	}
	body := renderHandler(t, NewWorkbenchIndexHandler(state))
	for _, wantText := range []string{"HTMX workbenches", "source planning", "legal acquisition", "parser arbitration", "rforge protocol", "No-JS fallback"} {
		if !strings.Contains(body, wantText) {
			t.Fatalf("body missing %q:\n%s", wantText, body)
		}
	}
}

func TestRouterServesWorkbenchRoutes(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	for _, path := range []string{"/workbenches", "/acquisition", "/parsing", "/retrieve", "/evidence", "/analysis", "/report", "/map", "/package"} {
		body := httpGetBody(t, ts.URL+path)
		if !strings.Contains(body, "Workbench") && !strings.Contains(body, "HTMX workbenches") && !strings.Contains(body, "Research map cockpit") && !strings.Contains(body, "Parser conflict review") && !strings.Contains(body, "Legal full-text acquisition queue") && !strings.Contains(body, "Retrieval tuning") && !strings.Contains(body, "Evidence extraction grid") && !strings.Contains(body, "meta-analysis Workbench") && !strings.Contains(body, "Claim traceability panel") {
			t.Fatalf("%s missing workbench UI: %s", path, body)
		}
	}
}
