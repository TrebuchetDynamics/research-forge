package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAlexConnectorExpandsCitationGraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/works/W123":
			_, _ = w.Write([]byte(`{"id":"https://openalex.org/W123","referenced_works":["https://openalex.org/WREF1","https://openalex.org/WREF2"]}`))
		case "/works":
			if r.URL.Query().Get("filter") != "cites:W123" {
				t.Fatalf("filter = %q", r.URL.Query().Get("filter"))
			}
			_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/WCITE1","doi":"https://doi.org/10.1000/cite","title":"Citing work","publication_year":2026}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	expansion, err := connector.ExpandCitationGraph(context.Background(), OpenAlexGraphQuery{WorkID: "https://openalex.org/W123", Limit: 2, Direction: SemanticScholarDirectionBoth})
	if err != nil {
		t.Fatalf("ExpandCitationGraph returned error: %v", err)
	}
	if expansion.RawRef != "openalex:/works/W123/both?limit=2" {
		t.Fatalf("RawRef = %q", expansion.RawRef)
	}
	want := []CitationEdge{{SourceID: "W123", TargetID: "WREF1"}, {SourceID: "W123", TargetID: "WREF2"}, {SourceID: "WCITE1", TargetID: "W123"}}
	if len(expansion.Edges) != len(want) {
		t.Fatalf("edges = %#v", expansion.Edges)
	}
	for i := range want {
		if expansion.Edges[i] != want[i] {
			t.Fatalf("edge[%d] = %#v, want %#v", i, expansion.Edges[i], want[i])
		}
	}
	if expansion.Records["WCITE1"].Title != "Citing work" || expansion.Records["WREF1"].Identifiers.OpenAlexID != "WREF1" {
		t.Fatalf("records = %#v", expansion.Records)
	}
}

func TestOpenAlexConnectorSupportsCursorPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "abc123" {
			t.Fatalf("cursor = %q", r.URL.Query().Get("cursor"))
		}
		if r.URL.Query().Get("per-page") != "1" {
			t.Fatalf("per-page = %q", r.URL.Query().Get("per-page"))
		}
		_, _ = w.Write([]byte(`{"meta":{"next_cursor":"def456"},"results":[{"id":"https://openalex.org/W456","doi":"https://doi.org/10.1000/page","title":"Paged artificial photosynthesis work","publication_year":2026}]}`))
	}))
	defer server.Close()

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1, PageCursor: "abc123"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.NextPageCursor != "def456" {
		t.Fatalf("NextPageCursor = %q", response.NextPageCursor)
	}
	if response.Records[0].SourceID != "W456" {
		t.Fatalf("record = %#v", response.Records[0])
	}
}

func TestOpenAlexConnectorRetriesRateLimitWithBackoff(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "3")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W789","doi":"https://doi.org/10.1000/rate","title":"Rate limited artificial photosynthesis work","publication_year":2026}]}`))
	}))
	defer server.Close()
	var slept []time.Duration

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second, MaxRetries: 1, Sleep: func(duration time.Duration) {
		slept = append(slept, duration)
	}}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if requests != 2 || len(slept) != 1 || slept[0] != 3*time.Second {
		t.Fatalf("requests=%d slept=%#v", requests, slept)
	}
	if response.Records[0].SourceID != "W789" {
		t.Fatalf("record = %#v", response.Records[0])
	}
}

