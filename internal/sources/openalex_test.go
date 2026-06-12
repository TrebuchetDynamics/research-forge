package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAlexConnectorSupportsCursorPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "abc123" {
			t.Fatalf("cursor = %q", r.URL.Query().Get("cursor"))
		}
		if r.URL.Query().Get("per-page") != "1" {
			t.Fatalf("per-page = %q", r.URL.Query().Get("per-page"))
		}
		_, _ = w.Write([]byte(`{"meta":{"next_cursor":"def456"},"results":[{"id":"https://openalex.org/W456","doi":"https://doi.org/10.1000/page","title":"Paged artificial photosynthesis work","publication_year":2026}]}`))
	}))
	defer server.Close()

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1, PageCursor: "abc123"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.NextPageCursor != "def456" {
		t.Fatalf("NextPageCursor = %q", response.NextPageCursor)
	}
	if response.Records[0].SourceID != "W456" {
		t.Fatalf("record = %#v", response.Records[0])
	}
}

func TestOpenAlexConnectorRetriesRateLimitWithBackoff(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "3")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W789","doi":"https://doi.org/10.1000/rate","title":"Rate limited artificial photosynthesis work","publication_year":2026}]}`))
	}))
	defer server.Close()
	var slept []time.Duration

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second, MaxRetries: 1, Sleep: func(duration time.Duration) {
		slept = append(slept, duration)
	}}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if requests != 2 || len(slept) != 1 || slept[0] != 3*time.Second {
		t.Fatalf("requests=%d slept=%#v", requests, slept)
	}
	if response.Records[0].SourceID != "W789" {
		t.Fatalf("record = %#v", response.Records[0])
	}
}

func TestOpenAlexConnectorSearchesAndNormalizesWorks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q, want /works", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "artificial photosynthesis" {
			t.Fatalf("search = %q", r.URL.Query().Get("search"))
		}
		if r.URL.Query().Get("per-page") != "2" {
			t.Fatalf("per-page = %q", r.URL.Query().Get("per-page"))
		}
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "https://openalex.org/W123",
				"doi": "https://doi.org/10.1000/example",
				"title": "Artificial photosynthesis catalyst review",
				"publication_year": 2026,
				"type": "review",
				"open_access": {"is_oa": true, "oa_status": "gold"},
				"primary_location": {"landing_page_url": "https://example.org/paper", "license": "cc-by"}
			}]
		}`))
	}))
	defer server.Close()

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if connector.Name() != "openalex" {
		t.Fatalf("Name = %q", connector.Name())
	}
	if response.RawRef != "openalex:/works?per-page=2&search=artificial+photosynthesis" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("len(records) = %d", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "openalex" || record.SourceID != "W123" || record.Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("record = %#v", record)
	}
	if record.Identifiers.DOI != "10.1000/example" || record.Identifiers.OpenAlexID != "W123" || record.Year != 2026 {
		t.Fatalf("record identifiers/year = %#v", record)
	}
	if record.Metadata["type"] != "review" || record.Metadata["oa_status"] != "gold" {
		t.Fatalf("metadata = %#v", record.Metadata)
	}
	if record.OpenAccess != true || record.License != "cc-by" || len(record.URLs) != 1 || record.URLs[0] != "https://example.org/paper" {
		t.Fatalf("access metadata = %#v", record)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("papers = %#v", papers)
	}
	if papers[0].Identifiers.DOI != "10.1000/example" || papers[0].Identifiers.OpenAlexID != "W123" {
		t.Fatalf("paper identifiers = %#v", papers[0].Identifiers)
	}
	if !papers[0].OpenAccess || papers[0].License != "cc-by" || papers[0].URLs[0] != "https://example.org/paper" {
		t.Fatalf("paper source metadata = %#v", papers[0])
	}
	if papers[0].SourceRefs[0].Metadata["oa_status"] != "gold" || papers[0].SourceRefs[0].Metadata["type"] != "review" {
		t.Fatalf("paper source ref metadata = %#v", papers[0].SourceRefs[0].Metadata)
	}
}
