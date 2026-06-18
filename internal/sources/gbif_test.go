package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGBIFSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/literature/search" {
			t.Fatalf("path = %q, want /v1/literature/search", r.URL.Path)
		}
		if r.URL.Query().Get("q") == "" {
			t.Fatal("missing q param")
		}
		_, _ = w.Write([]byte(`{
			"offset": 0,
			"limit": 5,
			"endOfRecords": false,
			"count": 9534,
			"results": [{
				"id": "bc860204-52b8-35a8-9f4f-7c9d55043864",
				"title": "Biodiversity Patterns in Alpine Ecosystems",
				"authors": [
					{"firstName": "Alice", "lastName": "Smith"},
					{"firstName": "Bob", "lastName": "Jones"}
				],
				"identifiers": {"doi": "10.1234/test.bio.2022"},
				"year": 2022,
				"abstract": "This paper examines biodiversity patterns.",
				"source": "Journal of Ecology",
				"literatureType": "JOURNAL_ARTICLE",
				"openAccess": true,
				"peerReview": true,
				"websites": ["https://doi.org/10.1234/test.bio.2022"]
			}]
		}`))
	}))
	defer server.Close()

	connector := NewGBIFConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "biodiversity alpine", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "gbif" {
		t.Fatalf("Source = %q, want gbif", r.Source)
	}
	if r.SourceID != "bc860204-52b8-35a8-9f4f-7c9d55043864" {
		t.Fatalf("SourceID = %q", r.SourceID)
	}
	if r.Identifiers.DOI != "10.1234/test.bio.2022" {
		t.Fatalf("DOI = %q, want 10.1234/test.bio.2022", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "Biodiversity Patterns in Alpine Ecosystems" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2022 {
		t.Fatalf("Year = %d, want 2022", r.Year)
	}
	if r.Abstract != "This paper examines biodiversity patterns." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Venue != "Journal of Ecology" {
		t.Fatalf("Venue = %q, want Journal of Ecology", r.Venue)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if r.Metadata["authors"] != "Smith, Alice; Jones, Bob" {
		t.Fatalf("authors = %q, want Smith, Alice; Jones, Bob", r.Metadata["authors"])
	}
	if r.Metadata["literature_type"] != "JOURNAL_ARTICLE" {
		t.Fatalf("literature_type = %q", r.Metadata["literature_type"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "Biodiversity Patterns in Alpine Ecosystems" {
		t.Fatalf("papers round-trip failed")
	}
}

func TestGBIFSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"offset": 0, "limit": 5, "count": 1,
			"results": [{
				"id": "aaaabbbb-cccc-dddd-eeee-ffffgggghhhh",
				"title": "No DOI Article",
				"authors": [],
				"identifiers": {},
				"year": 2020,
				"abstract": "Abstract here.",
				"source": "Conference Proceedings",
				"literatureType": "CONFERENCE_PROCEEDINGS",
				"openAccess": false,
				"peerReview": false,
				"websites": []
			}]
		}`))
	}))
	defer server.Close()

	connector := NewGBIFConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
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
	if r.Identifiers.CrossrefID != "gbif:aaaabbbb-cccc-dddd-eeee-ffffgggghhhh" {
		t.Fatalf("CrossrefID = %q", r.Identifiers.CrossrefID)
	}
	if r.OpenAccess {
		t.Fatal("OpenAccess = true, want false")
	}
}

func TestGBIFSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"offset": 0, "limit": 5, "count": 2,
			"results": [
				{"id": "id-1", "title": "", "authors": [], "identifiers": {"doi": "10.1111/blank"}, "year": 2021, "openAccess": false, "peerReview": false, "websites": []},
				{"id": "id-2", "title": "Valid Biodiversity Paper", "authors": [], "identifiers": {"doi": "10.1111/valid"}, "year": 2022, "openAccess": true, "peerReview": false, "websites": []}
			]
		}`))
	}))
	defer server.Close()

	connector := NewGBIFConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Biodiversity Paper" {
		t.Fatalf("Title = %q, want Valid Biodiversity Paper", response.Records[0].Title)
	}
}

func TestGBIFSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("default limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`{"offset":0,"limit":25,"count":0,"results":[]}`))
	}))
	defer server.Close()

	connector := NewGBIFConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "ecology", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
