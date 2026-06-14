package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSemanticScholarCitationGraphFetchesReferencesAndCitations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "2" {
			t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
		}
		switch r.URL.Path {
		case "/graph/v1/paper/seed-paper/references":
			_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"ref-1","title":"Reference one"}},{"citedPaper":{"paperId":"ref-2","title":"Reference two"}}]}`))
		case "/graph/v1/paper/seed-paper/citations":
			_, _ = w.Write([]byte(`{"data":[{"citingPaper":{"paperId":"citing-1","title":"Citing one"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	connector := NewSemanticScholarConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	expansion, err := connector.ExpandCitationGraph(context.Background(), SemanticScholarGraphQuery{PaperID: "seed-paper", Limit: 2, Direction: SemanticScholarDirectionBoth})
	if err != nil {
		t.Fatalf("ExpandCitationGraph returned error: %v", err)
	}
	if expansion.RawRef != "semantic-scholar:/graph/v1/paper/seed-paper/references+citations?limit=2" {
		t.Fatalf("RawRef = %q", expansion.RawRef)
	}
	if len(expansion.Edges) != 3 {
		t.Fatalf("edges = %#v, want 3", expansion.Edges)
	}
	want := []CitationEdge{{SourceID: "seed-paper", TargetID: "ref-1"}, {SourceID: "seed-paper", TargetID: "ref-2"}, {SourceID: "citing-1", TargetID: "seed-paper"}}
	for i := range want {
		if expansion.Edges[i] != want[i] {
			t.Fatalf("edge[%d] = %#v, want %#v", i, expansion.Edges[i], want[i])
		}
	}
	if expansion.Records["ref-1"].Title != "Reference one" || expansion.Records["citing-1"].Title != "Citing one" {
		t.Fatalf("records = %#v", expansion.Records)
	}
}
