package retrieval

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type fakeSearchAdapter struct {
	rebuilds int
	results  []PassageResult
}

func (f *fakeSearchAdapter) Rebuild([]parsing.ParsedDocument) error   { f.rebuilds++; return nil }
func (f *fakeSearchAdapter) Retrieve(string) ([]PassageResult, error) { return f.results, nil }
func (f *fakeSearchAdapter) Close() error                             { return nil }

func TestHybridIndexUsesReciprocalRankFusion(t *testing.T) {
	lexical := &fakeSearchAdapter{results: []PassageResult{{PassageID: "lexical-top", Text: "lexical top"}, {PassageID: "shared", Text: "lexical shared"}}}
	vector := &fakeSearchAdapter{results: []PassageResult{{PassageID: "vector-top", Text: "vector top"}, {PassageID: "shared", Text: "vector shared"}}}
	hybrid := HybridIndex{Lexical: lexical, Vector: vector, RRFK: 1}

	results, err := hybrid.Retrieve("query")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if len(results) != 3 || results[0].PassageID != "shared" {
		t.Fatalf("results = %#v, want shared overlap first", results)
	}
	if results[0].Text != "lexical shared" {
		t.Fatalf("lexical duplicate payload should win: %#v", results[0])
	}
}

func TestHybridIndexRebuildsBothBackendsAndDedupesResults(t *testing.T) {
	lexical := &fakeSearchAdapter{results: []PassageResult{{PassageID: "p1", Text: "lexical"}, {PassageID: "p2", Text: "lexical only"}}}
	vector := &fakeSearchAdapter{results: []PassageResult{{PassageID: "p1", Text: "vector duplicate"}, {PassageID: "p3", Text: "vector only"}}}
	hybrid := HybridIndex{Lexical: lexical, Vector: vector}

	if err := hybrid.Rebuild([]parsing.ParsedDocument{{PaperID: "paper-1"}}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	results, err := hybrid.Retrieve("query")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if lexical.rebuilds != 1 || vector.rebuilds != 1 {
		t.Fatalf("rebuilds lexical=%d vector=%d", lexical.rebuilds, vector.rebuilds)
	}
	if len(results) != 3 || results[0].PassageID != "p1" || results[1].PassageID != "p2" || results[2].PassageID != "p3" {
		t.Fatalf("results = %#v", results)
	}
	if results[0].Text != "lexical" {
		t.Fatalf("lexical result should win duplicate: %#v", results[0])
	}
}
