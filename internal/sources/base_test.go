package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBASESearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/BaseHttpSearchInterface.fcgi" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("func") != "PerformSearch" {
			t.Fatalf("func = %q", r.URL.Query().Get("func"))
		}
		if r.URL.Query().Get("query") != "superconductors" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("hits") != "5" {
			t.Fatalf("hits = %q", r.URL.Query().Get("hits"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format = %q", r.URL.Query().Get("format"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":1,"start":0,"docs":[{"dctitle":["Superconductor measurements"],"dcdescription":["A fixture abstract."],"dcidentifier":["oai:server:12345","doi:10.1234/test"],"dcdate":["2024"],"dccontributor":["Smith, John","Doe, Jane"],"dcsource":["Journal of Physics"],"dcpublisher":["Publisher Inc"],"dcrights":["cc-by"],"dcformat":["text/html"],"dctype":["1"],"dclink":"https://example.com/paper"}]}}`))
	}))
	defer server.Close()
	connector := NewBASEConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "superconductors", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "base" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "oai:server:12345" {
		t.Fatalf("SourceID = %q, want oai:server:12345", record.SourceID)
	}
	if record.Identifiers.DOI != "10.1234/test" {
		t.Fatalf("DOI = %q, want 10.1234/test", record.Identifiers.DOI)
	}
	if record.Title != "Superconductor measurements" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", record.Year)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.License != "cc-by" {
		t.Fatalf("License = %q, want cc-by", record.License)
	}
	if record.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", record.Identifiers.CrossrefID)
	}
}

func TestBASESearchNoDOIFallsBackToCrossrefID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"numFound":1,"start":0,"docs":[{"dctitle":["No DOI paper"],"dcidentifier":["oai:server:99999"],"dcdate":["2023"],"dcrights":[],"dclink":""}]}}`))
	}))
	defer server.Close()
	connector := NewBASEConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", record.Identifiers.DOI)
	}
	if record.Identifiers.CrossrefID != "oai:server:99999" {
		t.Fatalf("CrossrefID = %q, want oai:server:99999", record.Identifiers.CrossrefID)
	}
}

func TestBASESearchSkipsEmptyTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"numFound":2,"start":0,"docs":[{"dctitle":[],"dcidentifier":["oai:server:1"]},{"dctitle":["Valid Title"],"dcidentifier":["oai:server:2"],"dcrights":["cc-by"]}]}}`))
	}))
	defer server.Close()
	connector := NewBASEConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (empty-title record skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Title" {
		t.Fatalf("Title = %q, want Valid Title", response.Records[0].Title)
	}
}

func TestBASESearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("hits") != "25" {
			t.Fatalf("default hits = %q, want 25", r.URL.Query().Get("hits"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":0,"start":0,"docs":[]}}`))
	}))
	defer server.Close()
	connector := NewBASEConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
