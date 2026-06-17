package library

import "testing"

func TestBuildJabRefQualityReportFlagsKeysGroupsFilesCleanupAndApproval(t *testing.T) {
	records := []PaperRecord{
		{Title: "First", Identifiers: Identifiers{DOI: "10.1/a"}, SourceRefs: []SourceRef{{Source: "bibtex", Metadata: map[string]string{"citation_key": "dup", "groups": "included", "saved_searches": "year=2026", "cleanup_diff": "doi normalized", "linked_file_privacy_check": "redacted local path", "normalization_status": "reviewer-approved"}}}},
		{Title: "Second", Identifiers: Identifiers{DOI: "10.1/b"}, SourceRefs: []SourceRef{{Source: "bibtex", Metadata: map[string]string{"citation_key": "dup", "file": "/home/user/private.pdf", "cleanup_diff": "title braces removed"}}}},
		{Title: "Missing", Identifiers: Identifiers{DOI: "10.1/c"}},
	}
	report := BuildJabRefQualityReport(records)
	if report.SchemaVersion != "1" || report.RecordCount != 3 || len(report.CitationKeyCollisions) != 1 || len(report.MissingCitationKeys) != 1 || len(report.GroupsAndSavedSearches) == 0 || len(report.FieldCleanupDiffs) != 2 || len(report.LinkedFilePrivacy) == 0 || len(report.ReviewerApprovedNormalizations) != 1 {
		t.Fatalf("report = %#v", report)
	}
	if !report.HasIssue("citation-key-collision") || !report.HasIssue("linked-file-privacy") || !report.HasIssue("reviewer-approved-normalization") {
		t.Fatalf("issues = %#v", report.Issues)
	}
}
