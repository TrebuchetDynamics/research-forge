package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestZenodoSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/records" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "superconductors" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("size") != "2" {
			t.Fatalf("size = %q", r.URL.Query().Get("size"))
		}
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"id":7654321,"doi":"10.5281/zenodo.7654321","metadata":{"title":"Superconductor measurements dataset","description":"A fixture abstract.","publication_date":"2024-03-01","access_right":"open","license":{"id":"cc-by-4.0"},"resource_type":{"title":"Dataset","type":"dataset"}},"links":{"html":"https://zenodo.org/record/7654321"}}]}}`))
	}))
	defer server.Close()
	connector := NewZenodoConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "superconductors", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "zenodo:/api/records?q=superconductors&size=2&sort=bestmatch" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "zenodo" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "7654321" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.5281/zenodo.7654321" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Title != "Superconductor measurements dataset" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d", record.Year)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.License != "cc-by-4.0" {
		t.Fatalf("License = %q", record.License)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Superconductor measurements dataset" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestZenodoSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("size") != "25" {
			t.Fatalf("default size = %q, want 25", r.URL.Query().Get("size"))
		}
		_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
	}))
	defer server.Close()
	connector := NewZenodoConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
