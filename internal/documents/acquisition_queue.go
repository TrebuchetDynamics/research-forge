package documents

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

const (
	AcquisitionUseDownload = "download"
	AcquisitionUseArchive  = "archive"
)

type LegalAcquisitionQueue struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Items         []LegalAcquisitionQueueItem `json:"items"`
}

type LegalAcquisitionQueueItem struct {
	ID                       string `json:"id"`
	PaperTitle               string `json:"paperTitle"`
	DOI                      string `json:"doi,omitempty"`
	Source                   string `json:"source"`
	SourceURL                string `json:"sourceUrl"`
	ExpectedLocalPath        string `json:"expectedLocalPath"`
	License                  string `json:"license,omitempty"`
	OAStatus                 string `json:"oaStatus,omitempty"`
	Restricted               bool   `json:"restricted"`
	Shareable                bool   `json:"shareable"`
	ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
	ReviewerApproved         bool   `json:"reviewerApproved"`
	Reviewer                 string `json:"reviewer,omitempty"`
	ApprovalReason           string `json:"approvalReason,omitempty"`
	Provenance               string `json:"provenance"`
	Attribution              string `json:"attribution,omitempty"`
	RateLimitPolicy          string `json:"rateLimitPolicy,omitempty"`
	APIProvenance            string `json:"apiProvenance,omitempty"`
}

func BuildLegalAcquisitionQueue(projectPath string, comparison sources.OpenAccessCandidateComparison) LegalAcquisitionQueue {
	items := make([]LegalAcquisitionQueueItem, 0, len(comparison.Candidates))
	for i, candidate := range comparison.Candidates {
		restricted := acquisitionRestricted(candidate)
		items = append(items, LegalAcquisitionQueueItem{
			ID:                       fmt.Sprintf("acq-%03d", i+1),
			PaperTitle:               candidate.PaperTitle,
			DOI:                      candidate.DOI,
			Source:                   candidate.Source,
			SourceURL:                candidate.URL,
			ExpectedLocalPath:        expectedAcquisitionPath(projectPath, candidate),
			License:                  strings.TrimSpace(candidate.License),
			OAStatus:                 strings.TrimSpace(candidate.OAStatus),
			Restricted:               restricted,
			Shareable:                !restricted,
			ReviewerApprovalRequired: true,
			ReviewerApproved:         false,
			Provenance:               candidate.Provenance,
			Attribution:              strings.TrimSpace(candidate.Attribution),
			RateLimitPolicy:          strings.TrimSpace(candidate.RateLimitPolicy),
			APIProvenance:            strings.TrimSpace(candidate.APIProvenance),
		})
	}
	return LegalAcquisitionQueue{SchemaVersion: "1", Items: items}
}

func ApproveAcquisition(item LegalAcquisitionQueueItem, reviewer, reason string) LegalAcquisitionQueueItem {
	item.ReviewerApproved = true
	item.Reviewer = strings.TrimSpace(reviewer)
	item.ApprovalReason = strings.TrimSpace(reason)
	return item
}

func GuardAcquisition(item LegalAcquisitionQueueItem, use string) error {
	if item.ReviewerApprovalRequired && !item.ReviewerApproved {
		return fmt.Errorf("reviewer approval required before %s", use)
	}
	if strings.TrimSpace(item.SourceURL) == "" || strings.TrimSpace(item.ExpectedLocalPath) == "" {
		return fmt.Errorf("source URL and expected local path are required")
	}
	if use == AcquisitionUseArchive && (item.Restricted || !item.Shareable) {
		return fmt.Errorf("restricted acquisition candidate cannot be included in archive")
	}
	if !item.Restricted && strings.TrimSpace(item.License) == "" {
		return fmt.Errorf("shareable acquisition candidate requires license")
	}
	return nil
}

func acquisitionRestricted(candidate sources.OpenAccessCandidate) bool {
	status := strings.ToLower(strings.TrimSpace(candidate.OAStatus))
	return candidate.Source == "local" || status == "local-only" || strings.TrimSpace(candidate.License) == "" || strings.HasPrefix(strings.TrimSpace(candidate.URL), "/") || strings.HasPrefix(strings.TrimSpace(candidate.URL), "file:")
}

func expectedAcquisitionPath(projectPath string, candidate sources.OpenAccessCandidate) string {
	name := safeDocumentName(firstNonEmptyDocument(candidate.DOI, candidate.PaperTitle, filepath.Base(candidate.URL)))
	if candidate.Source == "local" || strings.HasPrefix(candidate.URL, "/") || strings.HasPrefix(candidate.URL, "file:") {
		base := filepath.Base(strings.TrimPrefix(candidate.URL, "file:"))
		if base == "." || base == string(filepath.Separator) || strings.TrimSpace(base) == "" {
			base = name + ".pdf"
		}
		return filepath.Join(projectPath, "documents", "local", base)
	}
	return filepath.Join(projectPath, "documents", "open-access", name+".pdf")
}

func firstNonEmptyDocument(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "document"
}
