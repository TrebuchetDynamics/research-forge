package retrieval

import (
	"fmt"
	"sort"
)

type RetrievalBenchmarkFixture struct {
	SchemaVersion  string                                `json:"schemaVersion"`
	FixtureID      string                                `json:"fixtureId,omitempty"`
	Queries        []HybridTuningQuery                   `json:"queries"`
	BackendResults map[string]map[string][]PassageResult `json:"backendResults"`
}

type RetrievalBenchmarkReport struct {
	SchemaVersion        string                          `json:"schemaVersion"`
	FixtureID            string                          `json:"fixtureId"`
	QuerySetChecksum     string                          `json:"querySetChecksum"`
	QueryCount           int                             `json:"queryCount"`
	K                    int                             `json:"k"`
	Backends             []RetrievalBackendBenchmark     `json:"backends"`
	SelectedBackend      string                          `json:"selectedBackend"`
	QueryResults         []RetrievalBenchmarkQueryResult `json:"queryResults"`
	ReproducibilityNotes []string                        `json:"reproducibilityNotes"`
	PrivacyNotes         []string                        `json:"privacyNotes"`
}

type RetrievalBackendBenchmark struct {
	Backend             string  `json:"backend"`
	RecallAtK           float64 `json:"recallAtK"`
	MRR                 float64 `json:"mrr"`
	Score               float64 `json:"score"`
	ReproducibilityNote string  `json:"reproducibilityNote,omitempty"`
	PrivacyNote         string  `json:"privacyNote,omitempty"`
}

type RetrievalBenchmarkQueryResult struct {
	QueryID string                    `json:"queryId"`
	Backend string                    `json:"backend"`
	TopK    []string                  `json:"topK"`
	Metrics RetrievalBackendBenchmark `json:"metrics"`
}

func DefaultRetrievalBenchmarkFixture() RetrievalBenchmarkFixture {
	queries := []HybridTuningQuery{
		{ID: "q1", Query: "solar catalyst", RelevantPassageIDs: []string{"p-solar"}},
		{ID: "q2", Query: "screening bias", RelevantPassageIDs: []string{"p-bias"}},
		{ID: "q3", Query: "forest plot heterogeneity", RelevantPassageIDs: []string{"p-forest"}},
	}
	backendResults := map[string]map[string][]PassageResult{
		"sqlite-fts": {
			"q1": {{PassageID: "p-solar"}, {PassageID: "p-noise-1"}},
			"q2": {{PassageID: "p-bias"}, {PassageID: "p-noise-2"}},
			"q3": {{PassageID: "p-noise-3"}, {PassageID: "p-forest"}},
		},
		"opensearch": {
			"q1": {{PassageID: "p-solar"}},
			"q2": {{PassageID: "p-noise-2"}, {PassageID: "p-bias"}},
			"q3": {{PassageID: "p-forest"}},
		},
		"qdrant": {
			"q1": {{PassageID: "p-noise-1"}, {PassageID: "p-solar"}},
			"q2": {{PassageID: "p-bias"}},
			"q3": {{PassageID: "p-forest"}},
		},
	}
	backendResults["hybrid"] = map[string][]PassageResult{}
	candidate := HybridTuningCandidate{Name: "balanced", LexicalWeight: 1, VectorWeight: 1, RRFK: 60}
	for _, query := range queries {
		backendResults["hybrid"][query.ID] = rankWeightedHybrid(backendResults["sqlite-fts"][query.ID], backendResults["qdrant"][query.ID], normalizeHybridCandidate(candidate))
	}
	return RetrievalBenchmarkFixture{SchemaVersion: "1", FixtureID: "deterministic-passage-query-v1", Queries: queries, BackendResults: backendResults}
}

