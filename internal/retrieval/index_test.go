package retrieval

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestSQLiteFTSIndexRebuildAndRetrievePassages(t *testing.T) {
	index, err := OpenSQLiteIndex(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteIndex returned error: %v", err)
	}
	defer index.Close()
	doc := parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "paper-1-sec-1", Title: "Introduction", Passages: []parsing.Passage{{ID: "paper-1-sec-1-p-1", PaperID: "paper-1", SectionID: "paper-1-sec-1", Text: "Solar fuel catalysts split water."}}}}}
	if err := index.Rebuild([]parsing.ParsedDocument{doc}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	results, err := index.Retrieve("solar catalysts")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if len(results) != 1 || results[0].PaperID != "paper-1" || results[0].SectionID != "paper-1-sec-1" || results[0].PassageID != "paper-1-sec-1-p-1" || results[0].Text != "Solar fuel catalysts split water." {
		t.Fatalf("results = %#v", results)
	}
}

func TestAdapterSeamsExistForOptionalSearchVectorAndEmbeddingBackends(t *testing.T) {
	var _ SearchAdapter = (*SQLiteIndex)(nil)
	var _ SearchAdapter = (*OpenSearchIndex)(nil)
	var _ SearchAdapter = (*QdrantIndex)(nil)
	var _ VectorAdapter = (*QdrantIndex)(nil)
	var _ VectorAdapter = NoopVectorAdapter{}
	var _ EmbeddingAdapter = NoopEmbeddingAdapter{}
}
