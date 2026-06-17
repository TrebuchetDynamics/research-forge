package webui

import "strings"

type DashboardPrivacyModel struct {
	SchemaVersion string
	Assets        []PrivacyAsset
}
type PrivacyAsset struct{ Name, DefaultPermission, ExportRule, ReviewGate, UIBehavior string }

func BuildDashboardPrivacyModel() DashboardPrivacyModel {
	return DashboardPrivacyModel{SchemaVersion: "1", Assets: []PrivacyAsset{
		{"local-only paths", "project-local only", "redact absolute paths from shareable outputs", "privacy/licensing review", "display basename plus redaction warning"},
		{"copyrighted PDFs", "view in local browser only", "exclude unless license/shareability approved", "legal acquisition approval", "embed only via local /papers/{id}/pdf route"},
		{"reviewer notes", "private by default", "exclude or redact unless marked shareable", "reviewer approval", "show private-note badge"},
		{"credentials", "never display secret values", "never export", "connector credential review", "show presence/requirements only"},
		{"embeddings", "local payload policy required", "export only redacted checksums or approved vectors", "embedding egress/privacy approval", "show provider, dimensions, and payload policy"},
		{"cache files", "private local state", "exclude from packages and reports", "package redaction audit", "show excluded count only"},
		{"shareable report fields", "allowed after trace/redaction gates", "export only supported claims and approved metadata", "claim traceability panel", "block final export on weak/unresolved claims"},
	}}
}

func (m DashboardPrivacyModel) HasAsset(name string) bool {
	for _, asset := range m.Assets {
		if strings.EqualFold(asset.Name, name) {
			return true
		}
	}
	return false
}
