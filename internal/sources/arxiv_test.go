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
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.00001v2</id>
    <title> Artificial photosynthesis catalyst preprint </title>
    <summary> A deterministic arXiv fixture. </summary>
    <published>2026-01-02T00:00:00Z</published>
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
	if record.Identifiers.ArXivID != "2401.00001" || record.Year != 2026 {
		t.Fatalf("identifiers/year = %#v", record)
	}
	if record.Metadata["version"] != "v2" || record.Metadata["categories"] != "cs.AI,physics.chem-ph" {
		t.Fatalf("metadata = %#v", record.Metadata)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if papers[0].Identifiers.ArXivID != "2401.00001" || papers[0].Abstract != "A deterministic arXiv fixture." {
		t.Fatalf("paper = %#v", papers[0])
	}
	if papers[0].SourceRefs[0].Metadata["version"] != "v2" || papers[0].SourceRefs[0].Metadata["categories"] != "cs.AI,physics.chem-ph" {
		t.Fatalf("paper source metadata = %#v", papers[0].SourceRefs[0].Metadata)
	}
}
