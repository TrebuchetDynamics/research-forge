package retrieval

import (
	"fmt"
	"sort"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// HybridIndex combines lexical and vector retrieval results with deterministic de-duplication.
type HybridIndex struct {
	Lexical SearchAdapter
	Vector  SearchAdapter
	RRFK    float64
}

// Rebuild updates both backing indexes.
func (h HybridIndex) Rebuild(docs []parsing.ParsedDocument) error {
	if h.Lexical == nil || h.Vector == nil {
		return fmt.Errorf("hybrid retrieval requires lexical and vector backends")
	}
	if err := h.Lexical.Rebuild(docs); err != nil {
		return err
	}
	return h.Vector.Rebuild(docs)
}

// Retrieve returns reciprocal-rank-fused lexical/vector hits with deterministic de-duplication.
func (h HybridIndex) Retrieve(query string) ([]PassageResult, error) {
	if h.Lexical == nil || h.Vector == nil {
		return nil, fmt.Errorf("hybrid retrieval requires lexical and vector backends")
	}
	lexical, err := h.Lexical.Retrieve(query)
	if err != nil {
		return nil, err
	}
	vector, err := h.Vector.Retrieve(query)
	if err != nil {
		return nil, err
	}
	k := h.RRFK
	if k <= 0 {
		k = 60
	}
	ranked := map[string]*hybridRankedResult{}
	order := 0
	addResults := func(results []PassageResult, source string) {
		for rank, result := range results {
			key := resultKey(result)
			entry, ok := ranked[key]
			if !ok {
				entry = &hybridRankedResult{Result: result, FirstOrder: order, LexicalRank: -1, VectorRank: -1}
				ranked[key] = entry
				order++
			}
			entry.Score += 1 / (k + float64(rank+1))
			if source == "lexical" && entry.LexicalRank == -1 {
				entry.LexicalRank = rank + 1
				entry.Result = result
			}
			if source == "vector" && entry.VectorRank == -1 {
				entry.VectorRank = rank + 1
			}
		}
	}
	addResults(lexical, "lexical")
	addResults(vector, "vector")
	entries := make([]hybridRankedResult, 0, len(ranked))
	for _, entry := range ranked {
		entries = append(entries, *entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if (entries[i].LexicalRank > 0) != (entries[j].LexicalRank > 0) {
			return entries[i].LexicalRank > 0
		}
		return entries[i].FirstOrder < entries[j].FirstOrder
	})
	out := make([]PassageResult, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Result)
	}
	return out, nil
}

// HybridRankedResult is an internal reciprocal-rank-fusion accumulator.
type hybridRankedResult struct {
	Result      PassageResult
	Score       float64
	FirstOrder  int
	LexicalRank int
	VectorRank  int
}

// Close closes both backing indexes.
func (h HybridIndex) Close() error {
	var first error
	if h.Lexical != nil {
		first = h.Lexical.Close()
	}
	if h.Vector != nil {
		if err := h.Vector.Close(); first == nil {
			first = err
		}
	}
	return first
}

func resultKey(result PassageResult) string {
	if result.PassageID != "" {
		return result.PassageID
	}
	return result.PaperID + "\x00" + result.SectionID + "\x00" + result.Text
}
