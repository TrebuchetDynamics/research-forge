package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// resolverFixtureServer serves minimal scholarly API responses for resolver tests.
func resolverFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// OpenAlex works (search by DOI filter or title)
	mux.HandleFunc("/openalex/works", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"meta": {"count": 1},
			"results": [{
				"id": "https://openalex.org/W123",
				"doi": "https://doi.org/10.1000/example",
				"title": "Example Paper on Transformer Scaling",
				"publication_year": 2023,
				"abstract_inverted_index": {"scaling": [0], "laws": [1]},
				"primary_location": {"landing_page_url": "https://example.com/paper", "pdf_url": "https://example.com/paper.pdf", "license": "cc-by"},
				"open_access": {"is_oa": true, "oa_status": "gold"}
			}]
		}`)
	})

	// Crossref works
	mux.HandleFunc("/crossref/works", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"status": "ok",
			"message": {
				"DOI": "10.1000/example",
				"title": ["Example Paper on Transformer Scaling"],
				"type": "journal-article",
				"published-print": {"date-parts": [[2023]]},
				"container-title": ["Journal of Scaling"],
				"publisher": "Scaling Press",
				"abstract": "<jats:p>We study scaling laws.</jats:p>"
			}
		}`)
	})

	// Unpaywall
	mux.HandleFunc("/unpaywall/v2/", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("email"), os.Getenv("RFORGE_UNPAYWALL_EMAIL"); got != want {
			t.Errorf("Unpaywall contact = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"doi": "10.1000/example",
			"is_oa": true,
			"oa_status": "gold",
			"best_oa_location": {
				"url": "https://example.com/paper.pdf",
				"url_for_pdf": "https://example.com/paper.pdf",
				"license": "cc-by",
				"host_type": "publisher"
			}
		}`)
	})

	// arXiv lookup
	mux.HandleFunc("/arxiv/api/query", func(w http.ResponseWriter, r *http.Request) {
		idList := r.URL.Query().Get("id_list")
		if idList == "2301.12345" {
			w.Header().Set("Content-Type", "application/atom+xml")
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2301.12345</id>
    <title>Scaling Laws for Transformer Models</title>
    <summary>We study scaling laws for transformers.</summary>
    <published>2023-01-31T00:00:00Z</published>
    <link href="http://arxiv.org/abs/2301.12345" rel="alternate"/>
    <link href="http://arxiv.org/pdf/2301.12345v1" rel="related"/>
  </entry>
</feed>`)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprintln(w, `<feed xmlns="http://www.w3.org/2005/Atom"></feed>`)
	})

	// Semantic Scholar paper lookup and search
	mux.HandleFunc("/s2/graph/v1/paper/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"total": 1,
			"data": [{
				"paperId": "abc123",
				"title": "Example Paper on Transformer Scaling",
				"abstract": "We study scaling laws.",
				"year": 2023,
				"venue": "Journal of Scaling",
				"url": "https://example.com/paper",
				"isOpenAccess": true,
				"citationCount": 100,
				"externalIds": {"DOI": "10.1000/example"}
			}]
		}`)
	})

	var listener net.Listener
	var err error
	if base, _ := strconv.Atoi(os.Getenv("AUTOWORK_PORT_BASE")); base > 0 {
		for port := base; port < base+100; port++ {
			listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err == nil {
				break
			}
		}
	} else {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	t.Cleanup(server.Close)
	return server
}

// resolverClient was removed — unused

func TestClassifyPaperInput(t *testing.T) {
	cases := map[string]string{
		"10.1000/example":             "doi",
		"W123456789":                  "openalex",
		"2301.12345":                  "arxiv",
		"arXiv:2301.12345":            "arxiv",
		"http://arxiv.org/abs/2301.1": "arxiv",
		"PMID:12345":                  "pmid",
		"PMC1234567":                  "pmcid",
		"CorpusID:12345":              "semantic-scholar",
		"https://doi.org/10.1000/foo": "url",
		"Some Paper Title":            "title",
		"":                            "title",
	}
	for input, want := range cases {
		if got := classifyPaperInput(input); got != want {
			t.Errorf("classifyPaperInput(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLookupPaperByArXivIDNormalizesVersion(t *testing.T) {
	server := resolverFixtureServer(t)
	client := NewHTTPClient(HTTPClientOptions{BaseURL: server.URL + "/arxiv", MaxRetries: 0})
	conn := NewArXivConnector(client)

	record, _, err := conn.LookupPaperByArXivID(context.Background(), "2301.12345v2")
	if err != nil {
		t.Fatalf("LookupPaperByArXivID: %v", err)
	}
	if record.SourceID != "2301.12345" {
		t.Fatalf("SourceID = %q, want normalized 2301.12345", record.SourceID)
	}
}

func TestPaperResolutionJSONShape(t *testing.T) {
	server := resolverFixtureServer(t)
	// Resolve a known DOI that will match openalex + crossref prefixes.
	// We only exercise JSON shape via the arXiv lookup path.
	client := NewHTTPClient(HTTPClientOptions{BaseURL: server.URL + "/arxiv", MaxRetries: 0})
	conn := NewArXivConnector(client)
	record, _, err := conn.LookupPaperByArXivID(context.Background(), "2301.12345")
	if err != nil {
		t.Fatalf("LookupPaperByArXivID: %v", err)
	}
	resolution := PaperResolution{
		Input: "2301.12345",
		Paper: mustPaperRecord(t, record),
		Sources: []ResolutionSource{{
			Source:  "arxiv",
			Title:   record.Title,
			RawRef:  "arxiv:/api/query",
			MatchBy: "arxiv-id",
		}},
		Resolved: true,
	}
	data, err := json.Marshal(resolution)
	if err != nil {
		t.Fatalf("marshal resolution: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal resolution: %v", err)
	}
	if _, ok := decoded["paper"]; !ok {
		t.Fatal("paper missing from JSON")
	}
	if _, ok := decoded["sources"]; !ok {
		t.Fatal("sources missing from JSON")
	}
	if !strings.Contains(string(data), `"resolved":true`) {
		t.Fatal("resolved flag missing")
	}
}

func TestResolvePaperUsesConfiguredContact(t *testing.T) {
	t.Setenv("RFORGE_UNPAYWALL_EMAIL", "researcher@example.org")
	server := resolverFixtureServer(t)
	client := NewHTTPClient(HTTPClientOptions{BaseURL: server.URL + "/unpaywall"})
	result, err := ResolvePaper(context.Background(), client, "10.1000/example")
	if err != nil || result.OA == nil || !result.OA.IsOA || result.OA.License != "cc-by" {
		t.Fatalf("configured Unpaywall lookup failed: %+v, %v", result, err)
	}
}

func TestTitleMatch(t *testing.T) {
	cases := []struct {
		result string
		query  string
		want   bool
	}{
		{"Example Paper on Transformer Scaling", "Transformer Scaling", true},
		{"Example Paper on Transformer Scaling", "Example Paper on Transformer Scaling", true},
		{"Completely Unrelated Title", "Transformer Scaling", false},
	}
	for _, c := range cases {
		if got := titleMatch(c.result, c.query); got != c.want {
			t.Errorf("titleMatch(%q, %q) = %t, want %t", c.result, c.query, got, c.want)
		}
	}
}

func TestExtractIDsFromURL(t *testing.T) {
	if doi := extractDOIFromURL("https://doi.org/10.1000/example"); doi != "10.1000/example" {
		t.Errorf("extractDOIFromURL = %q, want 10.1000/example", doi)
	}
	if id := extractArXivIDFromURL("https://arxiv.org/abs/2301.12345v2"); id != "2301.12345" {
		t.Errorf("extractArXivIDFromURL = %q, want 2301.12345", id)
	}
	if id := extractArXivIDFromURL("https://arxiv.org/pdf/2301.99999"); id != "2301.99999" {
		t.Errorf("extractArXivIDFromURL (pdf) = %q, want 2301.99999", id)
	}
}

// mustPaperRecord converts a SourceRecord into a library PaperRecord for tests.
func mustPaperRecord(t *testing.T, record SourceRecord) library.PaperRecord {
	t.Helper()
	paper, err := library.NewPaperRecord(library.PaperRecordInput{
		Title:    record.Title,
		Abstract: record.Abstract,
		Identifiers: library.Identifiers{
			DOI:        record.Identifiers.DOI,
			ArXivID:    record.Identifiers.ArXivID,
			OpenAlexID: record.Identifiers.OpenAlexID,
		},
	})
	if err != nil {
		t.Fatalf("mustPaperRecord: %v", err)
	}
	return paper
}
