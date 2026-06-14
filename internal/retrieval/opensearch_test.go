package retrieval

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestOpenSearchIndexRebuildAndRetrievePassages(t *testing.T) {
	var sawBulk, sawRefresh, sawSearch bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/researchforge-passages/_bulk":
			sawBulk = true
			if r.Header.Get("Content-Type") != "application/x-ndjson" {
				t.Fatalf("bulk content type = %q", r.Header.Get("Content-Type"))
			}
			data := readRequestBody(t, r)
			if !strings.Contains(data, `"PaperID":"paper-1"`) || !strings.Contains(data, `"Text":"Solar fuel catalysts split water."`) {
				t.Fatalf("bulk payload = %s", data)
			}
			_, _ = w.Write([]byte(`{"errors":false}`))
		case "/researchforge-passages/_refresh":
			sawRefresh = true
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/researchforge-passages/_search":
			sawSearch = true
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode search request: %v", err)
			}
			if !strings.Contains(toJSON(t, request), "solar catalysts") {
				t.Fatalf("search request = %#v", request)
			}
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"PaperID":"paper-1","SectionID":"s1","PassageID":"p1","Text":"Solar fuel catalysts split water."}}]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	index, err := NewOpenSearchIndex(OpenSearchOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewOpenSearchIndex returned error: %v", err)
	}
	doc := parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Solar fuel catalysts split water."}}}}}
	if err := index.Rebuild([]parsing.ParsedDocument{doc}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	results, err := index.Retrieve("solar catalysts")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if !sawBulk || !sawRefresh || !sawSearch {
		t.Fatalf("saw bulk=%t refresh=%t search=%t", sawBulk, sawRefresh, sawSearch)
	}
	if len(results) != 1 || results[0].PaperID != "paper-1" || results[0].PassageID != "p1" {
		t.Fatalf("results = %#v", results)
	}
}

func readRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}

func toJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}
