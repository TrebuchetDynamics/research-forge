package oss

import (
	"fmt"
	"sort"
	"strings"
)

// InventoryReportOptions filters an OSS inventory ecosystem report.
type InventoryReportOptions struct {
	Area string
}

// InventoryReport is a generated Markdown ecosystem report.
type InventoryReport struct {
	EntryCount int    `json:"entryCount"`
	Area       string `json:"area,omitempty"`
	Markdown   string `json:"markdown"`
}

// BuildInventoryReport renders a deterministic Markdown report from an inventory manifest.
func BuildInventoryReport(manifest InventoryManifest, opts InventoryReportOptions) InventoryReport {
	area := strings.TrimSpace(opts.Area)
	entries := make([]InventoryEntry, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if area != "" && entry.Area != area {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Area == entries[j].Area {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Area < entries[j].Area
	})

	var b strings.Builder
	b.WriteString("# OSS inventory report\n\n")
	if area != "" {
		fmt.Fprintf(&b, "Area: %s\n\n", area)
	} else {
		b.WriteString("Area: all\n\n")
	}
	fmt.Fprintf(&b, "Entries: %d\n\n", len(entries))
	b.WriteString("| Tool | Area | Disposition | Stars | Forks | License | Status | Risk | Next slice | Note |\n")
	b.WriteString("| --- | --- | --- | ---: | ---: | --- | --- | --- | --- | --- |\n")
	for _, entry := range entries {
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %d | %s | %s | %s | %s | %s |\n",
			escapeMarkdownCell(entry.Name), escapeMarkdownCell(entry.Area), escapeMarkdownCell(entry.Disposition), entry.Stars, entry.Forks,
			escapeMarkdownCell(entry.LicenseSPDX), escapeMarkdownCell(inventoryArchiveStatus(entry.Archived)),
			escapeMarkdownCell(entry.Risk), escapeMarkdownCell(entry.NextSlice), escapeMarkdownCell(entry.Note))
	}
	return InventoryReport{EntryCount: len(entries), Area: area, Markdown: b.String()}
}

func inventoryArchiveStatus(archived bool) string {
	if archived {
		return "archived"
	}
	return "active"
}

func escapeMarkdownCell(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "|", "\\|")
}
