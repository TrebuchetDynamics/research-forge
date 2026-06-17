package protocol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
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
	return os.WriteFile(path, payload, 0o644)
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
