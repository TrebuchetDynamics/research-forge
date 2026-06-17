package documents

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestPrivacyLicensingReviewFlagsRequiredRiskClasses(t *testing.T) {
	records := []library.PaperRecord{{Title: "Private import", Identifiers: library.Identifiers{DOI: "10.1000/private"}, SourceRefs: []library.SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"attachment_files": "paper.pdf", "note": "private note", "annotations": "highlight", "linked_file_privacy_check": "redacted-local-paths"}}}}}
	assets := []DocumentAsset{{PaperID: "10.1000/private", OAStatus: "closed", License: "", LocalPath: "/home/alice/private.pdf", LocalOnly: true}}
	review := ReviewPrivacyLicensing(PrivacyLicensingReviewInput{Records: records, Assets: assets, ShareableReport: "Report leaks /home/alice/private.pdf and private note"})
	for _, kind := range []string{"imported_attachment", "imported_note", "imported_annotation", "local_path", "copyrighted_pdf", "shareable_report"} {
		if !reviewHasIssue(review, kind) {
			t.Fatalf("missing %s in %#v", kind, review.Issues)
		}
	}
	if review.Approved || !review.Blocked {
		t.Fatalf("unapproved risk review should be blocked: %#v", review)
	}
}

func TestPrivacyLicensingReviewRequiresExplicitApproval(t *testing.T) {
	input := PrivacyLicensingReviewInput{Records: []library.PaperRecord{{Title: "Private import", Identifiers: library.Identifiers{DOI: "10.1000/private"}, SourceRefs: []library.SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"note": "private"}}}}}}
	review := ReviewPrivacyLicensing(input)
	if err := GuardPrivacyLicensing(review); err == nil {
		t.Fatalf("expected unapproved review to block")
	}
	approved := ApprovePrivacyLicensing(review, "reviewer-a", "redacted and license checked")
	if err := GuardPrivacyLicensing(approved); err != nil {
		t.Fatalf("approved review blocked: %v", err)
	}
	if approved.Reviewer != "reviewer-a" || approved.ApprovalReason == "" {
		t.Fatalf("approval metadata missing: %#v", approved)
	}
}

func reviewHasIssue(review PrivacyLicensingReview, kind string) bool {
	for _, issue := range review.Issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}
