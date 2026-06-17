package retrieval

import (
	"os"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestOptInQdrantIntegration(t *testing.T) {
	if os.Getenv("RFORGE_RUN_QDRANT_INTEGRATION") != "1" {
		t.Skip("set RFORGE_RUN_QDRANT_INTEGRATION=1 to run live Qdrant integration")
	}
	baseURL := os.Getenv("RFORGE_QDRANT_URL")
	if baseURL == "" {
		t.Skip("RFORGE_QDRANT_URL is required")
	}
	index, err := NewQdrantIndex(QdrantOptions{BaseURL: baseURL, Collection: os.Getenv("RFORGE_QDRANT_COLLECTION"), Embeddings: DeterministicEmbedding{Dimensions: 8}, InvalidateBeforeUpsert: true})
	if err != nil {
		t.Fatalf("NewQdrantIndex: %v", err)
	}
	doc := parsing.ParsedDocument{PaperID: "live-paper", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "live-qdrant-p1", PaperID: "live-paper", SectionID: "s1", Text: "Qdrant live integration deterministic catalyst passage."}}}}}
	if _, err := index.RebuildWithReport([]parsing.ParsedDocument{doc}); err != nil {
		t.Fatalf("RebuildWithReport: %v", err)
	}
	results, err := index.Retrieve("catalyst")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("no live Qdrant results")
	}
}
