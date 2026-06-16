package oss

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// InventoryDriftResult reports mismatches between manifest entries and inventory notes.
type InventoryDriftResult struct {
	EntryCount int      `json:"entryCount"`
	NoteCount  int      `json:"noteCount"`
	Issues     []string `json:"issues"`
	Markdown   string   `json:"markdown"`
}

// Contains reports whether an issue contains the provided substring.
func (r InventoryDriftResult) Contains(substr string) bool {
	for _, issue := range r.Issues {
		if strings.Contains(issue, substr) {
			return true
		}
	}
	return false
}

// CheckInventoryDrift compares manifest metadata with referenced Markdown note content.
func CheckInventoryDrift(manifestPath string) (InventoryDriftResult, error) {
	manifest, err := LoadInventoryManifest(manifestPath)
	if err != nil {
		return InventoryDriftResult{}, err
	}
	base := filepath.Dir(manifestPath)
	result := InventoryDriftResult{EntryCount: len(manifest.Entries)}
	referenced := map[string]string{}
	for _, entry := range manifest.Entries {
		note := strings.TrimSpace(entry.Note)
		if note == "" || filepath.IsAbs(note) || strings.Contains(filepath.Clean(note), "..") {
			continue
		}
		referenced[filepath.Clean(note)] = entry.ID
		data, err := os.ReadFile(filepath.Join(base, note))
		if err != nil {
			continue
		}
		metadata := parseInventoryNoteMetadata(string(data))
		id := inventoryDriftEntryID(entry)
		if heading := metadata["heading"]; heading != "" && !inventoryHeadingMatches(heading, entry.Name) {
			result.Issues = append(result.Issues, fmt.Sprintf("%s: note heading %q does not match manifest name %q", id, heading, entry.Name))
		}
		compareInventoryNoteField(&result, id, "area", metadata["area"], entry.Area)
		compareInventoryNoteField(&result, id, "disposition", metadata["disposition"], entry.Disposition)
		compareInventoryNoteField(&result, id, "repository", metadata["repository"], entry.Repository)
		compareInventoryNoteField(&result, id, "url", metadata["url"], entry.URL)
		if notePolicy := metadata["license policy"]; notePolicy != "" && !sameInventoryText(notePolicy, entry.LicensePolicy) {
			result.Issues = append(result.Issues, fmt.Sprintf("%s: note license policy does not match manifest licensePolicy", id))
		}
		if noteNext := metadata["next slice"]; noteNext != "" && !sameInventoryText(noteNext, entry.NextSlice) {
			result.Issues = append(result.Issues, fmt.Sprintf("%s: note next slice does not match manifest nextSlice", id))
		}
	}
	notes, err := filepath.Glob(filepath.Join(base, "*.md"))
	if err != nil {
		return InventoryDriftResult{}, err
	}
	sort.Strings(notes)
	for _, notePath := range notes {
		note := filepath.Base(notePath)
		if note == "README.md" {
			continue
		}
		result.NoteCount++
		if _, ok := referenced[note]; !ok {
			result.Issues = append(result.Issues, note+": note is not referenced by manifest")
		}
	}
	result.Markdown = inventoryDriftMarkdown(result)
	return result, nil
}

func inventoryDriftEntryID(entry InventoryEntry) string {
	if strings.TrimSpace(entry.ID) != "" {
		return strings.TrimSpace(entry.ID)
	}
	return strings.TrimSpace(entry.Name)
}

func compareInventoryNoteField(result *InventoryDriftResult, id, label, noteValue, manifestValue string) {
	if noteValue == "" || sameInventoryText(noteValue, manifestValue) {
		return
	}
	result.Issues = append(result.Issues, fmt.Sprintf("%s: note %s %q does not match manifest %s %q", id, label, noteValue, label, manifestValue))
}

var inventoryNoteFieldRE = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z ]+):\s*(.+?)\s*$`)

func parseInventoryNoteMetadata(markdown string) map[string]string {
	metadata := map[string]string{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && metadata["heading"] == "" {
			metadata["heading"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			continue
		}
		match := inventoryNoteFieldRE.FindStringSubmatch(trimmed)
		if len(match) == 3 {
			metadata[strings.ToLower(strings.TrimSpace(match[1]))] = strings.TrimSpace(match[2])
		}
	}
	return metadata
}

func inventoryHeadingMatches(heading, name string) bool {
	return sameInventoryText(heading, name) || sameInventoryText(heading, name+" study note")
}

func sameInventoryText(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " "))
}

func inventoryDriftMarkdown(result InventoryDriftResult) string {
	var b strings.Builder
	b.WriteString("## OSS inventory drift\n\n")
	fmt.Fprintf(&b, "Manifest entries: %d\n", result.EntryCount)
	fmt.Fprintf(&b, "Markdown notes: %d\n\n", result.NoteCount)
	if len(result.Issues) == 0 {
		b.WriteString("No drift found.\n")
		return b.String()
	}
	for _, issue := range result.Issues {
		fmt.Fprintf(&b, "- %s\n", issue)
	}
	return b.String()
}
