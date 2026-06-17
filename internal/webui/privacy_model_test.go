package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardPrivacyModelCoversSensitiveAssetClasses(t *testing.T) {
	model := BuildDashboardPrivacyModel()
	want := []string{"local-only paths", "copyrighted PDFs", "reviewer notes", "credentials", "embeddings", "cache files", "shareable report fields"}
	for _, name := range want {
		if !model.HasAsset(name) {
			t.Fatalf("missing asset %q in %#v", name, model.Assets)
		}
	}
	body := renderHandler(t, NewPrivacyModelHandler(model))
	for _, want := range []string{"Dashboard permissions/privacy model", "local-only paths", "copyrighted PDFs", "reviewer notes", "credentials", "embeddings", "cache files", "shareable report fields"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesPrivacyModel(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body := httpGetBody(t, ts.URL+"/privacy")
	if !strings.Contains(body, "Dashboard permissions/privacy model") || !strings.Contains(body, "shareable report fields") {
		t.Fatalf("/privacy missing model: %s", body)
	}
}
