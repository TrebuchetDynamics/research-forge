package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCrossrefConnectorSearchesAndNormalizesWorks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q, want /works", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "artificial photosynthesis" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("rows") != "1" {
			t.Fatalf("rows = %q", r.URL.Query().Get("rows"))
		}
		_, _ = w.Write([]byte(`{
			"message": {"items": [{
				"DOI": "10.5555/CROSSREF.EXAMPLE",
				"title": ["Artificial photosynthesis Crossref fixture"],
				"abstract": "<jats:p>Deterministic Crossref abstract.</jats:p>",
				"published-print": {"date-parts": [[2026, 1, 2]]},
				"container-title": ["Journal of Test Fixtures"],
				"publisher": "Fixture Publisher",
				"URL": "https://doi.org/10.5555/crossref.example",
				"type": "journal-article",
				"reference-count": 12
			}]}
		}`))
	}))
	defer server.Close()

	connector := NewCrossrefConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if connector.Name() != "crossref" {
		t.Fatalf("Name = %q", connector.Name())
	}
	if response.RawRef != "crossref:/works?query=artificial+photosynthesis&rows=1" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("len(records) = %d", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "crossref" || record.SourceID != "10.5555/crossref.example" || record.Title != "Artificial photosynthesis Crossref fixture" {
		t.Fatalf("record = %#v", record)
	}
	if record.Identifiers.DOI != "10.5555/crossref.example" || record.Identifiers.CrossrefID != "10.5555/crossref.example" || record.Year != 2026 {
		t.Fatalf("identifiers/year = %#v", record)
	}
	if record.Abstract != "Deterministic Crossref abstract." || len(record.URLs) != 1 || record.URLs[0] != "https://doi.org/10.5555/crossref.example" {
		t.Fatalf("text/urls = %#v", record)
	}
	if record.Metadata["type"] != "journal-article" || record.Metadata["reference_count"] != "12" {
		t.Fatalf("metadata = %#v", record.Metadata)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if papers[0].Identifiers.DOI != "10.5555/crossref.example" || papers[0].Identifiers.CrossrefID != "10.5555/crossref.example" {
		t.Fatalf("paper identifiers = %#v", papers[0].Identifiers)
	}
	if papers[0].Venue != "Journal of Test Fixtures" || papers[0].Publisher != "Fixture Publisher" {
		t.Fatalf("paper venue/publisher = %#v", papers[0])
	}
}
