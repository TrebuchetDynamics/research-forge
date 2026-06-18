package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResearchSquareSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			t.Fatalf("path = %q, want /api/search", r.URL.Path)
		}
		if r.URL.Query().Get("term") == "" {
			t.Fatal("missing term param")
		}
		_, _ = w.Write([]byte(`{
			"result": {
				"data": [{
					"article_identity": "rs-1234567",
					"authors": "Alice Smith, Bob Jones",
					"posted_at": "2024-03-15 12:00:00",
					"doi_version": 1,
					"journal_title": "Nature Portfolio",
					"status": "under-review",
					"title": "A Novel Approach to Testing",
					"article_type": "Research",
					"subject_areas": "Biochemistry",
					"url": "/article/rs-1234567/v1"
				}],
				"total": 1
			}
		}`))
	}))
	defer server.Close()

	connector := NewResearchSquareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "novel approach", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "researchsquare" {
		t.Fatalf("Source = %q, want researchsquare", r.Source)
	}
	if r.SourceID != "rs-1234567" {
		t.Fatalf("SourceID = %q, want rs-1234567", r.SourceID)
	}
	wantDOI := "10.21203/rs.3.rs-1234567/v1"
	if r.Identifiers.DOI != wantDOI {
		t.Fatalf("DOI = %q, want %q", r.Identifiers.DOI, wantDOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "A Novel Approach to Testing" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", r.Year)
	}
	if r.Publisher != "Research Square" {
		t.Fatalf("Publisher = %q, want Research Square", r.Publisher)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	wantURL := "https://www.researchsquare.com/article/rs-1234567/v1"
	if len(r.URLs) != 1 || r.URLs[0] != wantURL {
		t.Fatalf("URLs = %v, want [%q]", r.URLs, wantURL)
	}
	if r.Metadata["authors_raw"] != "Alice Smith, Bob Jones" {
		t.Fatalf("authors_raw = %q", r.Metadata["authors_raw"])
	}
	if r.Metadata["journal_title"] != "Nature Portfolio" {
		t.Fatalf("journal_title = %q", r.Metadata["journal_title"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 {
		t.Fatalf("papers count = %d, want 1", len(papers))
	}
	if papers[0].Title != "A Novel Approach to Testing" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestResearchSquareDOIVersionZeroFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"result": {
				"data": [{
					"article_identity": "rs-9999999",
					"authors": "Test Author",
					"posted_at": "2023-01-01 00:00:00",
					"doi_version": 0,
					"title": "Zero Version Article",
					"url": "/article/rs-9999999/v1"
				}],
				"total": 1
			}
		}`))
	}))
	defer server.Close()

	connector := NewResearchSquareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Identifiers.DOI != "10.21203/rs.3.rs-9999999/v1" {
		t.Fatalf("DOI = %q, want 10.21203/rs.3.rs-9999999/v1 (doi_version 0 treated as 1)", r.Identifiers.DOI)
	}
}

func TestResearchSquareSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"result": {
				"data": [
					{
						"article_identity": "rs-0000001",
						"posted_at": "2024-01-01 00:00:00",
						"doi_version": 1,
						"title": "",
						"url": "/article/rs-0000001/v1"
					},
					{
						"article_identity": "rs-0000002",
						"posted_at": "2024-02-01 00:00:00",
						"doi_version": 1,
						"title": "Valid Preprint Title",
						"url": "/article/rs-0000002/v1"
					}
				],
				"total": 2
			}
		}`))
	}))
	defer server.Close()

	connector := NewResearchSquareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Preprint Title" {
		t.Fatalf("Title = %q, want Valid Preprint Title", response.Records[0].Title)
	}
}

func TestResearchSquareSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("default limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`{"result":{"data":[],"total":0}}`))
	}))
	defer server.Close()

	connector := NewResearchSquareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "quantum", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestResearchSquareSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"result": {
				"data": [{
					"article_identity": "",
					"posted_at": "2024-01-01 00:00:00",
					"doi_version": 1,
					"title": "No Identity Article"
				}],
				"total": 1
			}
		}`))
	}))
	defer server.Close()

	connector := NewResearchSquareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty for missing article_identity", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "researchsquare:" {
		t.Fatalf("CrossrefID = %q, want researchsquare:", r.Identifiers.CrossrefID)
	}
}
