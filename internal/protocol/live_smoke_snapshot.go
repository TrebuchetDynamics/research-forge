package protocol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

const (
	LiveSmokePass    = "pass"
	LiveSmokeFail    = "fail"
	LiveSmokeSkipped = "skipped"
	LiveSmokeMissing = "missing"
)

const liveSmokeStaleAfter = 30 * 24 * time.Hour

type ConnectorLiveSmokeSnapshot struct {
	SchemaVersion string                     `json:"schemaVersion"`
	CapturedAt    time.Time                  `json:"capturedAt"`
	Results       []ConnectorLiveSmokeResult `json:"results"`
}

type ConnectorLiveSmokeResult struct {
	ConnectorID         string    `json:"connectorId"`
	Label               string    `json:"label,omitempty"`
	Status              string    `json:"status"`
	CheckedAt           time.Time `json:"checkedAt"`
	Message             string    `json:"message,omitempty"`
	EndpointFingerprint string    `json:"endpointFingerprint,omitempty"`
	ObservedFields      []string  `json:"observedFields,omitempty"`
}

type ConnectorLiveSmokeAlert struct {
	ConnectorID string `json:"connectorId"`
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
}

type SourceAPIDriftDashboard struct {
	SchemaVersion string                         `json:"schemaVersion"`
	CapturedAt    time.Time                      `json:"capturedAt"`
	Entries       []SourceAPIDriftDashboardEntry `json:"entries"`
}

type SourceAPIDriftDashboardEntry struct {
	ConnectorID         string                    `json:"connectorId"`
	Label               string                    `json:"label"`
	Status              string                    `json:"status"`
	CheckedAt           time.Time                 `json:"checkedAt"`
	EndpointFingerprint string                    `json:"endpointFingerprint,omitempty"`
	ObservedFields      []string                  `json:"observedFields,omitempty"`
	PreviousFields      []string                  `json:"previousFields,omitempty"`
	AddedFields         []string                  `json:"addedFields,omitempty"`
	RemovedFields       []string                  `json:"removedFields,omitempty"`
	Alerts              []ConnectorLiveSmokeAlert `json:"alerts,omitempty"`
	ProvenanceRef       string                    `json:"provenanceRef"`
}

func NewLiveSmokeSnapshot(registry ConnectorCapabilityRegistry, capturedAt time.Time) ConnectorLiveSmokeSnapshot {
	results := make([]ConnectorLiveSmokeResult, 0, len(registry.Connectors))
	for _, connector := range registry.Connectors {
		results = append(results, ConnectorLiveSmokeResult{ConnectorID: connector.ID, Label: connector.Label, Status: LiveSmokeMissing, CheckedAt: capturedAt, Message: "no live-smoke result recorded"})
	}
	return ConnectorLiveSmokeSnapshot{SchemaVersion: "1", CapturedAt: capturedAt, Results: results}
}

func (s *ConnectorLiveSmokeSnapshot) UpsertResult(result ConnectorLiveSmokeResult) {
	if result.Status == "" {
		result.Status = LiveSmokeMissing
	}
	for i := range s.Results {
		if s.Results[i].ConnectorID == result.ConnectorID {
			if result.Label == "" {
				result.Label = s.Results[i].Label
			}
			s.Results[i] = result
			return
		}
	}
	s.Results = append(s.Results, result)
	sort.Slice(s.Results, func(i, j int) bool { return s.Results[i].ConnectorID < s.Results[j].ConnectorID })
}

func (s ConnectorLiveSmokeSnapshot) Result(id string) (ConnectorLiveSmokeResult, bool) {
	for _, result := range s.Results {
		if result.ConnectorID == id {
			return result, true
		}
	}
	return ConnectorLiveSmokeResult{}, false
}

func (s ConnectorLiveSmokeSnapshot) MustResult(id string) ConnectorLiveSmokeResult {
	result, _ := s.Result(id)
	return result
}

func SaveLiveSmokeSnapshot(path string, snapshot ConnectorLiveSmokeSnapshot) error {
	if path == "" {
		return fmt.Errorf("snapshot path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return filetxn.Replace(path, payload, 0o644)
}

func LoadLiveSmokeSnapshot(path string) (ConnectorLiveSmokeSnapshot, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return ConnectorLiveSmokeSnapshot{}, err
	}
	var snapshot ConnectorLiveSmokeSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return ConnectorLiveSmokeSnapshot{}, err
	}
	return snapshot, nil
}

