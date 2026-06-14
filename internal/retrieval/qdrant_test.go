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

func TestDeterministicEmbeddingProducesStableNormalizedVector(t *testing.T) {
	emb := DeterministicEmbedding{Dimensions: 4}
	left, err := emb.Embed("solar catalysts solar")
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}
	right, err := emb.Embed("solar catalysts solar")
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}
	if len(left) != 4 || len(right) != 4 {
		t.Fatalf("vector lengths = %d/%d", len(left), len(right))
	}
	for i := range left {
		if left[i] != right[i] {
			t.Fatalf("vectors differ: %#v %#v", left, right)
		}
	}
}

func TestQdrantIndexRebuildAndRetrievePassages(t *testing.T) {
	var sawUpsert, sawSearch bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/researchforge_passages/points":
			sawUpsert = true
			if r.URL.Query().Get("wait") != "true" {
				t.Fatalf("wait = %q", r.URL.Query().Get("wait"))
			}
			data := readQdrantBody(t, r)
			if !strings.Contains(data, `"PaperID":"paper-1"`) || !strings.Contains(data, `"vector"`) {
				t.Fatalf("upsert payload = %s", data)
			}
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		case "/collections/researchforge_passages/points/search":
			sawSearch = true
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode search request: %v", err)
			}
			if _, ok := request["vector"].([]any); !ok {
				t.Fatalf("search request missing vector: %#v", request)
			}
			_, _ = w.Write([]byte(`{"result":[{"payload":{"PaperID":"paper-1","SectionID":"s1","PassageID":"p1","Text":"Solar fuel catalysts split water."}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	index, err := NewQdrantIndex(QdrantOptions{BaseURL: server.URL, Embeddings: DeterministicEmbedding{Dimensions: 4}})
	if err != nil {
		t.Fatalf("NewQdrantIndex returned error: %v", err)
	}
	doc := parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Solar fuel catalysts split water."}}}}}
	if err := index.Rebuild([]parsing.ParsedDocument{doc}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	results, err := index.Retrieve("solar catalysts")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if !sawUpsert || !sawSearch {
		t.Fatalf("saw upsert=%t search=%t", sawUpsert, sawSearch)
	}
	if len(results) != 1 || results[0].PaperID != "paper-1" || results[0].PassageID != "p1" {
		t.Fatalf("results = %#v", results)
	}
}

func readQdrantBody(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
