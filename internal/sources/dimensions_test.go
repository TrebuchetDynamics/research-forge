package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDimensionsSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dsl.json" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Fatalf("Content-Type = %q, want text/plain", ct)
		}
		_, _ = w.Write([]byte(`{"publications":[{
			"id":"pub.1234567890",
			"doi":"10.1000/xyz123",
			"title":"Machine Learning for Climate Science",
			"year":2024,
			"journal":{"title":"Nature Climate Change"},
			"abstract":"We apply ML to climate data.",
			"open_access":true
		}]}`))
	}))
	defer server.Close()

	connector := NewDimensionsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "machine learning climate", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if response.RawRef != "dimensions:/api/dsl.json" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "dimensions" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "pub.1234567890" {
		t.Fatalf("SourceID = %q", r.SourceID)
	}
	if r.Identifiers.DOI != "10.1000/xyz123" {
		t.Fatalf("DOI = %q", r.Identifiers.DOI)
	}
	if r.Title != "Machine Learning for Climate Science" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", r.Year)
	}
	if r.Abstract != "We apply ML to climate data." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Venue != "Nature Climate Change" {
		t.Fatalf("Venue = %q", r.Venue)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Machine Learning for Climate Science" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestDimensionsSearchNoDOIFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"publications":[{
			"id":"pub.9999999999",
			"doi":"",
			"title":"No DOI Publication",
			"year":2020,
			"open_access":false
		}]}`))
	}))
	defer server.Close()

	connector := NewDimensionsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	r := response.Records[0]
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "dimensions:pub.9999999999" {
		t.Fatalf("CrossrefID = %q", r.Identifiers.CrossrefID)
	}
}

func TestDimensionsSearchDSLBodyContainsQuery(t *testing.T) {
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		_, _ = w.Write([]byte(`{"publications":[]}`))
	}))
	defer server.Close()

	connector := NewDimensionsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "protein folding", Limit: 10})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if !strings.Contains(capturedBody, "protein folding") {
		t.Fatalf("DSL body does not contain query terms: %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "limit 10") {
		t.Fatalf("DSL body does not contain limit: %q", capturedBody)
	}
}
