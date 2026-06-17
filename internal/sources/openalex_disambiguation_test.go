package sources

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildOpenAlexDisambiguationQueueFlagsAmbiguousEntities(t *testing.T) {
	entities := []OpenAlexEntity{{Source: "openalex-author", SourceID: "A1", DisplayName: "Ada Lovelace", WorksCount: 10}, {Source: "openalex-author", SourceID: "A2", DisplayName: "Ada Lovelace", WorksCount: 8}}
	queue := BuildOpenAlexDisambiguationQueue("Ada Lovelace", "authors", entities, "openalex:/authors?search=Ada")
	if queue.Query != "Ada Lovelace" || queue.Entity != "authors" || len(queue.Items) != 1 {
		t.Fatalf("queue = %#v", queue)
	}
	if queue.Items[0].Reason != "multiple_candidates" || len(queue.Items[0].Candidates) != 2 || queue.Items[0].ProvenanceRef == "" {
		t.Fatalf("item = %#v", queue.Items[0])
	}
}

func TestOpenAlexConnectorSearchesConcepts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/concepts" || r.URL.Query().Get("search") != "machine learning" {
			t.Fatalf("path=%s query=%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/C41008148","display_name":"Computer science","works_count":123}]}`))
	}))
	defer server.Close()
	entities, rawRef, err := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).SearchConcepts(t.Context(), SourceQuery{Terms: "machine learning", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 1 || entities[0].Source != "openalex-concept" || entities[0].SourceID != "C41008148" || rawRef == "" {
		t.Fatalf("entities=%#v raw=%s", entities, rawRef)
	}
}
