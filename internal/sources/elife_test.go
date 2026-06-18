package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const elifeFixture = `{
  "total": 334,
  "items": [
    {
      "id": "02478",
      "doi": "10.7554/eLife.02478",
      "title": "The role of photorespiration in C<sub>4</sub> photosynthesis",
      "published": "2014-06-16T00:00:00Z",
      "type": "research-article",
      "authorLine": "Julia Mallmann, David Heckmann ... Udo Gowik",
      "volume": "3",
      "elocationId": "e02478"
    },
    {
      "id": "58984",
      "doi": "10.7554/eLife.58984",
      "title": "Photosynthesis without β-carotene",
      "published": "2020-09-25T00:00:00Z",
      "type": "research-article",
      "authorLine": "Pengqi Xu, Volha U Chukhutsina ... Roberta Croce",
      "volume": "9",
      "elocationId": "e58984"
    }
  ]
}`

func TestELifeSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("for") == "" {
			t.Fatalf("missing 'for' param")
		}
		_, _ = w.Write([]byte(elifeFixture))
	}))
	defer server.Close()

	connector := NewELifeConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	if connector.Name() != "elife" {
		t.Fatalf("Name = %q, want elife", connector.Name())
	}
	resp, err := connector.Search(context.Background(), SourceQuery{Terms: "photosynthesis", Limit: 25})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(resp.Records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(resp.Records))
	}
	r := resp.Records[0]
	if r.Source != "elife" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "02478" {
		t.Fatalf("SourceID = %q, want 02478", r.SourceID)
	}
	if r.Identifiers.DOI != "10.7554/elife.02478" {
		t.Fatalf("DOI = %q, want 10.7554/elife.02478", r.Identifiers.DOI)
	}
	if r.Year != 2014 {
		t.Fatalf("Year = %d, want 2014", r.Year)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess should be true for eLife articles")
	}
	if r.Metadata["type"] != "research-article" {
		t.Fatalf("type = %q", r.Metadata["type"])
	}
	if r.Metadata["author_line"] != "Julia Mallmann, David Heckmann ... Udo Gowik" {
		t.Fatalf("author_line = %q", r.Metadata["author_line"])
	}
	if r.Metadata["volume"] != "3" {
		t.Fatalf("volume = %q", r.Metadata["volume"])
	}
	if r.Metadata["elocation_id"] != "e02478" {
		t.Fatalf("elocation_id = %q", r.Metadata["elocation_id"])
	}
	papers, err := PaperRecords(resp)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if len(papers) != 2 {
		t.Fatalf("papers = %d, want 2", len(papers))
	}
}

func TestELifeSearchStripsHTMLFromTitles(t *testing.T) {
	fixture := `{"total":1,"items":[{
		"id": "12345",
		"doi": "10.7554/eLife.12345",
		"title": "Evolution of C<sub>4</sub> photosynthesis via <em>Arabidopsis</em>",
		"published": "2019-03-01T00:00:00Z",
		"type": "research-article",
		"authorLine": "Test Author"
	}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	resp, err := NewELifeConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "photosynthesis"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("records = %d", len(resp.Records))
	}
	// HTML tags should be stripped and space-compacted.
	got := resp.Records[0].Title
	want := "Evolution of C 4 photosynthesis via Arabidopsis"
	if got != want {
		t.Fatalf("Title = %q, want %q", got, want)
	}
}

func TestELifeSearchSkipsItemsWithNoDOI(t *testing.T) {
	fixture := `{"total":2,"items":[
		{"id":"111","doi":"","title":"No DOI Article","published":"2020-01-01T00:00:00Z","type":"research-article"},
		{"id":"222","doi":"10.7554/eLife.22222","title":"Valid Article","published":"2021-05-01T00:00:00Z","type":"research-article"}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	resp, err := NewELifeConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("records = %d, want 1 (no-DOI item skipped)", len(resp.Records))
	}
	if resp.Records[0].Identifiers.DOI != "10.7554/elife.22222" {
		t.Fatalf("DOI = %q", resp.Records[0].Identifiers.DOI)
	}
}

func TestELifeSearchDefaultLimit(t *testing.T) {
	var gotPerPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPerPage = r.URL.Query().Get("per-page")
		_, _ = w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer server.Close()

	_, _ = NewELifeConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "biology", Limit: 0})
	if gotPerPage != "25" {
		t.Fatalf("per-page = %q, want 25", gotPerPage)
	}
}
