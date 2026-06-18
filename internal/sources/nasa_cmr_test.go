package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNASACMRSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/collections.json" {
			t.Fatalf("path = %q, want /search/collections.json", r.URL.Path)
		}
		if r.URL.Query().Get("keyword") == "" {
			t.Fatal("missing keyword param")
		}
		_, _ = w.Write([]byte(`{
			"feed": {
				"entry": [{
					"id": "C3291177466-NSIDC_CPRD",
					"title": "MEaSUREs Greenland Ice Velocity V004",
					"summary": "This collection contains ice velocity maps derived from InSAR data.",
					"updated": "2024-01-15T00:00:00.000Z",
					"archive_center": "NASA NSIDC DAAC",
					"organizations": ["NASA NSIDC DAAC", "UWA/APL/PSC"],
					"short_name": "NSIDC-0481",
					"version_id": "4",
					"online_access_flag": true,
					"links": [
						{"rel": "http://esipfed.org/ns/fedsearch/1.1/metadata#", "hreflang": "en-US", "href": "https://doi.org/10.5067/GQZQY2M5507Z"},
						{"rel": "http://esipfed.org/ns/fedsearch/1.1/data#", "hreflang": "en-US", "href": "https://search.earthdata.nasa.gov/search"}
					]
				}]
			}
		}`))
	}))
	defer server.Close()

	connector := NewNASACMRConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "greenland ice", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "nasa-cmr" {
		t.Fatalf("Source = %q, want nasa-cmr", r.Source)
	}
	if r.SourceID != "C3291177466-NSIDC_CPRD" {
		t.Fatalf("SourceID = %q, want C3291177466-NSIDC_CPRD", r.SourceID)
	}
	if r.Identifiers.DOI != "10.5067/gqzqy2m5507z" {
		t.Fatalf("DOI = %q, want 10.5067/gqzqy2m5507z", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "MEaSUREs Greenland Ice Velocity V004" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", r.Year)
	}
	if r.Abstract != "This collection contains ice velocity maps derived from InSAR data." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Publisher != "NASA" {
		t.Fatalf("Publisher = %q, want NASA", r.Publisher)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if r.Metadata["archive_center"] != "NASA NSIDC DAAC" {
		t.Fatalf("archive_center = %q", r.Metadata["archive_center"])
	}
	if r.Metadata["organizations"] != "NASA NSIDC DAAC; UWA/APL/PSC" {
		t.Fatalf("organizations = %q", r.Metadata["organizations"])
	}
	if r.Metadata["short_name"] != "NSIDC-0481" {
		t.Fatalf("short_name = %q", r.Metadata["short_name"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "MEaSUREs Greenland Ice Velocity V004" {
		t.Fatalf("papers round-trip failed")
	}
}

func TestNASACMRSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"feed": {
				"entry": [{
					"id": "C9999999999-TEST_CENTER",
					"title": "Dataset Without DOI",
					"summary": "No DOI available for this collection.",
					"updated": "2020-06-01T00:00:00.000Z",
					"archive_center": "TEST_CENTER",
					"organizations": ["TEST_ORG"],
					"online_access_flag": false,
					"links": [
						{"rel": "http://esipfed.org/ns/fedsearch/1.1/data#", "hreflang": "en-US", "href": "https://example.com/data"}
					]
				}]
			}
		}`))
	}))
	defer server.Close()

	connector := NewNASACMRConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "nasa-cmr:C9999999999-TEST_CENTER" {
		t.Fatalf("CrossrefID = %q, want nasa-cmr:C9999999999-TEST_CENTER", r.Identifiers.CrossrefID)
	}
	if r.OpenAccess {
		t.Fatal("OpenAccess = true, want false for non-online-access collection")
	}
}

func TestNASACMRSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"feed": {
				"entry": [
					{
						"id": "C0000000001-TEST",
						"title": "",
						"summary": "empty title collection",
						"updated": "2021-01-01T00:00:00.000Z",
						"online_access_flag": true,
						"links": []
					},
					{
						"id": "C0000000002-TEST",
						"title": "Valid Collection Title",
						"summary": "A valid collection.",
						"updated": "2022-03-01T00:00:00.000Z",
						"online_access_flag": true,
						"links": [{"rel": "http://esipfed.org/ns/fedsearch/1.1/metadata#", "href": "https://doi.org/10.5067/TESTVALID"}]
					}
				]
			}
		}`))
	}))
	defer server.Close()

	connector := NewNASACMRConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Collection Title" {
		t.Fatalf("Title = %q, want Valid Collection Title", response.Records[0].Title)
	}
}

func TestNASACMRSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_size") != "25" {
			t.Fatalf("default page_size = %q, want 25", r.URL.Query().Get("page_size"))
		}
		_, _ = w.Write([]byte(`{"feed":{"entry":[]}}`))
	}))
	defer server.Close()

	connector := NewNASACMRConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "atmosphere", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
