package library

import "strings"

type JabRefQualityReport struct {
	SchemaVersion                  string                   `json:"schemaVersion"`
	RecordCount                    int                      `json:"recordCount"`
	MissingCitationKeys            []JabRefRecordFinding    `json:"missingCitationKeys,omitempty"`
	CitationKeyCollisions          []JabRefCollisionFinding `json:"citationKeyCollisions,omitempty"`
	GroupsAndSavedSearches         []JabRefRecordFinding    `json:"groupsAndSavedSearches,omitempty"`
	FieldCleanupDiffs              []JabRefRecordFinding    `json:"fieldCleanupDiffs,omitempty"`
	LinkedFilePrivacy              []JabRefRecordFinding    `json:"linkedFilePrivacy,omitempty"`
	ReviewerApprovedNormalizations []JabRefRecordFinding    `json:"reviewerApprovedNormalizations,omitempty"`
	Issues                         []JabRefQualityIssue     `json:"issues"`
}

type JabRefRecordFinding struct {
	RecordIndex int    `json:"recordIndex"`
	Title       string `json:"title"`
	CitationKey string `json:"citationKey,omitempty"`
	Detail      string `json:"detail"`
}

type JabRefCollisionFinding struct {
	CitationKey   string `json:"citationKey"`
	RecordIndexes []int  `json:"recordIndexes"`
}

type JabRefQualityIssue struct{ Kind, Severity, Message string }

func BuildJabRefQualityReport(records []PaperRecord) JabRefQualityReport {
	report := JabRefQualityReport{SchemaVersion: "1", RecordCount: len(records)}
	keys := map[string][]int{}
	for i, record := range records {
		key := jabrefMetadataValue(record, "citation_key")
		finding := func(detail string) JabRefRecordFinding {
			return JabRefRecordFinding{RecordIndex: i, Title: record.Title, CitationKey: key, Detail: detail}
		}
		if key == "" {
			report.MissingCitationKeys = append(report.MissingCitationKeys, finding("missing BibTeX/BibLaTeX citation key"))
			report.Issues = append(report.Issues, JabRefQualityIssue{"missing-citation-key", "warning", record.Title + " has no citation key"})
		} else {
			keys[strings.ToLower(key)] = append(keys[strings.ToLower(key)], i)
		}
		if detail := joinedMetadata(record, "groups", "collections", "saved_searches", "saved_search"); detail != "" {
			report.GroupsAndSavedSearches = append(report.GroupsAndSavedSearches, finding(detail))
		}
		if detail := jabrefMetadataValue(record, "cleanup_diff"); detail != "" {
			report.FieldCleanupDiffs = append(report.FieldCleanupDiffs, finding(detail))
		}
		if detail := joinedMetadata(record, "linked_file_privacy_check", "attachment_files", "file"); detail != "" {
			report.LinkedFilePrivacy = append(report.LinkedFilePrivacy, finding(detail))
			report.Issues = append(report.Issues, JabRefQualityIssue{"linked-file-privacy", "warning", record.Title + " has linked-file privacy context"})
		}
		if strings.Contains(strings.ToLower(jabrefMetadataValue(record, "normalization_status")), "approved") || strings.Contains(strings.ToLower(jabrefMetadataValue(record, "reviewer_approved_normalization")), "true") {
			report.ReviewerApprovedNormalizations = append(report.ReviewerApprovedNormalizations, finding("reviewer-approved normalization"))
			report.Issues = append(report.Issues, JabRefQualityIssue{"reviewer-approved-normalization", "info", record.Title + " normalization was reviewer-approved"})
		}
	}
	for key, indexes := range keys {
		if len(indexes) > 1 {
			report.CitationKeyCollisions = append(report.CitationKeyCollisions, JabRefCollisionFinding{CitationKey: key, RecordIndexes: indexes})
			report.Issues = append(report.Issues, JabRefQualityIssue{"citation-key-collision", "error", key + " appears on multiple records"})
		}
	}
	return report
}

func (r JabRefQualityReport) HasIssue(kind string) bool {
	for _, issue := range r.Issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}

func jabrefMetadataValue(record PaperRecord, key string) string {
	for _, ref := range record.SourceRefs {
		if ref.Metadata != nil && strings.TrimSpace(ref.Metadata[key]) != "" {
			return strings.TrimSpace(ref.Metadata[key])
		}
	}
	return ""
}

func joinedMetadata(record PaperRecord, keys ...string) string {
	parts := []string{}
	for _, key := range keys {
		if value := jabrefMetadataValue(record, key); value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	return strings.Join(parts, "; ")
}
