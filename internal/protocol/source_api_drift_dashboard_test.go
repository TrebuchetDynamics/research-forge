package protocol

import (
	"testing"
	"time"
)

func TestSourceAPIDriftDashboardCoversCoreConnectorsShapeChangesAndProvenance(t *testing.T) {
	registry := DefaultConnectorCapabilityRegistry()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	previous := NewLiveSmokeSnapshot(registry, now.Add(-24*time.Hour))
	previous.UpsertResult(ConnectorLiveSmokeResult{ConnectorID: "openalex", Status: LiveSmokePass, CheckedAt: now.Add(-24 * time.Hour), ObservedFields: []string{"source", "query", "work_id", "raw_ref", "old_field"}})
	current := NewLiveSmokeSnapshot(registry, now)
	current.UpsertResult(ConnectorLiveSmokeResult{ConnectorID: "openalex", Status: LiveSmokePass, CheckedAt: now, ObservedFields: []string{"source", "query", "work_id", "raw_ref", "new_field"}})
	current.UpsertResult(ConnectorLiveSmokeResult{ConnectorID: "semantic-scholar", Status: LiveSmokeFail, CheckedAt: now, Message: "429", ObservedFields: []string{"source"}})
	dashboard := BuildSourceAPIDriftDashboard(registry, current, &previous, now, map[string]string{"semantic-scholar": "data/provenance.jsonl#evt-semantic"})
	for _, connector := range []string{"openalex", "semantic-scholar", "pubmed", "europepmc", "crossref", "arxiv", "unpaywall"} {
		if _, ok := dashboard.Entry(connector); !ok {
			t.Fatalf("missing connector %s in %#v", connector, dashboard.Entries)
		}
	}
	openalex, _ := dashboard.Entry("openalex")
	if !entryHasAlert(openalex, "response_shape_changed") {
		t.Fatalf("missing shape-change alert: %#v", openalex.Alerts)
	}
	semantic, _ := dashboard.Entry("semantic-scholar")
	if semantic.ProvenanceRef != "data/provenance.jsonl#evt-semantic" || !entryHasAlert(semantic, "failing") {
		t.Fatalf("semantic entry = %#v", semantic)
	}
}

func entryHasAlert(entry SourceAPIDriftDashboardEntry, kind string) bool {
	for _, alert := range entry.Alerts {
		if alert.Kind == kind {
			return true
		}
	}
	return false
}