func RunRetrievalBenchmark(fixture RetrievalBenchmarkFixture, k int) (RetrievalBenchmarkReport, error) {
	if len(fixture.Queries) == 0 {
		return RetrievalBenchmarkReport{}, fmt.Errorf("retrieval benchmark queries are required")
	}
	if k <= 0 {
		k = 10
	}
	fixtureID := fixture.FixtureID
	if fixtureID == "" {
		fixtureID = "custom-deterministic-fixture"
	}
	report := RetrievalBenchmarkReport{SchemaVersion: "1", FixtureID: fixtureID, QuerySetChecksum: checksumHybridQueries(fixture.Queries), QueryCount: len(fixture.Queries), K: k, ReproducibilityNotes: []string{"deterministic passage-query fixtures are embedded in source control", "query-set checksum locks relevance labels and query IDs", "OpenSearch and Qdrant scores are evaluated from fixture outputs unless opt-in integration tests are run"}, PrivacyNotes: []string{"normal benchmark uses synthetic passage IDs and no private full text", "Qdrant fixture represents vector payload privacy without exporting raw embeddings", "OpenSearch fixture avoids live cluster data unless explicitly configured"}}
	backends := make([]string, 0, len(fixture.BackendResults))
	for backend := range fixture.BackendResults {
		backends = append(backends, backend)
	}
	sort.Strings(backends)
	for _, backend := range backends {
		bench, perQuery := evaluateRetrievalBackend(backend, fixture.Queries, fixture.BackendResults[backend], k)
		report.Backends = append(report.Backends, bench)
		report.QueryResults = append(report.QueryResults, perQuery...)
	}
	sort.SliceStable(report.Backends, func(i, j int) bool {
		if report.Backends[i].Score != report.Backends[j].Score {
			return report.Backends[i].Score > report.Backends[j].Score
		}
		return report.Backends[i].Backend < report.Backends[j].Backend
	})
	if len(report.Backends) > 0 {
		report.SelectedBackend = report.Backends[0].Backend
	}
	return report, nil
}

func evaluateRetrievalBackend(backend string, queries []HybridTuningQuery, results map[string][]PassageResult, k int) (RetrievalBackendBenchmark, []RetrievalBenchmarkQueryResult) {
	recallSum := 0.0
	mrrSum := 0.0
	queryResults := []RetrievalBenchmarkQueryResult{}
	for _, query := range queries {
		relevant := map[string]bool{}
		for _, id := range query.RelevantPassageIDs {
			relevant[id] = true
		}
		found := 0
		firstRank := 0
		topK := []string{}
		for i, result := range results[query.ID] {
			if i < k {
				topK = append(topK, result.PassageID)
				if relevant[result.PassageID] {
					found++
				}
			}
			if firstRank == 0 && relevant[result.PassageID] {
				firstRank = i + 1
			}
		}
		recall := 0.0
		if len(relevant) > 0 {
			recall = float64(found) / float64(len(relevant))
		}
		mrr := 0.0
		if firstRank > 0 {
			mrr = 1.0 / float64(firstRank)
		}
		recallSum += recall
		mrrSum += mrr
		queryResults = append(queryResults, RetrievalBenchmarkQueryResult{QueryID: query.ID, Backend: backend, TopK: topK, Metrics: RetrievalBackendBenchmark{Backend: backend, RecallAtK: recall, MRR: mrr, Score: recall + mrr, ReproducibilityNote: backendReproducibilityNote(backend), PrivacyNote: backendPrivacyNote(backend)}})
	}
	count := float64(len(queries))
	bench := RetrievalBackendBenchmark{Backend: backend, ReproducibilityNote: backendReproducibilityNote(backend), PrivacyNote: backendPrivacyNote(backend)}
	if count > 0 {
		bench.RecallAtK = recallSum / count
		bench.MRR = mrrSum / count
	}
	bench.Score = bench.RecallAtK + bench.MRR
	return bench, queryResults
}

func backendReproducibilityNote(backend string) string {
	switch backend {
	case "sqlite-fts":
		return "local SQLite FTS fixture ranking is deterministic"
	case "opensearch":
		return "OpenSearch fixture ranking is replayed from recorded passage IDs"
	case "qdrant":
		return "Qdrant vector fixture uses deterministic recorded nearest-neighbor order"
	case "hybrid":
		return "hybrid ranking combines fixture lexical/vector lists with fixed RRF weights"
	default:
		return "deterministic fixture ranking"
	}
}

func backendPrivacyNote(backend string) string {
	switch backend {
	case "sqlite-fts":
		return "runs locally over project passages"
	case "opensearch":
		return "external cluster use must avoid private payload leakage"
	case "qdrant":
		return "vector payload privacy policy controls raw text/embedding export"
	case "hybrid":
		return "hybrid evaluation inherits lexical and vector backend privacy constraints"
	default:
		return "review backend privacy before external execution"
	}
}
