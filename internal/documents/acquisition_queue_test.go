package documents

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func TestBuildLegalAcquisitionQueueCapturesPolicyFields(t *testing.T) {
	comparison := sources.OpenAccessCandidateComparison{SchemaVersion: "1", Candidates: []sources.OpenAccessCandidate{
		{PaperTitle: "OA paper", DOI: "10.1000/oa", Source: "unpaywall", URL: "https://example.org/paper.pdf", License: "CC-BY", OAStatus: "gold", Provenance: "unpaywall:/v2/10.1000/oa", Attribution: "Unpaywall", RateLimitPolicy: "polite", APIProvenance: "unpaywall:/v2/10.1000/oa", ReviewerApprovalRequired: true},
		{PaperTitle: "Local paper", DOI: "10.1000/local", Source: "local", URL: "/private/paper.pdf", License: "", OAStatus: "local-only", Provenance: "local-import", ReviewerApprovalRequired: true},
	}}
	queue := BuildLegalAcquisitionQueue("/project", comparison)
	if queue.SchemaVersion != "1" || len(queue.Items) != 2 {
		t.Fatalf("queue = %#v", queue)
	}
	open := queue.Items[0]
	if open.OAStatus != "gold" || open.License != "CC-BY" || open.SourceURL != "https://example.org/paper.pdf" || open.ExpectedLocalPath != filepath.Join("/project", "documents", "open-access", "10-1000-oa.pdf") || open.Attribution == "" || open.RateLimitPolicy == "" || open.APIProvenance == "" {
		t.Fatalf("open item = %#v", open)
	}
	if open.Restricted || !open.Shareable || !open.ReviewerApprovalRequired || open.ReviewerApproved {
		t.Fatalf("open policy flags wrong: %#v", open)
	}
	local := queue.Items[1]
	if !local.Restricted || local.Shareable || local.ExpectedLocalPath != filepath.Join("/project", "documents", "local", "paper.pdf") {
		t.Fatalf("local policy flags wrong: %#v", local)
	}
}

func TestAcquisitionApprovalRequiredBeforeDownloadOrArchive(t *testing.T) {
	item := LegalAcquisitionQueueItem{ID: "acq-1", License: "CC-BY", OAStatus: "gold", SourceURL: "https://example.org/paper.pdf", ExpectedLocalPath: "/project/documents/open-access/paper.pdf", Shareable: true, ReviewerApprovalRequired: true}
	if err := GuardAcquisition(item, AcquisitionUseDownload); err == nil {
		t.Fatalf("expected unapproved download to be blocked")
	}
	approved := ApproveAcquisition(item, "reviewer-a", "license checked")
	if err := GuardAcquisition(approved, AcquisitionUseDownload); err != nil {
		t.Fatalf("approved download blocked: %v", err)
	}
	restricted := approved
	restricted.Shareable = false
	restricted.Restricted = true
	if err := GuardAcquisition(restricted, AcquisitionUseArchive); err == nil {
		t.Fatalf("expected restricted archive inclusion to be blocked")
	}
}
