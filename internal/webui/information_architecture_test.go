package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardInformationArchitectureDocumentsRoutesPartialsViewModelsFallbacksJobsAndOwnership(t *testing.T) {
	ia := BuildDashboardInformationArchitecture()
	if ia.SchemaVersion != "1" || len(ia.Routes) == 0 || len(ia.BackgroundJobs) == 0 || len(ia.OwnershipBoundaries) == 0 || len(ia.Diagram) == 0 {
		t.Fatalf("ia = %#v", ia)
	}
	for _, route := range []string{"/forge", "/workbenches", "/notebook", "/parsing", "/map", "/acquisition", "/retrieve", "/screening", "/evidence", "/analysis", "/report", "/package", "/connectors", "/dedupe", "/privacy", "/architecture"} {
		if !ia.HasRoute(route) {
			t.Fatalf("missing route %s", route)
		}
	}
	body := renderHandler(t, NewInformationArchitectureHandler(ia))
	for _, want := range []string{"Dashboard information architecture", "Diagram", "routes", "Partial endpoints", "View models", "No-JS fallbacks", "Background jobs", "Ownership boundaries", "ForgeHomeState", "PackageExportCenterState"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesInformationArchitecture(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body := httpGetBody(t, ts.URL+"/architecture")
	if !strings.Contains(body, "Dashboard information architecture") || !strings.Contains(body, "Ownership boundaries") {
		t.Fatalf("/architecture missing IA: %s", body)
	}
}
