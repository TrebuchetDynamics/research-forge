package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestZbMATHSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/document/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "elliptic curves" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("per_page") != "2" {
			t.Fatalf("per_page = %q", r.URL.Query().Get("per_page"))
		}
		_, _ = w.Write([]byte(`{
			"result": {
				"hits": [
					{
						"document_id": 7789123,
						"title": "Elliptic Curves and Modular Forms",
						"abstract": "A survey of the theory.",
						"year": 2023,
						"authors": [{"name": "Smith, John"}, {"name": "Doe, Jane"}],
						"journal": {"name": "Annals of Mathematics"},
						"doi": "10.4007/annals.2023.100",
						"zbl_id": "07789123",
						"msc_codes": [{"msc_code": "11G05"}, {"msc_code": "14H52"}]
					}
				],
				"count": 1
			}
		}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "elliptic curves", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	wantRawRef := "zbmath:/v1/document/?q=elliptic+curves&per_page=2"
	if response.RawRef != wantRawRef {
		t.Fatalf("RawRef = %q, want %q", response.RawRef, wantRawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}

	record := response.Records[0]
	if record.Source != "zbmath" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "7789123" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.4007/annals.2023.100" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Title != "Elliptic Curves and Modular Forms" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2023 {
		t.Fatalf("Year = %d, want 2023", record.Year)
	}
	if record.Abstract != "A survey of the theory." {
		t.Fatalf("Abstract = %q", record.Abstract)
	}
	if record.Venue != "Annals of Mathematics" {
		t.Fatalf("Venue = %q", record.Venue)
	}
	if len(record.URLs) != 1 || record.URLs[0] != "https://zbmath.org/?q=an:07789123" {
		t.Fatalf("URLs = %v", record.URLs)
	}
	if record.Metadata["zbl_id"] != "07789123" {
		t.Fatalf("Metadata[zbl_id] = %q", record.Metadata["zbl_id"])
	}
	if record.Metadata["msc_codes"] != "11G05; 14H52" {
		t.Fatalf("Metadata[msc_codes] = %q", record.Metadata["msc_codes"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Elliptic Curves and Modular Forms" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestZbMATHSearchZblIDFallbackIdentifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"result": {
				"hits": [
					{
						"document_id": 1234567,
						"title": "No DOI Paper",
						"abstract": "",
						"year": 2019,
						"authors": [{"name": "Euler, L"}],
						"journal": {"name": "Acta Math"},
						"doi": "",
						"zbl_id": "01234567",
						"msc_codes": []
					}
				],
				"count": 1
			}
		}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "euler"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", record.Identifiers.DOI)
	}
	if record.Identifiers.CrossrefID != "zbmath:01234567" {
		t.Fatalf("CrossrefID = %q, want zbmath:01234567", record.Identifiers.CrossrefID)
	}
}

func TestZbMATHSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "25" {
			t.Fatalf("default per_page = %q, want 25", r.URL.Query().Get("per_page"))
		}
		_, _ = w.Write([]byte(`{"result":{"hits":[],"count":0}}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
