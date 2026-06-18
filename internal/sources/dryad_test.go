package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDryadSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/search" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"count": 1,
			"total": 1,
			"_embedded": {
				"stash:datasets": [{
					"identifier": "doi:10.5061/dryad.test123",
					"id": 99999,
					"title": "Climate Data for Testing",
					"authors": [
						{"firstName": "Jane", "lastName": "Smith", "orcid": "0000-0001-2345-6789", "affiliation": "Test University"}
					],
					"abstract": "<p>This is the abstract.</p>",
					"fieldOfScience": "Ecology, Evolution, Behavior and Systematics",
					"publicationDate": "2022-03-15",
					"sharingLink": "http://datadryad.org/dataset/doi:10.5061/dryad.test123",
					"license": "https://spdx.org/licenses/CC0-1.0.html"
				}]
			}
		}`))
	}))
	defer server.Close()

	connector := NewDryadConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "climate", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "dryad" {
		t.Fatalf("Source = %q, want dryad", r.Source)
	}
	if r.SourceID != "doi:10.5061/dryad.test123" {
		t.Fatalf("SourceID = %q, want doi:10.5061/dryad.test123", r.SourceID)
	}
	if r.Identifiers.DOI != "10.5061/dryad.test123" {
		t.Fatalf("DOI = %q, want 10.5061/dryad.test123", r.Identifiers.DOI)
	}
	if r.Title != "Climate Data for Testing" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2022 {
		t.Fatalf("Year = %d, want 2022", r.Year)
	}
	if r.Abstract != "This is the abstract." {
		t.Fatalf("Abstract = %q, want stripped HTML", r.Abstract)
	}
	if r.Publisher != "Dryad" {
		t.Fatalf("Publisher = %q, want Dryad", r.Publisher)
	}
	if r.License != "https://spdx.org/licenses/CC0-1.0.html" {
		t.Fatalf("License = %q", r.License)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(r.URLs) != 1 || r.URLs[0] != "http://datadryad.org/dataset/doi:10.5061/dryad.test123" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	if r.Metadata["field_of_science"] != "Ecology, Evolution, Behavior and Systematics" {
		t.Fatalf("field_of_science = %q", r.Metadata["field_of_science"])
	}

	// Round-trip via PaperRecords.
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 {
		t.Fatalf("papers count = %d, want 1", len(papers))
	}
	if papers[0].Title != "Climate Data for Testing" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestDryadSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"count": 2,
			"total": 2,
			"_embedded": {
				"stash:datasets": [
					{
						"identifier": "doi:10.5061/dryad.blank1",
						"id": 1,
						"title": "",
						"authors": [],
						"abstract": "",
						"publicationDate": "2020-01-01",
						"sharingLink": "http://datadryad.org/dataset/doi:10.5061/dryad.blank1",
						"license": "https://spdx.org/licenses/CC0-1.0.html"
					},
					{
						"identifier": "doi:10.5061/dryad.valid1",
						"id": 2,
						"title": "Valid Dataset Title",
						"authors": [],
						"abstract": "",
						"publicationDate": "2021-05-10",
						"sharingLink": "http://datadryad.org/dataset/doi:10.5061/dryad.valid1",
						"license": "https://spdx.org/licenses/CC0-1.0.html"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	connector := NewDryadConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Dataset Title" {
		t.Fatalf("Title = %q, want Valid Dataset Title", response.Records[0].Title)
	}
}

func TestDryadSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "25" {
			t.Fatalf("default per_page = %q, want 25", r.URL.Query().Get("per_page"))
		}
		_, _ = w.Write([]byte(`{"count":0,"total":0,"_embedded":{"stash:datasets":[]}}`))
	}))
	defer server.Close()

	connector := NewDryadConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "ecology", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestDryadSearchHTMLAbstractStripped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"count": 1,
			"total": 1,
			"_embedded": {
				"stash:datasets": [{
					"identifier": "doi:10.5061/dryad.html1",
					"id": 3,
					"title": "HTML Abstract Test",
					"authors": [],
					"abstract": "<p style=\"margin-bottom:11px;\">Some text.</p><p>More text.</p>",
					"publicationDate": "2023-07-01",
					"sharingLink": "http://datadryad.org/dataset/doi:10.5061/dryad.html1",
					"license": "https://spdx.org/licenses/CC0-1.0.html"
				}]
			}
		}`))
	}))
	defer server.Close()

	connector := NewDryadConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "html", Limit: 1})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	want := "Some text. More text."
	if response.Records[0].Abstract != want {
		t.Fatalf("Abstract = %q, want %q", response.Records[0].Abstract, want)
	}
}
