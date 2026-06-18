package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBioStudiesSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/biostudies/api/v1/search" {
			t.Fatalf("path = %q, want /biostudies/api/v1/search", r.URL.Path)
		}
		if r.URL.Query().Get("query") == "" {
			t.Fatal("missing query param")
		}
		_, _ = w.Write([]byte(`{
			"hits": [{
				"accession": "S-EPMC12345678",
				"type": "study",
				"title": "Machine Learning for Microbiologists",
				"author": "Asnicar F Waldron L Segata N",
				"release_date": "2024-01-15",
				"views": 100,
				"isPublic": true,
				"content": "S-EPMC12345678 PMC12345678 Machine learning for microbiologists abstract text"
			}],
			"totalHits": 1
		}`))
	}))
	defer server.Close()

	connector := NewBioStudiesConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "biostudies" {
		t.Fatalf("Source = %q, want biostudies", r.Source)
	}
	if r.SourceID != "S-EPMC12345678" {
		t.Fatalf("SourceID = %q, want S-EPMC12345678", r.SourceID)
	}
	if r.Identifiers.CrossrefID != "biostudies:S-EPMC12345678" {
		t.Fatalf("CrossrefID = %q, want biostudies:S-EPMC12345678", r.Identifiers.CrossrefID)
	}
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty (BioStudies does not provide DOI)", r.Identifiers.DOI)
	}
	if r.Title != "Machine Learning for Microbiologists" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", r.Year)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	wantURL := "https://www.ebi.ac.uk/biostudies/studies/S-EPMC12345678"
	if len(r.URLs) != 1 || r.URLs[0] != wantURL {
		t.Fatalf("URLs = %v, want [%q]", r.URLs, wantURL)
	}
	if r.Metadata["authors_raw"] != "Asnicar F Waldron L Segata N" {
		t.Fatalf("authors_raw = %q", r.Metadata["authors_raw"])
	}
	if r.Metadata["study_type"] != "study" {
		t.Fatalf("study_type = %q, want study", r.Metadata["study_type"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 {
		t.Fatalf("papers count = %d, want 1", len(papers))
	}
	if papers[0].Title != "Machine Learning for Microbiologists" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestBioStudiesSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"hits": [
				{
					"accession": "S-BSST000001",
					"type": "study",
					"title": "",
					"release_date": "2022-01-01",
					"isPublic": true
				},
				{
					"accession": "S-BSST000002",
					"type": "study",
					"title": "Valid Biology Study",
					"release_date": "2022-06-01",
					"isPublic": true
				}
			],
			"totalHits": 2
		}`))
	}))
	defer server.Close()

	connector := NewBioStudiesConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "biology"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Biology Study" {
		t.Fatalf("Title = %q, want Valid Biology Study", response.Records[0].Title)
	}
}

func TestBioStudiesSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageSize") != "25" {
			t.Fatalf("default pageSize = %q, want 25", r.URL.Query().Get("pageSize"))
		}
		_, _ = w.Write([]byte(`{"hits": [], "totalHits": 0}`))
	}))
	defer server.Close()

	connector := NewBioStudiesConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "genomics", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestBioStudiesSearchPrivateStudyNotOpenAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"hits": [{
				"accession": "S-BSST999999",
				"type": "study",
				"title": "Private Study",
				"release_date": "2021-07-01",
				"isPublic": false
			}],
			"totalHits": 1
		}`))
	}))
	defer server.Close()

	connector := NewBioStudiesConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "private"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	if response.Records[0].OpenAccess {
		t.Fatal("OpenAccess = true, want false for non-public study")
	}
}
