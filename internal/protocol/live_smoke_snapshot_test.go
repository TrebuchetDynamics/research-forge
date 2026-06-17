package protocol

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLiveSmokeSnapshotStorageAndAlertsCoverAllConnectors(t *testing.T) {
	registry := DefaultConnectorCapabilityRegistry()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	snapshot := NewLiveSmokeSnapshot(registry, now)
	if len(snapshot.Results) != len(registry.Connectors) {
		t.Fatalf("snapshot results = %d, want %d", len(snapshot.Results), len(registry.Connectors))
	}
	snapshot.UpsertResult(ConnectorLiveSmokeResult{ConnectorID: "openalex", Status: LiveSmokePass, CheckedAt: now, Message: "ok", EndpointFingerprint: "works:v1", ObservedFields: []string{"source", "query", "work_id", "raw_ref"}})
	snapshot.UpsertResult(ConnectorLiveSmokeResult{ConnectorID: "semantic-scholar", Status: LiveSmokeFail, CheckedAt: now, Message: "429", EndpointFingerprint: "papers:v1"})

	path := filepath.Join(t.TempDir(), "latest.json")
	if err := SaveLiveSmokeSnapshot(path, snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	loaded, err := LoadLiveSmokeSnapshot(path)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if loaded.MustResult("openalex").Status != LiveSmokePass {
		t.Fatalf("openalex result not persisted: %#v", loaded.MustResult("openalex"))
	}

	alerts := ConnectorLiveSmokeAlerts(registry, loaded, now.Add(31*24*time.Hour))
	if !hasConnectorAlert(alerts, "semantic-scholar", "failing") {
		t.Fatalf("missing failing semantic-scholar alert: %#v", alerts)
	}
	if !hasConnectorAlert(alerts, "openalex", "stale") {
		t.Fatalf("missing stale openalex alert: %#v", alerts)
	}
	if !hasConnectorAlert(alerts, "crossref", "missing") {
		t.Fatalf("missing crossref missing alert: %#v", alerts)
	}
}

func TestConnectorLiveSmokeAlertsDetectAPIFieldDrift(t *testing.T) {
	registry := DefaultConnectorCapabilityRegistry()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	snapshot := NewLiveSmokeSnapshot(registry, now)
	snapshot.UpsertResult(ConnectorLiveSmokeResult{ConnectorID: "openalex", Status: LiveSmokePass, CheckedAt: now, Message: "ok", ObservedFields: []string{"source"}})
	alerts := ConnectorLiveSmokeAlerts(registry, snapshot, now)
	if !hasConnectorAlert(alerts, "openalex", "api_drift") {
		t.Fatalf("missing api drift alert: %#v", alerts)
	}
}

func hasConnectorAlert(alerts []ConnectorLiveSmokeAlert, connector, kind string) bool {
	for _, alert := range alerts {
		if alert.ConnectorID == connector && alert.Kind == kind {
			return true
		}
	}
	return false
}
