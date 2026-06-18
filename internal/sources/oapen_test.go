package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOAPenSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/search" {
			t.Fatalf("path = %q, want /rest/search", r.URL.Path)
		}
		if r.URL.Query().Get("query") == "" {
			t.Fatal("missing query param")
		}
		if r.URL.Query().Get("expand") != "metadata" {
			t.Fatalf("expand = %q, want metadata", r.URL.Query().Get("expand"))
		}
		_, _ = w.Write([]byte(`[{
			"uuid": "aaaa1111-2222-3333-4444-555566667777",
			"name": "Communicating Climate Change",
			"handle": "20.500.12657/62154",
			"type": "item",
			"metadata": [
				{"key": "dc.title", "value": "Communicating Climate Change"},
				{"key": "dc.title.alternative", "value": "A Guide for Educators"},
				{"key": "dc.contributor.author", "value": "Armstrong, Anne K."},
				{"key": "dc.contributor.author", "value": "Krasny, Marianne E."},
				{"key": "dc.date.issued", "value": "2018"},
				{"key": "dc.description.abstract", "value": "Environmental educators face a formidable challenge."},
				{"key": "dc.language", "value": "English"},
				{"key": "oapen.identifier.doi", "value": "10.7298/cnbq-an02"},
				{"key": "publisher.name", "value": "Cornell University Press"},
				{"key": "publisher.website", "value": "https://www.cornellpress.cornell.edu/"}
			]
		}]`))
	}))
	defer server.Close()

	connector := NewOAPenConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "climate change education", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "oapen" {
		t.Fatalf("Source = %q, want oapen", r.Source)
	}
	if r.SourceID != "20.500.12657/62154" {
		t.Fatalf("SourceID = %q, want 20.500.12657/62154", r.SourceID)
	}
	if r.Identifiers.DOI != "10.7298/cnbq-an02" {
		t.Fatalf("DOI = %q, want 10.7298/cnbq-an02", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "Communicating Climate Change" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2018 {
		t.Fatalf("Year = %d, want 2018", r.Year)
	}
	if r.Abstract != "Environmental educators face a formidable challenge." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Publisher != "Cornell University Press" {
		t.Fatalf("Publisher = %q, want Cornell University Press", r.Publisher)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if r.Metadata["authors"] != "Armstrong, Anne K.; Krasny, Marianne E." {
		t.Fatalf("authors = %q, want Armstrong, Anne K.; Krasny, Marianne E.", r.Metadata["authors"])
	}
	if r.Metadata["language"] != "English" {
		t.Fatalf("language = %q, want English", r.Metadata["language"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "Communicating Climate Change" {
		t.Fatalf("papers round-trip failed")
	}
}

func TestOAPenSearchDOIFromFullURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{
			"uuid": "bbbb2222",
			"name": "European Climate Report",
			"handle": "20.500.12657/99999",
			"type": "item",
			"metadata": [
				{"key": "dc.title", "value": "European Climate Investment Report 2021"},
				{"key": "dc.date.issued", "value": "2021"},
				{"key": "oapen.identifier.doi", "value": "https://doi.org/10.2867/768526"},
				{"key": "publisher.name", "value": "European Investment Bank"},
				{"key": "dc.contributor.author", "value": "EIB Research"}
			]
		}]`))
	}))
	defer server.Close()

	connector := NewOAPenConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "climate"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	// normalizeSourceDOI should strip the https://doi.org/ prefix
	if r.Identifiers.DOI != "10.2867/768526" {
		t.Fatalf("DOI = %q, want 10.2867/768526 (stripped from URL)", r.Identifiers.DOI)
	}
}

func TestOAPenSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{
			"uuid": "cccc3333",
			"name": "No DOI Book",
			"handle": "20.500.12657/88888",
			"type": "item",
			"metadata": [
				{"key": "dc.title", "value": "Book Without DOI"},
				{"key": "dc.date.issued", "value": "2020"},
				{"key": "publisher.name", "value": "University Press"}
			]
		}]`))
	}))
	defer server.Close()

	connector := NewOAPenConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
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
	if r.Identifiers.CrossrefID != "oapen:20.500.12657/88888" {
		t.Fatalf("CrossrefID = %q, want oapen:20.500.12657/88888", r.Identifiers.CrossrefID)
	}
}

func TestOAPenSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"uuid": "u1", "name": "", "handle": "h1", "type": "item", "metadata": [{"key": "dc.title", "value": ""}, {"key": "oapen.identifier.doi", "value": "10.0/x1"}, {"key": "dc.date.issued", "value": "2022"}]},
			{"uuid": "u2", "name": "Fallback Name", "handle": "h2", "type": "item", "metadata": [{"key": "oapen.identifier.doi", "value": "10.0/x2"}, {"key": "dc.date.issued", "value": "2023"}]}
		]`))
	}))
	defer server.Close()

	connector := NewOAPenConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	// First item has empty dc.title and empty name → skipped.
	// Second item has no dc.title but name = "Fallback Name" → kept.
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	if response.Records[0].Title != "Fallback Name" {
		t.Fatalf("Title = %q, want Fallback Name (from item.Name fallback)", response.Records[0].Title)
	}
}

func TestOAPenSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("default limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	connector := NewOAPenConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "humanities", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
