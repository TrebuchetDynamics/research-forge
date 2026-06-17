package retrieval

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestOpenSearchRebuildWritesMappingAndReportsPartialBulkFailures(t *testing.T) {
	var sawMapping bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/researchforge-passages":
			sawMapping = true
			if r.Method != http.MethodPut || !strings.Contains(readRequestBody(t, r), OpenSearchMappingVersion) {
				t.Fatalf("mapping request method/body = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case "/researchforge-passages/_bulk":
			_, _ = w.Write([]byte(`{"errors":true,"items":[{"index":{"_id":"p1","status":201}},{"index":{"_id":"p2","status":400,"error":{"type":"mapper_parsing_exception","reason":"bad text"}}}]}`))
		case "/researchforge-passages/_refresh":
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	index, err := NewOpenSearchIndex(OpenSearchOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewOpenSearchIndex: %v", err)
	}
	doc := parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "ok"}, {ID: "p2", PaperID: "paper-1", SectionID: "s1", Text: "bad"}}}}}
	report, err := index.RebuildWithReport([]parsing.ParsedDocument{doc})
	if err != nil {
		t.Fatalf("RebuildWithReport: %v", err)
	}
	if !sawMapping || report.MappingVersion != OpenSearchMappingVersion || report.Attempted != 2 || report.Indexed != 1 || report.Failed != 1 || report.Failures[0].Reason != "bad text" {
		t.Fatalf("report=%#v sawMapping=%t", report, sawMapping)
	}
}

func TestOpenSearchRetrieveIncludesHighlights(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/researchforge-passages/_search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(toJSON(t, request), "highlight") {
			t.Fatalf("missing highlight request: %#v", request)
		}
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"PaperID":"paper-1","SectionID":"s1","PassageID":"p1","Text":"Solar fuel catalysts split water."},"highlight":{"Text":["<em>Solar</em> fuel catalysts"]}}]}}`))
	}))
	defer server.Close()
	index, err := NewOpenSearchIndex(OpenSearchOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewOpenSearchIndex: %v", err)
	}
	results, err := index.Retrieve("solar")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) != 1 || len(results[0].Highlights) != 1 || !strings.Contains(results[0].Highlights[0], "<em>Solar</em>") {
		t.Fatalf("results = %#v", results)
	}
}
