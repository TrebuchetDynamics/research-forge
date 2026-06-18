package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOSTISearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/records" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{
				"osti_id": 3371691,
				"title": "Recent advances in plasma control",
				"doi": "10.1088/1741-4326/ae71eb",
				"publication_date": "2026-05-22T00:00:00Z",
				"journal_name": "Nuclear Fusion",
				"product_type": "Journal Article",
				"authors": [
					{"name": "Tanaka, Kenji", "affiliation_name": "National Institute for Fusion Science", "orcid": "0000000216063204"}
				],
				"subjects": "Nuclear physics"
			}
		]`))
	}))
	defer server.Close()

	connector := NewOSTIConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "plasma control", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]

	if record.Source != "osti" {
		t.Fatalf("Source = %q, want osti", record.Source)
	}
	if record.SourceID != "3371691" {
		t.Fatalf("SourceID = %q, want 3371691", record.SourceID)
	}
	if record.Identifiers.DOI != "10.1088/1741-4326/ae71eb" {
		t.Fatalf("DOI = %q, want 10.1088/1741-4326/ae71eb", record.Identifiers.DOI)
	}
	if record.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", record.Identifiers.CrossrefID)
	}
	if record.Title != "Recent advances in plasma control" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2026 {
		t.Fatalf("Year = %d, want 2026", record.Year)
	}
	if record.Venue != "Nuclear Fusion" {
		t.Fatalf("Venue = %q, want Nuclear Fusion", record.Venue)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.Metadata["product_type"] != "Journal Article" {
		t.Fatalf("Metadata[product_type] = %q, want Journal Article", record.Metadata["product_type"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 {
		t.Fatalf("PaperRecords len = %d, want 1", len(papers))
	}
	if papers[0].Title != "Recent advances in plasma control" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestOSTISearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{
				"osti_id": 9999999,
				"title": "Technical Report Without DOI",
				"doi": "",
				"publication_date": "2025-01-15T00:00:00Z",
				"journal_name": "",
				"product_type": "Technical Report",
				"authors": [],
				"subjects": ""
			}
		]`))
	}))
	defer server.Close()

	connector := NewOSTIConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "technical report", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]

	if record.Identifiers.CrossrefID != "osti:9999999" {
		t.Fatalf("CrossrefID = %q, want osti:9999999", record.Identifiers.CrossrefID)
	}
	if record.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", record.Identifiers.DOI)
	}
}

func TestOSTISearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{
				"osti_id": 1111111,
				"title": "",
				"doi": "",
				"publication_date": "2024-01-01T00:00:00Z",
				"product_type": "Journal Article",
				"authors": [],
				"subjects": ""
			},
			{
				"osti_id": 2222222,
				"title": "Valid Research Paper",
				"doi": "10.1000/xyz123",
				"publication_date": "2024-06-01T00:00:00Z",
				"journal_name": "Science",
				"product_type": "Journal Article",
				"authors": [],
				"subjects": ""
			}
		]`))
	}))
	defer server.Close()

	connector := NewOSTIConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "research", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title should be skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Research Paper" {
		t.Fatalf("Title = %q, want Valid Research Paper", response.Records[0].Title)
	}
}

func TestOSTISearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("size") != "25" {
			t.Fatalf("default size = %q, want 25", r.URL.Query().Get("size"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	connector := NewOSTIConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
