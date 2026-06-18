package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNTRSSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/citations/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("keyword") != "solar energy" {
			t.Fatalf("keyword = %q", r.URL.Query().Get("keyword"))
		}
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": 20240013836,
				"title": "Solar Energy Research at NASA",
				"abstract": "We report on solar energy advances.",
				"submittedDate": "2024-11-01T11:08:13.6818070+00:00",
				"distributionDate": "2024-11-06T05:00:00.0000000+00:00",
				"keywords": ["solar energy", "photovoltaics"],
				"center": {"code": "LaRC", "name": "Langley Research Center"},
				"stiType": "PRESENTATION"
			}]
		}`))
	}))
	defer server.Close()
	connector := NewNTRSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "solar energy", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "ntrs" {
		t.Fatalf("Source = %q, want ntrs", record.Source)
	}
	if record.SourceID != "20240013836" {
		t.Fatalf("SourceID = %q, want 20240013836", record.SourceID)
	}
	if record.Identifiers.CrossrefID != "ntrs:20240013836" {
		t.Fatalf("CrossrefID = %q, want ntrs:20240013836", record.Identifiers.CrossrefID)
	}
	if record.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", record.Identifiers.DOI)
	}
	if record.Title != "Solar Energy Research at NASA" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", record.Year)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.Venue != "Langley Research Center" {
		t.Fatalf("Venue = %q, want Langley Research Center", record.Venue)
	}
	if record.Metadata["stiType"] != "PRESENTATION" {
		t.Fatalf("Metadata[stiType] = %q, want PRESENTATION", record.Metadata["stiType"])
	}
	if record.Metadata["center_code"] != "LaRC" {
		t.Fatalf("Metadata[center_code] = %q, want LaRC", record.Metadata["center_code"])
	}
	wantURL := "https://ntrs.nasa.gov/citations/20240013836"
	if len(record.URLs) != 1 || record.URLs[0] != wantURL {
		t.Fatalf("URLs = %v, want [%s]", record.URLs, wantURL)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Solar Energy Research at NASA" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestNTRSSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": 1, "title": "", "distributionDate": "2024-01-01T00:00:00+00:00"},
				{"id": 2, "title": "   ", "distributionDate": "2024-01-01T00:00:00+00:00"},
				{"id": 3, "title": "Valid Title", "distributionDate": "2024-01-01T00:00:00+00:00",
				 "center": {"code": "HQ", "name": "NASA Headquarters"}}
			]
		}`))
	}))
	defer server.Close()
	connector := NewNTRSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank titles should be skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Title" {
		t.Fatalf("Title = %q, want Valid Title", response.Records[0].Title)
	}
}

func TestNTRSSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("count") != "25" {
			t.Fatalf("default count = %q, want 25", r.URL.Query().Get("count"))
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	connector := NewNTRSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestNTRSSearchYearFromDistributionDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": 20230001234,
				"title": "Mars Terrain Analysis",
				"abstract": "This paper analyzes Mars terrain data collected by rovers.",
				"submittedDate": "2022-03-15T00:00:00+00:00",
				"distributionDate": "2023-06-20T00:00:00+00:00",
				"center": {"code": "JSC", "name": "Johnson Space Center"},
				"stiType": "TECHNICAL_REPORT"
			}]
		}`))
	}))
	defer server.Close()
	connector := NewNTRSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "mars terrain", Limit: 3})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Year != 2023 {
		t.Fatalf("Year = %d, want 2023 (from distributionDate)", record.Year)
	}
	if record.Abstract == "" {
		t.Fatal("Abstract is empty, want populated")
	}
	if record.Abstract != "This paper analyzes Mars terrain data collected by rovers." {
		t.Fatalf("Abstract = %q", record.Abstract)
	}
}
