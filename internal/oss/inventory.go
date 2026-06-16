package oss

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InventoryManifest is the committed machine-readable OSS inventory index.
type InventoryManifest struct {
	SchemaVersion string           `json:"schemaVersion"`
	Entries       []InventoryEntry `json:"entries"`
}

// InventoryEntry describes one studied OSS tool/source and its governance metadata.
type InventoryEntry struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Repository          string `json:"repository,omitempty"`
	URL                 string `json:"url,omitempty"`
	Area                string `json:"area"`
	Disposition         string `json:"disposition"`
	LicensePolicy       string `json:"licensePolicy"`
	Note                string `json:"note"`
	Risk                string `json:"risk"`
	NextSlice           string `json:"nextSlice"`
	Stars               int    `json:"stars,omitempty"`
	Forks               int    `json:"forks,omitempty"`
	LicenseSPDX         string `json:"licenseSPDX,omitempty"`
	Archived            bool   `json:"archived,omitempty"`
	PushedAt            string `json:"pushedAt,omitempty"`
	MetadataRefreshedAt string `json:"metadataRefreshedAt,omitempty"`
}

// InventoryValidationResult reports deterministic inventory manifest validation.
type InventoryValidationResult struct {
	EntryCount int      `json:"entryCount"`
	Issues     []string `json:"issues"`
}

// Contains reports whether an issue contains the provided substring.
func (r InventoryValidationResult) Contains(substr string) bool {
	for _, issue := range r.Issues {
		if strings.Contains(issue, substr) {
			return true
		}
	}
	return false
}

// LoadInventoryManifest reads a machine-readable OSS inventory manifest.
func LoadInventoryManifest(path string) (InventoryManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return InventoryManifest{}, err
	}
	var manifest InventoryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return InventoryManifest{}, err
	}
	return manifest, nil
}

// ValidateInventoryManifest verifies the OSS inventory manifest and referenced notes.
func ValidateInventoryManifest(path string) (InventoryValidationResult, error) {
	manifest, err := LoadInventoryManifest(path)
	if err != nil {
		return InventoryValidationResult{}, err
	}
	result := InventoryValidationResult{EntryCount: len(manifest.Entries)}
	if strings.TrimSpace(manifest.SchemaVersion) == "" {
		result.Issues = append(result.Issues, "manifest missing schemaVersion")
	}
	seen := map[string]bool{}
	base := filepath.Dir(path)
	for i, entry := range manifest.Entries {
		prefix := fmt.Sprintf("entries[%d]", i)
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			result.Issues = append(result.Issues, prefix+" missing id")
		} else if seen[id] {
			result.Issues = append(result.Issues, prefix+" duplicate id "+id)
		}
		seen[id] = true
		if strings.TrimSpace(entry.Name) == "" {
			result.Issues = append(result.Issues, prefix+" missing name")
		}
		if strings.TrimSpace(entry.Area) == "" {
			result.Issues = append(result.Issues, prefix+" missing area")
		}
		if !validInventoryDisposition(entry.Disposition) {
			result.Issues = append(result.Issues, prefix+" missing disposition")
		}
		if strings.TrimSpace(entry.LicensePolicy) == "" {
			result.Issues = append(result.Issues, prefix+" missing licensePolicy")
		}
		if strings.TrimSpace(entry.Risk) == "" {
			result.Issues = append(result.Issues, prefix+" missing risk")
		}
		if strings.TrimSpace(entry.NextSlice) == "" {
			result.Issues = append(result.Issues, prefix+" missing nextSlice")
		}
		note := strings.TrimSpace(entry.Note)
		if note == "" {
			result.Issues = append(result.Issues, prefix+" missing note")
			continue
		}
		if filepath.IsAbs(note) || strings.Contains(filepath.Clean(note), "..") {
			result.Issues = append(result.Issues, prefix+" note path must stay within inventory")
			continue
		}
		if _, err := os.Stat(filepath.Join(base, note)); err != nil {
			if os.IsNotExist(err) {
				result.Issues = append(result.Issues, prefix+" note not found: "+note)
			} else {
				return result, err
			}
		}
	}
	return result, nil
}

func validInventoryDisposition(value string) bool {
	switch strings.TrimSpace(value) {
	case "pattern-reference", "adapter-only", "integrate", "needs-license-review", "avoid":
		return true
	default:
		return false
	}
}
