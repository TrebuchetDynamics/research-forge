package oss

import (
	"strings"
	"testing"
)

func TestBuildInventoryReportFiltersAreaAndIncludesRisksAndNextSlices(t *testing.T) {
	manifest := InventoryManifest{SchemaVersion: "1", Entries: []InventoryEntry{
		{ID: "openalex", Name: "OpenAlex", Area: "scholarly-graph-source", Disposition: "adapter-only", Note: "openalex.md", Risk: "Cursor state risk", NextSlice: "Paginated import", Stars: 123, Forks: 45, LicenseSPDX: "MIT", Archived: true},
		{ID: "semantic-scholar", Name: "Semantic Scholar", Area: "scholarly-graph-source", Disposition: "adapter-only", Note: "semantic-scholar.md", Risk: "Quota risk", NextSlice: "Resumable graph expansion"},
		{ID: "zotero", Name: "Zotero", Area: "reference-management", Disposition: "pattern-reference", Note: "zotero.md", Risk: "Attachment privacy", NextSlice: "Collections"},
	}}
	report := BuildInventoryReport(manifest, InventoryReportOptions{Area: "scholarly-graph-source"})
	if report.EntryCount != 2 {
		t.Fatalf("EntryCount = %d, want 2", report.EntryCount)
	}
	for _, want := range []string{"# OSS inventory report", "Area: scholarly-graph-source", "OpenAlex", "Semantic Scholar", "Cursor state risk", "Resumable graph expansion", "semantic-scholar.md", "MIT", "123", "archived"} {
		if !strings.Contains(report.Markdown, want) {
			t.Fatalf("report missing %q:\n%s", want, report.Markdown)
		}
	}
	if strings.Contains(report.Markdown, "Zotero") {
		t.Fatalf("area-filtered report included Zotero:\n%s", report.Markdown)
	}
}
