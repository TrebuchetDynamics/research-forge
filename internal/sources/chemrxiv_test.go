package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChemRxivSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/engage/chemrxiv/public-api/v1/items" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("term") != "catalysis" {
			t.Fatalf("term = %q", r.URL.Query().Get("term"))
		}
		if r.URL.Query().Get("limit") != "2" {
			t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("skip") != "0" {
			t.Fatalf("skip = %q", r.URL.Query().Get("skip"))
		}
		_, _ = w.Write([]byte(`{
			"itemHits": [{
				"item": {
					"id": "64e83f7d1a44e25c6b000001",
					"doi": "10.26434/chemrxiv-2023-abcde",
					"title": "Catalytic Cycle of a Model Enzyme",
					"abstract": "We report a fixture abstract.",
					"statusDate": "2023-08-24T12:00:00.000Z",
					"authors": [{"firstName": "Jane", "lastName": "Smith"}],
					"categories": [{"id": "cat1", "name": "Inorganic Chemistry"}],
					"license": {
						"id": "CC BY 4.0",
						"name": "Creative Commons Attribution 4.0 International",
						"url": "https://creativecommons.org/licenses/by/4.0/"
					}
				}
			}],
			"totalCount": 1
		}`))
	}))
	defer server.Close()
	connector := NewChemRxivConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "catalysis", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "chemrxiv:/engage/chemrxiv/public-api/v1/items?term=catalysis&limit=2" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	rec := response.Records[0]
	if rec.Source != "chemrxiv" {
		t.Fatalf("Source = %q", rec.Source)
	}
	if rec.SourceID != "64e83f7d1a44e25c6b000001" {
		t.Fatalf("SourceID = %q", rec.SourceID)
	}
	if rec.Identifiers.DOI != "10.26434/chemrxiv-2023-abcde" {
		t.Fatalf("DOI = %q", rec.Identifiers.DOI)
	}
	if rec.Title != "Catalytic Cycle of a Model Enzyme" {
		t.Fatalf("Title = %q", rec.Title)
	}
	if rec.Year != 2023 {
		t.Fatalf("Year = %d, want 2023", rec.Year)
	}
	if rec.Abstract != "We report a fixture abstract." {
		t.Fatalf("Abstract = %q", rec.Abstract)
	}
	if !rec.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if rec.License != "Creative Commons Attribution 4.0 International" {
		t.Fatalf("License = %q", rec.License)
	}
	if rec.Metadata["category"] != "Inorganic Chemistry" {
		t.Fatalf("category = %q", rec.Metadata["category"])
	}
	if rec.Metadata["license_id"] != "CC BY 4.0" {
		t.Fatalf("license_id = %q", rec.Metadata["license_id"])
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 {
		t.Fatalf("PaperRecords count = %d, want 1", len(papers))
	}
}

func TestChemRxivSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"itemHits": [{
				"item": {
					"id": "64e83f7d1a44e25c6b000002",
					"doi": "",
					"title": "Preprint Without DOI",
					"abstract": "A preprint not yet assigned a DOI.",
					"statusDate": "2024-01-10T00:00:00.000Z",
					"authors": [],
					"categories": [],
					"license": {"id": "CC BY 4.0", "name": "CC BY 4.0", "url": ""}
				}
			}],
			"totalCount": 1
		}`))
	}))
	defer server.Close()
	connector := NewChemRxivConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "preprint", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	rec := response.Records[0]
	if rec.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", rec.Identifiers.DOI)
	}
	if rec.Identifiers.CrossrefID != "chemrxiv:64e83f7d1a44e25c6b000002" {
		t.Fatalf("CrossrefID = %q", rec.Identifiers.CrossrefID)
	}
}

func TestChemRxivSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"itemHits": [
				{"item": {"id": "aaa", "doi": "10.1234/aaa", "title": "", "statusDate": "2024-01-01T00:00:00.000Z"}},
				{"item": {"id": "bbb", "doi": "10.1234/bbb", "title": "Valid Title", "statusDate": "2024-01-02T00:00:00.000Z",
					"license": {"id": "CC BY", "name": "CC BY", "url": ""}}}
			],
			"totalCount": 2
		}`))
	}))
	defer server.Close()
	connector := NewChemRxivConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title filtered)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Title" {
		t.Fatalf("Title = %q", response.Records[0].Title)
	}
}

func TestChemRxivSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`{"itemHits":[],"totalCount":0}`))
	}))
	defer server.Close()
	connector := NewChemRxivConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
}
