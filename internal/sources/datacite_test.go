package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDataCiteSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dois" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "climate data" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("page[size]") != "2" {
			t.Fatalf("page[size] = %q", r.URL.Query().Get("page[size]"))
		}
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "10.5281/zenodo.12345",
					"type": "dois",
					"attributes": {
						"doi": "10.5281/zenodo.12345",
						"titles": [{"title": "Dataset on Climate Change"}],
						"descriptions": [{"description": "Global temperature records.", "descriptionType": "Abstract"}],
						"publicationYear": 2024,
						"publisher": "Zenodo",
						"types": {"resourceTypeGeneral": "Dataset"},
						"rightsList": [
							{"rights": "Creative Commons Attribution 4.0 International", "rightsUri": "https://creativecommons.org/licenses/by/4.0/"}
						],
						"url": "https://zenodo.org/record/12345"
					}
				}
			],
			"meta": {"total": 1}
		}`))
	}))
	defer server.Close()

	connector := NewDataCiteConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "climate data", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if response.RawRef != "datacite:/dois?query=climate data&page[size]=2" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}

	record := response.Records[0]
	if record.Source != "datacite" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "10.5281/zenodo.12345" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.5281/zenodo.12345" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Title != "Dataset on Climate Change" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", record.Year)
	}
	if record.Abstract != "Global temperature records." {
		t.Fatalf("Abstract = %q", record.Abstract)
	}
	if record.Venue != "Dataset" {
		t.Fatalf("Venue = %q", record.Venue)
	}
	if record.Publisher != "Zenodo" {
		t.Fatalf("Publisher = %q", record.Publisher)
	}
	if len(record.URLs) != 1 || record.URLs[0] != "https://zenodo.org/record/12345" {
		t.Fatalf("URLs = %v", record.URLs)
	}
	if record.License != "Creative Commons Attribution 4.0 International" {
		t.Fatalf("License = %q", record.License)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true (CC license URI)")
	}
	if record.Metadata["resource_type"] != "Dataset" {
		t.Fatalf("Metadata[resource_type] = %q", record.Metadata["resource_type"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Dataset on Climate Change" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestDataCiteSearchOpenAccessDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "10.5281/zenodo.99999",
					"type": "dois",
					"attributes": {
						"doi": "10.5281/zenodo.99999",
						"titles": [{"title": "Proprietary Dataset"}],
						"descriptions": [],
						"publicationYear": 2023,
						"publisher": "ACME Corp",
						"types": {"resourceTypeGeneral": "Dataset"},
						"rightsList": [
							{"rights": "All rights reserved", "rightsUri": "https://example.com/proprietary"}
						],
						"url": "https://example.com/data"
					}
				}
			],
			"meta": {"total": 1}
		}`))
	}))
	defer server.Close()

	connector := NewDataCiteConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "proprietary"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	if response.Records[0].OpenAccess {
		t.Fatal("OpenAccess = true, want false for non-CC license")
	}
}

func TestDataCiteSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page[size]") != "25" {
			t.Fatalf("default page[size] = %q, want 25", r.URL.Query().Get("page[size]"))
		}
		_, _ = w.Write([]byte(`{"data":[],"meta":{"total":0}}`))
	}))
	defer server.Close()

	connector := NewDataCiteConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestDataCiteSearchFirstTitleAndAbstract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "10.1234/multi",
					"type": "dois",
					"attributes": {
						"doi": "10.1234/multi",
						"titles": [{"title": ""}, {"title": "Second Title"}],
						"descriptions": [
							{"description": "Methods section.", "descriptionType": "Methods"},
							{"description": "The real abstract.", "descriptionType": "Abstract"}
						],
						"publicationYear": 2022,
						"publisher": "Test",
						"types": {"resourceTypeGeneral": "Software"},
						"rightsList": [],
						"url": ""
					}
				}
			],
			"meta": {"total": 1}
		}`))
	}))
	defer server.Close()

	connector := NewDataCiteConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "multi"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	record := response.Records[0]
	if record.Title != "Second Title" {
		t.Fatalf("Title = %q, want first non-empty", record.Title)
	}
	if record.Abstract != "The real abstract." {
		t.Fatalf("Abstract = %q, want Abstract descriptionType", record.Abstract)
	}
	if record.Venue != "Software" {
		t.Fatalf("Venue = %q", record.Venue)
	}
}
