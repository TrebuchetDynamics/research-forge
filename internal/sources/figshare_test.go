package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFigshareSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/articles" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("search_for") != "neural networks" {
			t.Fatalf("search_for = %q", r.URL.Query().Get("search_for"))
		}
		if r.URL.Query().Get("page_size") != "3" {
			t.Fatalf("page_size = %q", r.URL.Query().Get("page_size"))
		}
		_, _ = w.Write([]byte(`[{"id":12345678,"title":"Dataset on neural networks","doi":"10.6084/m9.figshare.12345678.v1","url":"https://api.figshare.com/v2/articles/12345678","url_public_html":"https://figshare.com/articles/dataset/neural_networks/12345678","published_date":"2024-01-15T00:00:00Z","defined_type":3,"defined_type_name":"dataset","description":"We collected data on...","license":{"value":1,"name":"CC BY 4.0","url":"https://creativecommons.org/licenses/by/4.0/"}}]`))
	}))
	defer server.Close()
	connector := NewFigshareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "neural networks", Limit: 3})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "figshare" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "12345678" {
		t.Fatalf("SourceID = %q, want 12345678", record.SourceID)
	}
	if record.Identifiers.DOI != "10.6084/m9.figshare.12345678.v1" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", record.Identifiers.CrossrefID)
	}
	if record.Title != "Dataset on neural networks" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", record.Year)
	}
	if record.License != "CC BY 4.0" {
		t.Fatalf("License = %q, want CC BY 4.0", record.License)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.Venue != "dataset" {
		t.Fatalf("Venue = %q, want dataset", record.Venue)
	}
}

func TestFigshareSearchNoDOIFallsBackToCrossrefID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":99999999,"title":"No DOI Article","doi":"","url":"","url_public_html":"","published_date":"2023-05-01T00:00:00Z","defined_type":1,"defined_type_name":"figure","description":"","license":null}]`))
	}))
	defer server.Close()
	connector := NewFigshareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
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
	if record.Identifiers.CrossrefID != "figshare:99999999" {
		t.Fatalf("CrossrefID = %q, want figshare:99999999", record.Identifiers.CrossrefID)
	}
	if record.OpenAccess {
		t.Fatal("OpenAccess = true, want false (no license)")
	}
	if record.License != "" {
		t.Fatalf("License = %q, want empty", record.License)
	}
}

func TestFigshareSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_size") != "25" {
			t.Fatalf("default page_size = %q, want 25", r.URL.Query().Get("page_size"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	connector := NewFigshareConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
