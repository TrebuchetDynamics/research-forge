package webui

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/protocol"
)

func TestConnectorHealthHandlerRendersLiveSmokeAlerts(t *testing.T) {
	project := t.TempDir()
	snapshot := protocol.NewLiveSmokeSnapshot(protocol.DefaultConnectorCapabilityRegistry(), time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	snapshot.UpsertResult(protocol.ConnectorLiveSmokeResult{ConnectorID: "openalex", Label: "OpenAlex", Status: protocol.LiveSmokeFail, CheckedAt: snapshot.CapturedAt, Message: "429"})
	if err := protocol.SaveLiveSmokeSnapshot(filepath.Join(project, "data", "source-live-smoke-snapshots", "latest.json"), snapshot); err != nil {
		t.Fatalf("save fixture: %v", err)
	}
	req := httptest.NewRequest("GET", "/connectors", nil)
	rec := httptest.NewRecorder()
	newConnectorHealthHandler(func() string { return project }).ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"Connector health/control center", "OpenAlex", "failing", "429", "Semantic Scholar", "missing"} {
		if !strings.Contains(body, want) {
			t.Fatalf("connector health missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesConnectorsRoute(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/connectors")
	if status != 200 {
		t.Fatalf("GET /connectors status = %d", status)
	}
	if !strings.Contains(body, "Connector health/control center") {
		t.Fatalf("/connectors missing health center: %s", body)
	}
}