func TestOpenAlexConnectorMapsConceptsAndDomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W999","title":"Domain work","concepts":[{"id":"https://openalex.org/C41008148","display_name":"Computer science","score":0.91}],"primary_topic":{"display_name":"Machine learning","domain":{"display_name":"Physical Sciences"},"field":{"display_name":"Computer Science"},"subfield":{"display_name":"Artificial Intelligence"}}}]}`))
	}))
	defer server.Close()
	response, err := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(context.Background(), SourceQuery{Terms: "ml", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	metadata := response.Records[0].Metadata
	for key, want := range map[string]string{"concepts": "Computer science", "concept_ids": "C41008148", "top_concept": "Computer science", "topic": "Machine learning", "domain": "Physical Sciences", "field": "Computer Science", "subfield": "Artificial Intelligence"} {
		if metadata[key] != want {
			t.Fatalf("metadata[%s] = %q, want %q (metadata=%#v)", key, metadata[key], want, metadata)
		}
	}
}

func TestOpenAlexConnectorSearchesAndNormalizesWorks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q, want /works", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "artificial photosynthesis" {
			t.Fatalf("search = %q", r.URL.Query().Get("search"))
		}
		if r.URL.Query().Get("per-page") != "2" {
			t.Fatalf("per-page = %q", r.URL.Query().Get("per-page"))
		}
		if r.URL.Query().Get("filter") != "type:review,from_publication_date:2020-01-01" {
			t.Fatalf("filter = %q", r.URL.Query().Get("filter"))
		}
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "https://openalex.org/W123",
				"doi": "https://doi.org/10.1000/example",
				"title": "Artificial photosynthesis catalyst review",
				"publication_year": 2026,
				"type": "review",
				"open_access": {"is_oa": true, "oa_status": "gold"},
				"primary_location": {"landing_page_url": "https://example.org/paper", "license": "cc-by"},
				"concepts": [{"id":"https://openalex.org/C41008148","display_name":"Computer science","score":0.8},{"display_name":"Catalysis","score":0.7}],
				"related_works": ["https://openalex.org/W999", "https://openalex.org/W998"]
			}]
		}`))
	}))
	defer server.Close()

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 2, Filters: map[string]string{"filter": "type:review,from_publication_date:2020-01-01"}})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if connector.Name() != "openalex" {
		t.Fatalf("Name = %q", connector.Name())
	}
	if response.RawRef != "openalex:/works?filter=type%3Areview%2Cfrom_publication_date%3A2020-01-01&per-page=2&search=artificial+photosynthesis" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("len(records) = %d", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "openalex" || record.SourceID != "W123" || record.Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("record = %#v", record)
	}
	if record.Identifiers.DOI != "10.1000/example" || record.Identifiers.OpenAlexID != "W123" || record.Year != 2026 {
		t.Fatalf("record identifiers/year = %#v", record)
	}
	if record.Metadata["type"] != "review" || record.Metadata["oa_status"] != "gold" || record.Metadata["concepts"] != "Computer science; Catalysis" || record.Metadata["related_openalex_ids"] != "W999; W998" {
		t.Fatalf("metadata = %#v", record.Metadata)
	}
	if record.OpenAccess != true || record.License != "cc-by" || len(record.URLs) != 1 || record.URLs[0] != "https://example.org/paper" {
		t.Fatalf("access metadata = %#v", record)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("papers = %#v", papers)
	}
	if papers[0].Identifiers.DOI != "10.1000/example" || papers[0].Identifiers.OpenAlexID != "W123" {
		t.Fatalf("paper identifiers = %#v", papers[0].Identifiers)
	}
	if !papers[0].OpenAccess || papers[0].License != "cc-by" || papers[0].URLs[0] != "https://example.org/paper" {
		t.Fatalf("paper source metadata = %#v", papers[0])
	}
	if papers[0].SourceRefs[0].Metadata["oa_status"] != "gold" || papers[0].SourceRefs[0].Metadata["type"] != "review" || papers[0].SourceRefs[0].Metadata["concepts"] == "" {
		t.Fatalf("paper source ref metadata = %#v", papers[0].SourceRefs[0].Metadata)
	}
}

func TestOpenAlexConnectorExpandsCitationGraphFromDOI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("unexpected path: %s (want /works for all DOI-based lookups)", r.URL.Path)
		}
		filter := r.URL.Query().Get("filter")
		switch filter {
		case "doi:10.1038/s41467-023-42110-y":
			// DOI resolution — returns the resolved work with referenced_works
			_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W9999","doi":"https://doi.org/10.1038/s41467-023-42110-y","title":"Multi-level FeFET crossbar","publication_year":2023,"referenced_works":["https://openalex.org/WREF1"]}]}`))
		case "cites:W9999":
			_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/WCITE1","doi":"https://doi.org/10.1000/cite1","title":"Citing FeFET work","publication_year":2024}]}`))
		default:
			t.Fatalf("unexpected filter: %q", filter)
		}
	}))
	defer server.Close()

	connector := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	expansion, err := connector.ExpandCitationGraph(context.Background(), OpenAlexGraphQuery{
		WorkID:    "10.1038/s41467-023-42110-y",
		Limit:     10,
		Direction: SemanticScholarDirectionBoth,
	})
	if err != nil {
		t.Fatalf("ExpandCitationGraph returned error: %v", err)
	}
	want := []CitationEdge{
		{SourceID: "W9999", TargetID: "WREF1"},
		{SourceID: "WCITE1", TargetID: "W9999"},
	}
	if len(expansion.Edges) != len(want) {
		t.Fatalf("edges = %#v, want %#v", expansion.Edges, want)
	}
	for i := range want {
		if expansion.Edges[i] != want[i] {
			t.Fatalf("edge[%d] = %#v, want %#v", i, expansion.Edges[i], want[i])
		}
	}
	if expansion.SeedID != "W9999" {
		t.Fatalf("SeedID = %q, want W9999", expansion.SeedID)
	}
}
