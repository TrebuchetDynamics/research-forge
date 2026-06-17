package retrieval

import (
	"os"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestOptInOpenSearchIntegration(t *testing.T) {
	if os.Getenv("RFORGE_RUN_OPENSEARCH_INTEGRATION") != "1" {
		t.Skip("set RFORGE_RUN_OPENSEARCH_INTEGRATION=1 to run live OpenSearch integration")
	}
	baseURL := os.Getenv("RFORGE_OPENSEARCH_URL")
	if baseURL == "" {
		t.Skip("RFORGE_OPENSEARCH_URL is required")
	}
	index, err := NewOpenSearchIndex(OpenSearchOptions{BaseURL: baseURL, Index: os.Getenv("RFORGE_OPENSEARCH_INDEX")})
	if err != nil {
		t.Fatalf("NewOpenSearchIndex: %v", err)
	}
	doc := parsing.ParsedDocument{PaperID: "live-paper", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "live-p1", PaperID: "live-paper", SectionID: "s1", Text: "OpenSearch live integration deterministic catalyst passage."}}}}}
	if _, err := index.RebuildWithReport([]parsing.ParsedDocument{doc}); err != nil {
		t.Fatalf("RebuildWithReport: %v", err)
	}
	results, err := index.Retrieve("catalyst")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("no live OpenSearch results")
	}
}
