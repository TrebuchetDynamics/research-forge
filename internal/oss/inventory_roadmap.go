package oss

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type InventoryRoadmapReport struct {
	Areas        map[string][]InventoryRoadmapSlice `json:"areas"`
	CoverageGaps []string                           `json:"coverageGaps"`
	Markdown     string                             `json:"markdown"`
}

type InventoryRoadmapSlice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	NextSlice string `json:"nextSlice"`
}

func BuildInventoryRoadmapReport(manifestPath, todoPath string) (InventoryRoadmapReport, error) {
	manifest, err := LoadInventoryManifest(manifestPath)
	if err != nil {
		return InventoryRoadmapReport{}, err
	}
	todoBytes, err := os.ReadFile(todoPath)
	if err != nil {
		return InventoryRoadmapReport{}, err
	}
	todo := string(todoBytes)
	report := InventoryRoadmapReport{Areas: map[string][]InventoryRoadmapSlice{}}
	base := filepath.Dir(manifestPath)
	referenced := map[string]bool{}
	for _, entry := range manifest.Entries {
		slice := InventoryRoadmapSlice{ID: entry.ID, Name: entry.Name, Note: entry.Note, NextSlice: entry.NextSlice}
		report.Areas[entry.Area] = append(report.Areas[entry.Area], slice)
		note := filepath.ToSlash(entry.Note)
		referenced[note] = true
		if note != "" && !strings.Contains(todo, note) && !strings.Contains(todo, entry.NextSlice) {
			report.CoverageGaps = append(report.CoverageGaps, fmt.Sprintf("%s: nextSlice not covered by TODO.md", note))
		}
	}
	notes, err := filepath.Glob(filepath.Join(base, "*.md"))
	if err != nil {
		return InventoryRoadmapReport{}, err
	}
	for _, notePath := range notes {
		note := filepath.ToSlash(filepath.Base(notePath))
		if note == "README.md" {
			continue
		}
		if !referenced[note] {
			report.CoverageGaps = append(report.CoverageGaps, fmt.Sprintf("%s: inventory note is not referenced by manifest", note))
		}
	}
	for area := range report.Areas {
		sort.Slice(report.Areas[area], func(i, j int) bool { return report.Areas[area][i].ID < report.Areas[area][j].ID })
	}
	sort.Strings(report.CoverageGaps)
	report.Markdown = inventoryRoadmapMarkdown(report)
	return report, nil
}

func (r InventoryRoadmapReport) ContainsGap(substr string) bool {
	for _, gap := range r.CoverageGaps {
		if strings.Contains(gap, substr) {
			return true
		}
	}
	return false
}

func inventoryRoadmapMarkdown(report InventoryRoadmapReport) string {
	var b strings.Builder
	b.WriteString("# OSS inventory roadmap\n\n")
	areas := make([]string, 0, len(report.Areas))
	for area := range report.Areas {
		areas = append(areas, area)
	}
	sort.Strings(areas)
	for _, area := range areas {
		fmt.Fprintf(&b, "## %s\n\n", area)
		b.WriteString("| Tool | Note | Next slice |\n| --- | --- | --- |\n")
		for _, slice := range report.Areas[area] {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", escapeMarkdownCell(slice.Name), escapeMarkdownCell(slice.Note), escapeMarkdownCell(slice.NextSlice))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Coverage gaps\n\n")
	if len(report.CoverageGaps) == 0 {
		b.WriteString("No TODO coverage gaps found.\n")
		return b.String()
	}
	for _, gap := range report.CoverageGaps {
		fmt.Fprintf(&b, "- %s\n", gap)
	}
	return b.String()
}
