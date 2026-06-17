package retrieval

import "testing"

func TestCalibrateHybridRetrievalSelectsWeightedConfigAndRecordsChecksum(t *testing.T) {
	queries := []HybridTuningQuery{{ID: "q1", Query: "solar catalyst", RelevantPassageIDs: []string{"p1"}}, {ID: "q2", Query: "water splitting", RelevantPassageIDs: []string{"p3"}}}
	lexical := map[string][]PassageResult{
		"q1": {{PassageID: "p2"}, {PassageID: "p1"}},
		"q2": {{PassageID: "p3"}},
	}
	vector := map[string][]PassageResult{
		"q1": {{PassageID: "p1"}},
		"q2": {{PassageID: "p4"}, {PassageID: "p3"}},
	}
	candidates := []HybridTuningCandidate{
		{Name: "lexical-heavy", LexicalWeight: 2, VectorWeight: 0.5, BackendWeights: map[string]float64{"sqlite-fts5": 2, "qdrant": 0.5}},
		{Name: "balanced", LexicalWeight: 1, VectorWeight: 1, BackendWeights: map[string]float64{"sqlite-fts5": 1, "qdrant": 1}},
	}
	file, err := CalibrateHybridRetrieval(queries, lexical, vector, candidates, 1)
	if err != nil {
		t.Fatalf("CalibrateHybridRetrieval returned error: %v", err)
	}
	if file.SchemaVersion != "1" || file.QuerySetChecksum == "" || file.QueryCount != 2 || len(file.Evaluations) != 2 {
		t.Fatalf("file = %#v", file)
	}
	if file.SelectedConfiguration.Name != "balanced" || file.SelectedConfiguration.LexicalWeight != 1 || file.SelectedConfiguration.BackendWeights["qdrant"] != 1 {
		t.Fatalf("selected = %#v", file.SelectedConfiguration)
	}
	if file.Evaluations[0].RecallAtK == 0 && file.Evaluations[1].RecallAtK == 0 {
		t.Fatalf("evaluations = %#v", file.Evaluations)
	}
}

func TestCalibrateHybridRetrievalRequiresQuerySet(t *testing.T) {
	_, err := CalibrateHybridRetrieval(nil, nil, nil, nil, 0)
	if err == nil {
		t.Fatalf("expected error")
	}
}
