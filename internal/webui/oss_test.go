package webui

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

func TestOSSHandlerRendersRepositoryStudies(t *testing.T) {
	state := ui.NewOSSDashboardViewModel([]ui.OSSRow{{Name: "TrebuchetDynamics/research-forge"}, {Name: "openai/evals"}})

	req := httptest.NewRequest("GET", "/oss", nil)
	rec := httptest.NewRecorder()

	NewOSSHandler(state).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"OSS repository studies",
		"hx-get=\"/oss/repositories\"",
		"role=\"table\"",
		"Repository",
		"TrebuchetDynamics/research-forge",
		"openai/evals",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("OSS screen missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "No repository studies yet") {
		t.Fatalf("populated OSS dashboard should not render empty state:\n%s", body)
	}
}

func TestOSSHandlerRendersEmptyState(t *testing.T) {
	req := httptest.NewRequest("GET", "/oss", nil)
	rec := httptest.NewRecorder()

	NewOSSHandler(ui.NewOSSDashboardViewModel(nil)).ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "No repository studies yet") || !strings.Contains(body, "Run an OSS repository study") {
		t.Fatalf("empty OSS dashboard missing:\n%s", body)
	}
}
