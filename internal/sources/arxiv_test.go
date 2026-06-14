package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestArXivConnectorSearchesAndNormalizesEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query" {
			t.Fatalf("path = %q, want /api/query", r.URL.Path)
		}
		if r.URL.Query().Get("search_query") != "all:artificial photosynthesis" {
			t.Fatalf("search_query = %q", r.URL.Query().Get("search_query"))
		}
		if r.URL.Query().Get("max_results") != "1" {
			t.Fatalf("max_results = %q", r.URL.Query().Get("max_results"))
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:arxiv="http://arxiv.org/schemas/atom">
  <entry>
    <id>http://arxiv.org/abs/2401.00001v2</id>
    <title> Artificial photosynthesis catalyst preprint </title>
    <summary> A deterministic arXiv fixture. </summary>
    <published>2026-01-02T00:00:00Z</published>
    <arxiv:doi>10.1000/arxiv-doi</arxiv:doi>
    <arxiv:comment> 12 pages, 3 figures </arxiv:comment>
    <arxiv:journal_ref> Journal of Test Preprints 1 (2026) </arxiv:journal_ref>
    <author><name>Ada Lovelace</name></author>
    <link href="http://arxiv.org/abs/2401.00001v2" rel="alternate" />
    <category term="cs.AI" />
    <category term="physics.chem-ph" />
  </entry>
</feed>`))
	}))
	defer server.Close()

	connector := NewArXivConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if connector.Name() != "arxiv" {
		t.Fatalf("Name = %q", connector.Name())
	}
	if len(response.Records) != 1 {
		t.Fatalf("len(records) = %d", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "arxiv" || record.SourceID != "2401.00001" || record.Title != "Artificial photosynthesis catalyst preprint" {
		t.Fatalf("record = %#v", record)
	}
	if record.Identifiers.ArXivID != "2401.00001" || record.Identifiers.DOI != "10.1000/arxiv-doi" || record.Year != 2026 {
		t.Fatalf("identifiers/year = %#v", record)
	}
	if record.Metadata["version"] != "v2" || record.Metadata["categories"] != "cs.AI,physics.chem-ph" || record.Metadata["comment"] != "12 pages, 3 figures" || record.Metadata["journal_ref"] != "Journal of Test Preprints 1 (2026)" {
		t.Fatalf("metadata = %#v", record.Metadata)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if papers[0].Identifiers.ArXivID != "2401.00001" || papers[0].Identifiers.DOI != "10.1000/arxiv-doi" || papers[0].Abstract != "A deterministic arXiv fixture." {
		t.Fatalf("paper = %#v", papers[0])
	}
	if papers[0].SourceRefs[0].Metadata["version"] != "v2" || papers[0].SourceRefs[0].Metadata["categories"] != "cs.AI,physics.chem-ph" || papers[0].SourceRefs[0].Metadata["journal_ref"] == "" {
		t.Fatalf("paper source metadata = %#v", papers[0].SourceRefs[0].Metadata)
	}
}

func TestArXivConnectorSupportsCategoryFilterAndAcquisitionURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search_query") != "cat:cs.LG AND all:graph learning" {
			t.Fatalf("search_query = %q", r.URL.Query().Get("search_query"))
		}
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.00002v1</id>
    <title>Graph learning fixture</title>
    <summary>Fixture.</summary>
    <published>2026-01-02T00:00:00Z</published>
  </entry>
</feed>`))
	}))
	defer server.Close()

	connector := NewArXivConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "graph learning", Limit: 1, Filters: map[string]string{"category": "cs.LG"}})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "arxiv:/api/query?max_results=1&search_query=cat%3Acs.LG+AND+all%3Agraph+learning" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	urls := response.Records[0].URLs
	if len(urls) < 2 || !containsString(urls, "https://arxiv.org/pdf/2401.00002") || !containsString(urls, "https://arxiv.org/e-print/2401.00002") {
		t.Fatalf("acquisition URLs = %#v", urls)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