func BuildSourceAPIDriftDashboard(registry ConnectorCapabilityRegistry, current ConnectorLiveSmokeSnapshot, previous *ConnectorLiveSmokeSnapshot, now time.Time, provenanceRefs map[string]string) SourceAPIDriftDashboard {
	wanted := map[string]bool{"openalex": true, "semantic-scholar": true, "pubmed": true, "europepmc": true, "crossref": true, "arxiv": true, "unpaywall": true}
	baseAlerts := ConnectorLiveSmokeAlerts(registry, current, now)
	entries := []SourceAPIDriftDashboardEntry{}
	for _, connector := range registry.Connectors {
		if !wanted[connector.ID] {
			continue
		}
		result, ok := current.Result(connector.ID)
		if !ok {
			result = ConnectorLiveSmokeResult{ConnectorID: connector.ID, Label: connector.Label, Status: LiveSmokeMissing, CheckedAt: current.CapturedAt}
		}
		entry := SourceAPIDriftDashboardEntry{ConnectorID: connector.ID, Label: connector.Label, Status: result.Status, CheckedAt: result.CheckedAt, EndpointFingerprint: result.EndpointFingerprint, ObservedFields: append([]string{}, result.ObservedFields...), ProvenanceRef: provenanceRefs[connector.ID]}
		if entry.ProvenanceRef == "" {
			entry.ProvenanceRef = "data/provenance.jsonl#connector=" + connector.ID
		}
		for _, alert := range baseAlerts {
			if alert.ConnectorID == connector.ID {
				entry.Alerts = append(entry.Alerts, alert)
			}
		}
		if previous != nil {
			if prior, ok := previous.Result(connector.ID); ok {
				entry.PreviousFields = append([]string{}, prior.ObservedFields...)
				entry.AddedFields = diffStrings(result.ObservedFields, prior.ObservedFields)
				entry.RemovedFields = diffStrings(prior.ObservedFields, result.ObservedFields)
				if len(entry.AddedFields) > 0 || len(entry.RemovedFields) > 0 {
					entry.Alerts = append(entry.Alerts, ConnectorLiveSmokeAlert{ConnectorID: connector.ID, Label: connector.Label, Kind: "response_shape_changed", Severity: "warning", Message: "live-smoke observed fields changed since previous snapshot"})
				}
			}
		}
		entries = append(entries, entry)
	}
	return SourceAPIDriftDashboard{SchemaVersion: "1", CapturedAt: now, Entries: entries}
}

func (d SourceAPIDriftDashboard) Entry(connectorID string) (SourceAPIDriftDashboardEntry, bool) {
	for _, entry := range d.Entries {
		if entry.ConnectorID == connectorID {
			return entry, true
		}
	}
	return SourceAPIDriftDashboardEntry{}, false
}

func diffStrings(left, right []string) []string {
	seen := map[string]bool{}
	for _, value := range right {
		seen[value] = true
	}
	out := []string{}
	for _, value := range left {
		if !seen[value] {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func ConnectorLiveSmokeAlerts(registry ConnectorCapabilityRegistry, snapshot ConnectorLiveSmokeSnapshot, now time.Time) []ConnectorLiveSmokeAlert {
	alerts := []ConnectorLiveSmokeAlert{}
	for _, connector := range registry.Connectors {
		result, ok := snapshot.Result(connector.ID)
		if !ok || result.Status == LiveSmokeMissing {
			alerts = append(alerts, ConnectorLiveSmokeAlert{ConnectorID: connector.ID, Label: connector.Label, Kind: "missing", Severity: "warning", Message: "no live-smoke snapshot recorded for connector"})
			continue
		}
		if result.Status == LiveSmokeFail {
			alerts = append(alerts, ConnectorLiveSmokeAlert{ConnectorID: connector.ID, Label: connector.Label, Kind: "failing", Severity: "critical", Message: result.Message})
		}
		if result.Status == LiveSmokePass && now.Sub(result.CheckedAt) > liveSmokeStaleAfter {
			alerts = append(alerts, ConnectorLiveSmokeAlert{ConnectorID: connector.ID, Label: connector.Label, Kind: "stale", Severity: "warning", Message: "live-smoke snapshot is older than 30 days"})
		}
		missingFields := missingProvenanceFields(connector.ProvenanceFields, result.ObservedFields)
		if result.Status == LiveSmokePass && len(result.ObservedFields) > 0 && len(missingFields) > 0 {
			alerts = append(alerts, ConnectorLiveSmokeAlert{ConnectorID: connector.ID, Label: connector.Label, Kind: "api_drift", Severity: "warning", Message: "observed live-smoke fields missing expected provenance fields: " + joinComma(missingFields)})
		}
	}
	return alerts
}

func missingProvenanceFields(expected, observed []string) []string {
	seen := map[string]bool{}
	for _, field := range observed {
		seen[field] = true
	}
	missing := []string{}
	for _, field := range expected {
		if !seen[field] {
			missing = append(missing, field)
		}
	}
	return missing
}

func joinComma(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ", "
		}
		out += value
	}
	return out
}
