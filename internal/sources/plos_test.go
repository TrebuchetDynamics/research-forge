package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPLOSSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "machine learning" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		_, _ = w.Write([]byte(`{
			"response": {
				"numFound": 1,
				"docs": [{
					"id": "10.1371/journal.pone.0348574",
					"journal": "PLOS One",
					"publication_date": "2026-05-27T00:00:00Z",
					"article_type": "Research Article",
					"author": ["Smith J", "Jones K"],
					"abstract": ["Abstract text here"],
					"title": "Paper Title Here"
				}]
			}
		}`))
	}))
	defer server.Close()
	connector := NewPLOSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "plos" {
		t.Fatalf("Source = %q, want plos", record.Source)
	}
	if record.SourceID != "10.1371/journal.pone.0348574" {
		t.Fatalf("SourceID = %q, want 10.1371/journal.pone.0348574", record.SourceID)
	}
	wantDOI := "10.1371/journal.pone.0348574"
	if record.Identifiers.DOI != wantDOI {
		t.Fatalf("Identifiers.DOI = %q, want %q", record.Identifiers.DOI, wantDOI)
	}
	if record.Title != "Paper Title Here" {
		t.Fatalf("Title = %q, want Paper Title Here", record.Title)
	}
	if record.Year != 2026 {
		t.Fatalf("Year = %d, want 2026", record.Year)
	}
	if record.Abstract != "Abstract text here" {
		t.Fatalf("Abstract = %q, want Abstract text here", record.Abstract)
	}
	if record.Venue != "PLOS One" {
		t.Fatalf("Venue = %q, want PLOS One", record.Venue)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	wantURL := "https://doi.org/10.1371/journal.pone.0348574"
	if len(record.URLs) != 1 || record.URLs[0] != wantURL {
		t.Fatalf("URLs = %v, want [%s]", record.URLs, wantURL)
	}
	if record.Metadata["article_type"] != "Research Article" {
		t.Fatalf("Metadata[article_type] = %q, want Research Article", record.Metadata["article_type"])
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Paper Title Here" {
		t.Fatalf("papers[0].Title = %q, want Paper Title Here", papers[0].Title)
	}
}

func TestPLOSSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response": {
				"numFound": 2,
				"docs": [
					{
						"id": "10.1371/journal.pone.0000001",
						"title": "",
						"publication_date": "2024-01-01T00:00:00Z",
						"abstract": []
					},
					{
						"id": "10.1371/journal.pone.0000002",
						"title": "Valid PLOS Title",
						"journal": "PLOS Biology",
						"publication_date": "2024-06-01T00:00:00Z",
						"abstract": ["Some abstract."],
						"article_type": "Research Article"
					}
				]
			}
		}`))
	}))
	defer server.Close()
	connector := NewPLOSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "biology"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank titles should be skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid PLOS Title" {
		t.Fatalf("Title = %q, want Valid PLOS Title", response.Records[0].Title)
	}
}

func TestPLOSSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("rows") != "25" {
			t.Fatalf("default rows = %q, want 25", r.URL.Query().Get("rows"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":0,"docs":[]}}`))
	}))
	defer server.Close()
	connector := NewPLOSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestPLOSSearchAbstractFromArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response": {
				"numFound": 1,
				"docs": [{
					"id": "10.1371/journal.pbio.0000042",
					"title": "Multi-Abstract Paper",
					"journal": "PLOS Biology",
					"publication_date": "2023-03-15T00:00:00Z",
					"abstract": ["First abstract", "second"],
					"article_type": "Review"
				}]
			}
		}`))
	}))
	defer server.Close()
	connector := NewPLOSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "review", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	if response.Records[0].Abstract != "First abstract" {
		t.Fatalf("Abstract = %q, want First abstract", response.Records[0].Abstract)
	}
}
