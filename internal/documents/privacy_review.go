package documents

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

type PrivacyLicensingReviewInput struct {
	Records         []library.PaperRecord `json:"records"`
	Assets          []DocumentAsset       `json:"assets"`
	ShareableReport string                `json:"shareableReport,omitempty"`
}

type PrivacyLicensingReview struct {
	SchemaVersion  string                  `json:"schemaVersion"`
	Issues         []PrivacyLicensingIssue `json:"issues"`
	Blocked        bool                    `json:"blocked"`
	Approved       bool                    `json:"approved"`
	Reviewer       string                  `json:"reviewer,omitempty"`
	ApprovalReason string                  `json:"approvalReason,omitempty"`
}

type PrivacyLicensingIssue struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	Target   string `json:"target"`
	Message  string `json:"message"`
}

func ReviewPrivacyLicensing(input PrivacyLicensingReviewInput) PrivacyLicensingReview {
	issues := []PrivacyLicensingIssue{}
	add := func(kind, severity, target, message string) {
		issues = append(issues, PrivacyLicensingIssue{Kind: kind, Severity: severity, Target: target, Message: message})
	}
	for _, record := range input.Records {
		for _, ref := range record.SourceRefs {
			m := ref.Metadata
			if strings.TrimSpace(m["attachment_files"]) != "" {
				add("imported_attachment", "warning", record.Title, "imported attachment metadata requires privacy/licensing review")
			}
			if strings.TrimSpace(m["note"]) != "" {
				add("imported_note", "warning", record.Title, "imported private notes require reviewer approval before sharing")
			}
			if strings.TrimSpace(m["annotations"]) != "" {
				add("imported_annotation", "warning", record.Title, "imported annotations require reviewer approval before sharing")
			}
			if strings.Contains(strings.ToLower(m["linked_file_privacy_check"]), "redacted") {
				add("local_path", "warning", record.Title, "linked local file paths were redacted and require review")
			}
		}
	}
	for _, asset := range input.Assets {
		if asset.LocalOnly || strings.HasPrefix(asset.LocalPath, "/") || strings.HasPrefix(asset.LocalPath, "file:") {
			add("local_path", "warning", asset.PaperID, "local document path requires privacy review")
		}
		if asset.LocalOnly || strings.EqualFold(asset.OAStatus, "closed") || strings.TrimSpace(asset.License) == "" {
			add("copyrighted_pdf", "critical", asset.PaperID, "copyright or missing-license PDF cannot be shared without explicit approval")
		}
	}
	if reportLeak(input.ShareableReport) {
		add("shareable_report", "critical", "shareable-report", "shareable report contains local paths or private notes and must be redacted/reviewed")
	}
	return PrivacyLicensingReview{SchemaVersion: "1", Issues: issues, Blocked: len(issues) > 0, Approved: false}
}

func ApprovePrivacyLicensing(review PrivacyLicensingReview, reviewer, reason string) PrivacyLicensingReview {
	review.Approved = true
	review.Blocked = false
	review.Reviewer = strings.TrimSpace(reviewer)
	review.ApprovalReason = strings.TrimSpace(reason)
	return review
}

func GuardPrivacyLicensing(review PrivacyLicensingReview) error {
	if len(review.Issues) > 0 && !review.Approved {
		return fmt.Errorf("privacy/licensing review approval required")
	}
	return nil
}

func reportLeak(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "/home/") || strings.Contains(lower, "/users/") || strings.Contains(lower, "file:") || strings.Contains(lower, "private note") || strings.Contains(lower, "secret")
}
