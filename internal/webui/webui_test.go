package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShellHandlerRendersGoHTMXWorkspace(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	NewShellHandler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"ResearchForge",
		"/assets/htmx.min.js",
		"hx-boost=\"true\"",
		"/assets/researchforge.css",
		"Project dashboard",
		"CLI-generated artifacts",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("shell missing %q:\n%s", want, body)
		}
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q", got)
	}
}
