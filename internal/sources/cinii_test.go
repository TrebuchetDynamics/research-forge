package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCiNiiSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/opensearch/articles" {
			t.Fatalf("path = %q, want /opensearch/articles", r.URL.Path)
		}
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format = %q, want json", r.URL.Query().Get("format"))
		}
		_, _ = w.Write([]byte(`{
			"opensearch:totalResults": 1,
			"opensearch:startIndex": 1,
			"opensearch:itemsPerPage": 2,
			"items": [{
				"@id": "https://cir.nii.ac.jp/crid/1360298762025368704",
				"@type": "item",
				"title": "Machine Learning Basics",
				"dc:creator": ["Alice Test", "Bob Sample"],
				"dc:publisher": "Test Publisher",
				"dc:type": "Article",
				"prism:publicationName": "Journal of Test Studies",
				"prism:issn": "0000-0000",
				"prism:volume": "10",
				"prism:number": "2",
				"prism:startingPage": "100",
				"prism:endingPage": "110",
				"prism:publicationDate": "2021-06-01",
				"description": "<jats:p>This is the abstract.</jats:p>",
				"dc:identifier": [
					{"@type": "cir:DOI", "@value": "10.9999/test.doi"},
					{"@type": "cir:URI", "@value": "https://example.com/paper"}
				]
			}]
		}`))
	}))
	defer server.Close()

	connector := NewCiNiiConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "machine learning", Limit: 2})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "cinii" {
		t.Fatalf("Source = %q, want cinii", r.Source)
	}
	if r.SourceID != "https://cir.nii.ac.jp/crid/1360298762025368704" {
		t.Fatalf("SourceID = %q", r.SourceID)
	}
	if r.Identifiers.DOI != "10.9999/test.doi" {
		t.Fatalf("DOI = %q, want 10.9999/test.doi", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "Machine Learning Basics" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2021 {
		t.Fatalf("Year = %d, want 2021", r.Year)
	}
	if r.Abstract != "This is the abstract." {
		t.Fatalf("Abstract = %q, want stripped JATS", r.Abstract)
	}
	if r.Venue != "Journal of Test Studies" {
		t.Fatalf("Venue = %q, want Journal of Test Studies", r.Venue)
	}
	if r.Publisher != "Test Publisher" {
		t.Fatalf("Publisher = %q, want Test Publisher", r.Publisher)
	}
	if r.Metadata["authors"] != "Alice Test; Bob Sample" {
		t.Fatalf("authors = %q, want Alice Test; Bob Sample", r.Metadata["authors"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 {
		t.Fatalf("papers count = %d, want 1", len(papers))
	}
	if papers[0].Title != "Machine Learning Basics" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestCiNiiSearchNAIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"opensearch:totalResults": 1,
			"items": [{
				"@id": "https://cir.nii.ac.jp/crid/1572261550686743936",
				"@type": "item",
				"title": "Scikit-learn Article",
				"dc:creator": ["PEDREGOSA F."],
				"prism:publicationName": "J. Machine Learn. Res.",
				"prism:publicationDate": "2011",
				"dc:identifier": [
					{"@type": "cir:NAID", "@value": "10030337906"}
				]
			}]
		}`))
	}))
	defer server.Close()

	connector := NewCiNiiConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "scikit-learn"})
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
	if r.Identifiers.CrossrefID != "cinii-naid:10030337906" {
		t.Fatalf("CrossrefID = %q, want cinii-naid:10030337906", r.Identifiers.CrossrefID)
	}
	if r.Year != 2011 {
		t.Fatalf("Year = %d, want 2011", r.Year)
	}
}

func TestCiNiiSearchCRIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"opensearch:totalResults": 1,
			"items": [{
				"@id": "https://cir.nii.ac.jp/crid/9876543210",
				"@type": "item",
				"title": "No Identifier Article",
				"prism:publicationDate": "2020",
				"dc:identifier": []
			}]
		}`))
	}))
	defer server.Close()

	connector := NewCiNiiConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Identifiers.CrossrefID != "cinii:9876543210" {
		t.Fatalf("CrossrefID = %q, want cinii:9876543210", r.Identifiers.CrossrefID)
	}
}

func TestCiNiiSearchSingleStringCreator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"opensearch:totalResults": 1,
			"items": [{
				"@id": "https://cir.nii.ac.jp/crid/1111111111",
				"@type": "item",
				"title": "Single Author Paper",
				"dc:creator": "Solo Researcher",
				"prism:publicationDate": "2019",
				"dc:identifier": [{"@type": "cir:DOI", "@value": "10.1234/test.solo"}]
			}]
		}`))
	}))
	defer server.Close()

	connector := NewCiNiiConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "solo"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Metadata["authors"] != "Solo Researcher" {
		t.Fatalf("authors = %q, want Solo Researcher", r.Metadata["authors"])
	}
}

func TestCiNiiSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"opensearch:totalResults": 2,
			"items": [
				{
					"@id": "https://cir.nii.ac.jp/crid/1111111111",
					"title": "",
					"dc:identifier": [{"@type": "cir:DOI", "@value": "10.9999/blank"}]
				},
				{
					"@id": "https://cir.nii.ac.jp/crid/2222222222",
					"title": "Valid Article Title",
					"prism:publicationDate": "2022",
					"dc:identifier": [{"@type": "cir:DOI", "@value": "10.9999/valid"}]
				}
			]
		}`))
	}))
	defer server.Close()

	connector := NewCiNiiConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Article Title" {
		t.Fatalf("Title = %q, want Valid Article Title", response.Records[0].Title)
	}
}

func TestCiNiiSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("count") != "25" {
			t.Fatalf("default count = %q, want 25", r.URL.Query().Get("count"))
		}
		_, _ = w.Write([]byte(`{"opensearch:totalResults": 0, "items": []}`))
	}))
	defer server.Close()

	connector := NewCiNiiConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "biology", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
