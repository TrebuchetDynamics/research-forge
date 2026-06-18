package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDOABSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/search" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "open science" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("limit") != "3" {
			t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`[{
			"uuid": "abc123",
			"handle": "20.500.12854/72803",
			"name": "Open Science: the Very Idea",
			"type": "item",
			"metadata": [
				{"key": "dc.title", "value": "Open Science: the Very Idea", "language": null},
				{"key": "dc.contributor.author", "value": "Miedema, Frank", "language": null},
				{"key": "dc.date.issued", "value": "2022", "language": null},
				{"key": "oapen.identifier.doi", "value": "10.1007/978-94-024-2115-6", "language": null},
				{"key": "dc.description.abstract", "value": "This open access book explores open science.", "language": null},
				{"key": "publisher.name", "value": "Springer", "language": null},
				{"key": "publisher.oalicense", "value": "https://creativecommons.org/licenses/by/4.0/", "language": null},
				{"key": "dc.identifier.uri", "value": "https://directory.doabooks.org/handle/20.500.12854/72803", "language": null},
				{"key": "oapen.imprint", "value": "Springer Nature", "language": null}
			]
		}]`))
	}))
	defer server.Close()

	connector := NewDOABConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "open science", Limit: 3})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if response.RawRef != "doab:/rest/search?query=open+science&limit=3" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "doab" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "20.500.12854/72803" {
		t.Fatalf("SourceID = %q", r.SourceID)
	}
	if r.Title != "Open Science: the Very Idea" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Identifiers.DOI != "10.1007/978-94-024-2115-6" {
		t.Fatalf("DOI = %q", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", r.Identifiers.CrossrefID)
	}
	if r.Year != 2022 {
		t.Fatalf("Year = %d, want 2022", r.Year)
	}
	if r.Abstract != "This open access book explores open science." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Publisher != "Springer" {
		t.Fatalf("Publisher = %q", r.Publisher)
	}
	if r.License != "https://creativecommons.org/licenses/by/4.0/" {
		t.Fatalf("License = %q", r.License)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(r.URLs) != 1 || r.URLs[0] != "https://directory.doabooks.org/handle/20.500.12854/72803" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Open Science: the Very Idea" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestDOABSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{
			"uuid": "def456",
			"handle": "20.500.12854/99999",
			"name": "No DOI Book",
			"type": "item",
			"metadata": [
				{"key": "dc.title", "value": "No DOI Book", "language": null},
				{"key": "dc.date.issued", "value": "2020", "language": null}
			]
		}]`))
	}))
	defer server.Close()

	connector := NewDOABConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
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
	if r.Identifiers.CrossrefID != "doab:20.500.12854/99999" {
		t.Fatalf("CrossrefID = %q, want doab:20.500.12854/99999", r.Identifiers.CrossrefID)
	}
}

func TestDOABSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{
				"uuid": "aaa",
				"handle": "20.500.12854/11111",
				"name": "",
				"type": "item",
				"metadata": [
					{"key": "dc.title", "value": "", "language": null}
				]
			},
			{
				"uuid": "bbb",
				"handle": "20.500.12854/22222",
				"name": "A Real Book",
				"type": "item",
				"metadata": [
					{"key": "dc.title", "value": "A Real Book", "language": null}
				]
			}
		]`))
	}))
	defer server.Close()

	connector := NewDOABConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "A Real Book" {
		t.Fatalf("Title = %q", response.Records[0].Title)
	}
}

func TestDOABSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("default limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	connector := NewDOABConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
