package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestDOAJConnectorSearchNormalizesOpenAccessCandidates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/search/articles/artificial%20photosynthesis" {
			t.Fatalf("path = %s", r.URL.EscapedPath())
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"doaj-1","bibjson":{"title":"DOAJ OA article","year":"2026","journal":{"title":"OA Journal"},"identifier":[{"type":"doi","id":"10.1000/doaj"}],"link":[{"type":"fulltext","url":"https://example.org/doaj.pdf"}],"license":[{"type":"CC-BY"}]}}]}`))
	}))
	defer ts.Close()
	response, err := NewDOAJConnector(NewHTTPClient(HTTPClientOptions{BaseURL: ts.URL})).Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("DOAJ Search: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %#v", response.Records)
	}
	record := response.Records[0]
	if record.Source != "doaj" || record.Identifiers.DOI != "10.1000/doaj" || !record.OpenAccess || record.License != "CC-BY" || record.URLs[0] != "https://example.org/doaj.pdf" || record.Metadata["attribution"] == "" || record.Metadata["rate_limit_policy"] == "" {
		t.Fatalf("record = %#v", record)
	}
}

func TestCOREConnectorSearchNormalizesOpenAccessCandidates(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/search/works" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"core-1","title":"CORE OA article","doi":"10.1000/core","yearPublished":2025,"publisher":"Repository","downloadUrl":"https://example.org/core.pdf","links":[{"url":"https://example.org/landing"}],"license":"CC-BY-4.0"}]}`))
	}))
	defer ts.Close()
	response, err := NewCOREConnector(NewHTTPClient(HTTPClientOptions{BaseURL: ts.URL})).Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("CORE Search: %v", err)
	}
	record := response.Records[0]
	if record.Source != "core" || record.Identifiers.DOI != "10.1000/core" || !record.OpenAccess || record.License != "CC-BY-4.0" || !strings.Contains(strings.Join(record.URLs, ";"), "core.pdf") || record.Metadata["attribution"] == "" || record.Metadata["rate_limit_policy"] == "" {
		t.Fatalf("record = %#v", record)
	}
}

func TestCompareOpenAccessCandidatesCoversRequiredSources(t *testing.T) {
	paper := library.PaperRecord{Title: "Candidate fixture", Identifiers: library.Identifiers{DOI: "10.1000/candidate", ArXivID: "2401.12345", PMID: "123", PMCID: "PMC123"}, URLs: []string{"/tmp/local.pdf"}, License: "CC-BY", OpenAccess: true, SourceRefs: []library.SourceRef{{Source: "unpaywall", Metadata: map[string]string{"pdf_url": "https://example.org/unpaywall.pdf", "oa_status": "gold"}}, {Source: "doaj", RawPayloadRef: "doaj:/api/search/articles/candidate", Metadata: map[string]string{"full_text_url": "https://example.org/doaj.pdf", "license": "CC-BY", "attribution": "DOAJ", "rate_limit_policy": "polite"}}, {Source: "core", RawPayloadRef: "core:/v3/search/works?q=candidate", Metadata: map[string]string{"download_url": "https://example.org/core.pdf", "license": "CC-BY", "attribution": "CORE", "rate_limit_policy": "keyed"}}, {Source: "europepmc", Metadata: map[string]string{"full_text_url": "https://example.org/pmc.pdf", "license": "CC-BY"}}}}
	comparison := CompareOpenAccessCandidates([]library.PaperRecord{paper})
	if len(comparison.Candidates) < 6 {
		t.Fatalf("candidates = %#v", comparison.Candidates)
	}
	for _, source := range []string{"unpaywall", "doaj", "core", "pubmed-europepmc-pmc", "arxiv", "local"} {
		if !hasOACandidate(comparison.Candidates, source) {
			t.Fatalf("missing %s in %#v", source, comparison.Candidates)
		}
	}
	for _, candidate := range comparison.Candidates {
		if candidate.Source == "doaj" || candidate.Source == "core" {
			if candidate.Attribution == "" || candidate.RateLimitPolicy == "" || candidate.APIProvenance == "" || !candidate.ReviewerApprovalRequired {
				t.Fatalf("candidate missing DOAJ/CORE policy provenance: %#v", candidate)
			}
		}
	}
}

func hasOACandidate(candidates []OpenAccessCandidate, source string) bool {
	for _, candidate := range candidates {
		if candidate.Source == source {
			return true
		}
	}
	return false
}
