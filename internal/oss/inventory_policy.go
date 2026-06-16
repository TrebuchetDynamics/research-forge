package oss

import (
	"fmt"
	"strings"
	"time"
)

// InventoryPolicyOptions configures policy checks over refreshed inventory metadata.
type InventoryPolicyOptions struct {
	StaleAfterMonths int
	Now              time.Time
}

// InventoryPolicyResult reports stale/license/archive policy issues.
type InventoryPolicyResult struct {
	EntryCount int      `json:"entryCount"`
	Issues     []string `json:"issues"`
	Markdown   string   `json:"markdown"`
}

// Contains reports whether an issue contains the provided substring.
func (r InventoryPolicyResult) Contains(substr string) bool {
	for _, issue := range r.Issues {
		if strings.Contains(issue, substr) {
			return true
		}
	}
	return false
}

// CheckInventoryPolicy evaluates refreshed OSS inventory metadata for governance risks.
func CheckInventoryPolicy(manifest InventoryManifest, opts InventoryPolicyOptions) InventoryPolicyResult {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	months := opts.StaleAfterMonths
	if months <= 0 {
		months = 18
	}
	result := InventoryPolicyResult{EntryCount: len(manifest.Entries)}
	for _, entry := range manifest.Entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			id = strings.TrimSpace(entry.Name)
		}
		if entry.Archived {
			result.Issues = append(result.Issues, id+": repository is archived")
		}
		license := strings.TrimSpace(entry.LicenseSPDX)
		if license == "" || strings.EqualFold(license, "NOASSERTION") || strings.EqualFold(license, "unknown") {
			result.Issues = append(result.Issues, id+": missing licenseSPDX")
		}
		if isCopyleftLicense(license) && entry.Disposition != "adapter-only" && entry.Disposition != "pattern-reference" {
			result.Issues = append(result.Issues, fmt.Sprintf("%s: copyleft license %s requires adapter-only or pattern-reference disposition", id, license))
		}
		if pushed := strings.TrimSpace(entry.PushedAt); pushed != "" {
			pushedAt, err := time.Parse(time.RFC3339, pushed)
			if err != nil {
				result.Issues = append(result.Issues, id+": invalid pushedAt "+pushed)
			} else if pushedAt.Before(now.AddDate(0, -months, 0)) {
				result.Issues = append(result.Issues, fmt.Sprintf("%s: stale, last pushed at %s exceeds %d months", id, pushedAt.Format(time.RFC3339), months))
			}
		}
	}
	result.Markdown = inventoryPolicyMarkdown(result)
	return result
}

func isCopyleftLicense(license string) bool {
	license = strings.ToUpper(strings.TrimSpace(license))
	return strings.HasPrefix(license, "GPL-") || strings.HasPrefix(license, "AGPL-") || strings.HasPrefix(license, "LGPL-")
}

func inventoryPolicyMarkdown(result InventoryPolicyResult) string {
	var b strings.Builder
	b.WriteString("## OSS inventory policy issues\n\n")
	fmt.Fprintf(&b, "Entries checked: %d\n\n", result.EntryCount)
	if len(result.Issues) == 0 {
		b.WriteString("No policy issues found.\n")
		return b.String()
	}
	for _, issue := range result.Issues {
		fmt.Fprintf(&b, "- %s\n", issue)
	}
	return b.String()
}
