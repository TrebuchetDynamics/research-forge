package retrieval

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func BenchmarkIndexRebuild(b *testing.B) {
	doc := parsing.ParsedDocument{PaperID: "p", Sections: []parsing.Section{{ID: "s", Passages: []parsing.Passage{{ID: "p1", PaperID: "p", SectionID: "s", Text: "solar catalysts split water"}}}}}
	for i := 0; i < b.N; i++ {
		idx, _ := OpenSQLiteIndex(filepath.Join(b.TempDir(), "x.db"))
		_ = idx.Rebuild([]parsing.ParsedDocument{doc})
		_ = idx.Close()
	}
}
