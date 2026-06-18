package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const nberFixture = `{
  "totalResults": 2,
  "results": [
    {
      "title": "Climate Risk and Investment",
      "url": "/papers/w34276",
      "authors": ["<a href=\"/people/john_doe\">John Doe</a>", "<a href=\"/people/jane_smith\">Jane Smith</a>"],
      "displaydate": "January 2025",
      "abstract": "This paper studies climate risk exposure and firm investment."
    },
    {
      "title": "Labor Market Effects of AI",
      "url": "/papers/w32035",
      "authors": ["<a href=\"/people/alice_wu\">Alice Wu</a>"],
      "displaydate": "October 2024",
      "abstract": ""
    }
  ]
}`

func TestNBERSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/working_page_listing/contentType/working_paper/_/_/search" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(nberFixture))
	}))
	defer server.Close()

	connector := NewNBERConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	if connector.Name() != "nber" {
		t.Fatalf("Name = %q, want nber", connector.Name())
	}
	resp, err := connector.Search(context.Background(), SourceQuery{Terms: "climate", Limit: 25})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(resp.Records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(resp.Records))
	}
	r := resp.Records[0]
	if r.Source != "nber" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "w34276" {
		t.Fatalf("SourceID = %q, want w34276", r.SourceID)
	}
	if r.Identifiers.DOI != "10.3386/w34276" {
		t.Fatalf("DOI = %q, want 10.3386/w34276", r.Identifiers.DOI)
	}
	if r.Year != 2025 {
		t.Fatalf("Year = %d, want 2025", r.Year)
	}
	if r.Title != "Climate Risk and Investment" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Abstract != "This paper studies climate risk exposure and firm investment." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Metadata["authors"] != "John Doe; Jane Smith" {
		t.Fatalf("authors = %q, want 'John Doe; Jane Smith'", r.Metadata["authors"])
	}
	if r.OpenAccess {
		t.Fatal("OpenAccess should be false for NBER papers")
	}
	papers, err := PaperRecords(resp)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if len(papers) != 2 {
		t.Fatalf("papers = %d, want 2", len(papers))
	}
}

func TestNBERSearchStripsHTMLFromAuthors(t *testing.T) {
	fixture := `{"totalResults":1,"results":[{
		"title": "Test Paper",
		"url": "/papers/w99999",
		"authors": ["<a href=\"/people/alice_bob\">Alice Bob</a>", "<b>Charlie Delta</b>"],
		"displaydate": "March 2023",
		"abstract": ""
	}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	resp, err := NewNBERConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if got := resp.Records[0].Metadata["authors"]; got != "Alice Bob; Charlie Delta" {
		t.Fatalf("authors = %q, want 'Alice Bob; Charlie Delta'", got)
	}
}

func TestNBERSearchSkipsResultsWithNoURL(t *testing.T) {
	// A result with an empty URL produces no paper number and no identifier — it must be skipped.
	fixture := `{"totalResults":2,"results":[
		{"title":"No URL Paper","url":"","authors":[],"displaydate":"June 2022","abstract":""},
		{"title":"Valid Paper","url":"/papers/w11111","authors":[],"displaydate":"June 2022","abstract":""}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	resp, err := NewNBERConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("records = %d, want 1 (no-URL result skipped)", len(resp.Records))
	}
	if resp.Records[0].Identifiers.DOI != "10.3386/w11111" {
		t.Fatalf("DOI = %q", resp.Records[0].Identifiers.DOI)
	}
}

func TestNBERSearchDefaultLimit(t *testing.T) {
	var gotPerPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPerPage = r.URL.Query().Get("perPage")
		_, _ = w.Write([]byte(`{"totalResults":0,"results":[]}`))
	}))
	defer server.Close()

	_, _ = NewNBERConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "economics", Limit: 0})
	// Default limit=25 → perPage bumped to 50 (API minimum).
	if gotPerPage != "50" {
		t.Fatalf("perPage = %q, want 50 (bumped from default 25 to API minimum)", gotPerPage)
	}
}

func TestNBERSearchLimitRespectsAPIMinimum(t *testing.T) {
	var gotPerPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPerPage = r.URL.Query().Get("perPage")
		_, _ = w.Write([]byte(`{"totalResults":0,"results":[]}`))
	}))
	defer server.Close()

	_, _ = NewNBERConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "economics", Limit: 10})
	// Requested 10 but API minimum is 50.
	if gotPerPage != "50" {
		t.Fatalf("perPage = %q, want 50 (bumped from 10 to API minimum)", gotPerPage)
	}
}
