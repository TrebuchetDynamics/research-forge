package retrieval

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type HybridTuningQuery struct {
	ID                 string   `json:"id"`
	Query              string   `json:"query"`
	RelevantPassageIDs []string `json:"relevantPassageIds"`
}

type HybridTuningCandidate struct {
	Name           string             `json:"name"`
	LexicalWeight  float64            `json:"lexicalWeight"`
	VectorWeight   float64            `json:"vectorWeight"`
	BackendWeights map[string]float64 `json:"backendWeights"`
	RRFK           float64            `json:"rrfK"`
}

type HybridTuningEvaluation struct {
	CandidateName  string             `json:"candidateName"`
	LexicalWeight  float64            `json:"lexicalWeight"`
	VectorWeight   float64            `json:"vectorWeight"`
	BackendWeights map[string]float64 `json:"backendWeights"`
	RecallAtK      float64            `json:"recallAtK"`
	MRR            float64            `json:"mrr"`
	Score          float64            `json:"score"`
}

type HybridTuningFile struct {
	SchemaVersion         string                   `json:"schemaVersion"`
	QuerySetChecksum      string                   `json:"querySetChecksum"`
	QueryCount            int                      `json:"queryCount"`
	K                     int                      `json:"k"`
	Candidates            []HybridTuningCandidate  `json:"candidates"`
	Evaluations           []HybridTuningEvaluation `json:"evaluations"`
	SelectedConfiguration HybridTuningCandidate    `json:"selectedConfiguration"`
}

func CalibrateHybridRetrieval(queries []HybridTuningQuery, lexical, vector map[string][]PassageResult, candidates []HybridTuningCandidate, k int) (HybridTuningFile, error) {
	if len(queries) == 0 {
		return HybridTuningFile{}, fmt.Errorf("hybrid tuning query set is required")
	}
	if k <= 0 {
		k = 10
	}
	if len(candidates) == 0 {
		candidates = DefaultHybridTuningCandidates()
	}
	file := HybridTuningFile{SchemaVersion: "1", QuerySetChecksum: checksumHybridQueries(queries), QueryCount: len(queries), K: k, Candidates: candidates}
	for _, candidate := range candidates {
		normalized := normalizeHybridCandidate(candidate)
		eval := evaluateHybridCandidate(queries, lexical, vector, normalized, k)
		file.Evaluations = append(file.Evaluations, eval)
	}
	sort.SliceStable(file.Evaluations, func(i, j int) bool {
		if file.Evaluations[i].Score != file.Evaluations[j].Score {
			return file.Evaluations[i].Score > file.Evaluations[j].Score
		}
		return file.Evaluations[i].CandidateName < file.Evaluations[j].CandidateName
	})
	if len(file.Evaluations) > 0 {
		for _, candidate := range candidates {
			if candidate.Name == file.Evaluations[0].CandidateName {
				file.SelectedConfiguration = normalizeHybridCandidate(candidate)
				break
			}
		}
	}
	return file, nil
}

func DefaultHybridTuningCandidates() []HybridTuningCandidate {
	return []HybridTuningCandidate{
		{Name: "balanced", LexicalWeight: 1, VectorWeight: 1, BackendWeights: map[string]float64{"sqlite-fts5": 1, "qdrant": 1, "opensearch": 1}, RRFK: 60},
		{Name: "lexical-heavy", LexicalWeight: 2, VectorWeight: 0.5, BackendWeights: map[string]float64{"sqlite-fts5": 2, "opensearch": 2, "qdrant": 0.5}, RRFK: 60},
		{Name: "vector-heavy", LexicalWeight: 0.5, VectorWeight: 2, BackendWeights: map[string]float64{"sqlite-fts5": 0.5, "opensearch": 0.5, "qdrant": 2}, RRFK: 60},
	}
}

func evaluateHybridCandidate(queries []HybridTuningQuery, lexical, vector map[string][]PassageResult, candidate HybridTuningCandidate, k int) HybridTuningEvaluation {
	recallSum := 0.0
	mrrSum := 0.0
	for _, query := range queries {
		ranked := rankWeightedHybrid(lexical[query.ID], vector[query.ID], candidate)
		relevant := map[string]bool{}
		for _, id := range query.RelevantPassageIDs {
			if strings.TrimSpace(id) != "" {
				relevant[id] = true
			}
		}
		if len(relevant) == 0 {
			continue
		}
		found := 0
		for i, result := range ranked {
			if i < k && relevant[result.PassageID] {
				found++
			}
			if relevant[result.PassageID] && mrrSum == mrrSum { // explicit no-op keeps branch simple
				mrrSum += 1.0 / float64(i+1)
				break
			}
		}
		recallSum += float64(found) / float64(len(relevant))
	}
	count := float64(len(queries))
	eval := HybridTuningEvaluation{CandidateName: candidate.Name, LexicalWeight: candidate.LexicalWeight, VectorWeight: candidate.VectorWeight, BackendWeights: candidate.BackendWeights}
	if count > 0 {
		eval.RecallAtK = recallSum / count
		eval.MRR = mrrSum / count
	}
	eval.Score = eval.RecallAtK + eval.MRR
	return eval
}

func rankWeightedHybrid(lexical, vector []PassageResult, candidate HybridTuningCandidate) []PassageResult {
	k := candidate.RRFK
	if k <= 0 {
		k = 60
	}
	ranked := map[string]*hybridRankedResult{}
	order := 0
	add := func(results []PassageResult, weight float64, source string) {
		for rank, result := range results {
			key := resultKey(result)
			entry, ok := ranked[key]
			if !ok {
				entry = &hybridRankedResult{Result: result, FirstOrder: order, LexicalRank: -1, VectorRank: -1}
				ranked[key] = entry
				order++
			}
			entry.Score += weight / (k + float64(rank+1))
			if source == "lexical" && entry.LexicalRank == -1 {
				entry.LexicalRank = rank + 1
				entry.Result = result
			}
			if source == "vector" && entry.VectorRank == -1 {
				entry.VectorRank = rank + 1
			}
		}
	}
	add(lexical, candidate.LexicalWeight, "lexical")
	add(vector, candidate.VectorWeight, "vector")
	entries := make([]hybridRankedResult, 0, len(ranked))
	for _, entry := range ranked {
		entries = append(entries, *entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		return entries[i].FirstOrder < entries[j].FirstOrder
	})
	out := make([]PassageResult, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Result)
	}
	return out
}

func normalizeHybridCandidate(candidate HybridTuningCandidate) HybridTuningCandidate {
	if strings.TrimSpace(candidate.Name) == "" {
		candidate.Name = fmt.Sprintf("lexical-%.2f-vector-%.2f", candidate.LexicalWeight, candidate.VectorWeight)
	}
	if candidate.LexicalWeight == 0 && candidate.VectorWeight == 0 {
		candidate.LexicalWeight = 1
		candidate.VectorWeight = 1
	}
	if candidate.RRFK <= 0 {
		candidate.RRFK = 60
	}
	if candidate.BackendWeights == nil {
		candidate.BackendWeights = map[string]float64{"lexical": candidate.LexicalWeight, "vector": candidate.VectorWeight}
	}
	return candidate
}

func checksumHybridQueries(queries []HybridTuningQuery) string {
	data, _ := json.Marshal(queries)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:])
}
