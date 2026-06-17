package oss

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInventoryRoadmapReportGroupsNextSlicesByAreaAndFindsCoverageGaps(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("# Alpha\nArea: retrieval\nNext slice: Add alpha retrieval."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.md"), []byte("# Beta\nArea: screening\nNext slice: Add beta screening."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"schemaVersion":"1","entries":[{"id":"alpha","name":"Alpha","area":"retrieval","disposition":"pattern-reference","licensePolicy":"study","note":"alpha.md","risk":"low","nextSlice":"Add alpha retrieval."},{"id":"beta","name":"Beta","area":"screening","disposition":"pattern-reference","licensePolicy":"study","note":"beta.md","risk":"low","nextSlice":"Add beta screening."}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	todoPath := filepath.Join(dir, "TODO.md")
	if err := os.WriteFile(todoPath, []byte("- [ ] Add alpha retrieval (`opensource/inventory/alpha.md`).\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := BuildInventoryRoadmapReport(manifestPath, todoPath)
	if err != nil {
		t.Fatalf("BuildInventoryRoadmapReport: %v", err)
	}
	if len(report.Areas) != 2 || report.Areas["retrieval"][0].NextSlice != "Add alpha retrieval." {
		t.Fatalf("areas = %#v", report.Areas)
	}
	if !report.ContainsGap("beta.md") {
		t.Fatalf("expected beta coverage gap: %#v", report.CoverageGaps)
	}
	if len(report.SuggestedSlices) == 0 || !roadmapHasSuggestion(report, "beta", "test") {
		t.Fatalf("suggested slices missing: %#v", report.SuggestedSlices)
	}
	if !strings.Contains(report.Markdown, "## retrieval") || !strings.Contains(report.Markdown, "Add alpha retrieval") || !strings.Contains(report.Markdown, "Suggested implementation slices") || strings.Contains(report.Markdown, "[x]") {
		t.Fatalf("markdown:\n%s", report.Markdown)
	}
}

func roadmapHasSuggestion(report InventoryRoadmapReport, id, kind string) bool {
	for _, suggestion := range report.SuggestedSlices {
		if suggestion.ID == id && suggestion.Kind == kind {
			return true
		}
	}
	return false
}
