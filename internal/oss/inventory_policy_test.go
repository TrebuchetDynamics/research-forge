package oss

import (
	"strings"
	"testing"
	"time"
)

func TestCheckInventoryPolicyFlagsArchivedStaleMissingLicenseAndRiskyGPL(t *testing.T) {
	manifest := InventoryManifest{SchemaVersion: "1", Entries: []InventoryEntry{
		{ID: "archived", Name: "Archived", Disposition: "adapter-only", Archived: true, PushedAt: "2026-01-01T00:00:00Z", LicenseSPDX: "MIT"},
		{ID: "stale", Name: "Stale", Disposition: "pattern-reference", PushedAt: "2024-01-01T00:00:00Z", LicenseSPDX: "Apache-2.0"},
		{ID: "missing-license", Name: "Missing License", Disposition: "adapter-only", PushedAt: "2026-01-01T00:00:00Z"},
		{ID: "risky-gpl", Name: "Risky GPL", Disposition: "integrate", PushedAt: "2026-01-01T00:00:00Z", LicenseSPDX: "AGPL-3.0-only"},
		{ID: "safe-gpl", Name: "Safe GPL", Disposition: "pattern-reference", PushedAt: "2026-01-01T00:00:00Z", LicenseSPDX: "GPL-3.0-only"},
	}}
	result := CheckInventoryPolicy(manifest, InventoryPolicyOptions{StaleAfterMonths: 18, Now: time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)})
	if result.EntryCount != 5 {
		t.Fatalf("EntryCount = %d, want 5", result.EntryCount)
	}
	for _, want := range []string{"archived: repository is archived", "stale: stale", "missing-license: missing licenseSPDX", "risky-gpl: copyleft license AGPL-3.0-only requires adapter-only or pattern-reference disposition"} {
		if !result.Contains(want) {
			t.Fatalf("policy issues missing %q: %#v", want, result.Issues)
		}
	}
	if result.Contains("safe-gpl") {
		t.Fatalf("safe GPL pattern-reference should not be flagged: %#v", result.Issues)
	}
	if !strings.Contains(result.Markdown, "## OSS inventory policy issues") || !strings.Contains(result.Markdown, "risky-gpl") {
		t.Fatalf("markdown missing policy issue details:\n%s", result.Markdown)
	}
}
