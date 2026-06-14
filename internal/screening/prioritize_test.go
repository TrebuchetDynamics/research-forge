package screening

import "testing"

func TestPrioritizeActiveLearningRecordsRanksUnscreenedByReviewerSignals(t *testing.T) {
	records := []ScreeningRecord{
		{ID: "seed-include", Title: "LightGBM leakage detection for crypto order books", Abstract: "microstructure forecasting"},
		{ID: "seed-exclude", Title: "Plant photosynthesis catalyst review", Abstract: "materials chemistry"},
		{ID: "candidate-relevant", Title: "Crypto order book leakage detection", Abstract: "LightGBM microstructure forecasting"},
		{ID: "candidate-irrelevant", Title: "Artificial photosynthesis catalyst", Abstract: "materials review"},
		{ID: "candidate-neutral", Title: "Forecast evaluation", Abstract: "time series validation"},
	}
	events := []DecisionEvent{
		{PaperID: "seed-include", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"},
		{PaperID: "seed-exclude", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "ada"},
	}

	prioritized := PrioritizeActiveLearningRecords(records, events, StageTitleAbstract)

	if len(prioritized) != 3 {
		t.Fatalf("prioritized length = %d, want 3: %#v", len(prioritized), prioritized)
	}
	if prioritized[0].ID != "candidate-relevant" {
		t.Fatalf("first priority = %#v, want candidate-relevant", prioritized[0])
	}
	if prioritized[len(prioritized)-1].ID != "candidate-irrelevant" {
		t.Fatalf("last priority = %#v, want candidate-irrelevant", prioritized[len(prioritized)-1])
	}
	if prioritized[0].Score <= prioritized[1].Score || prioritized[1].Score <= prioritized[2].Score {
		t.Fatalf("scores not descending: %#v", prioritized)
	}
}

func TestPrioritizeModelRecordsRanksBySmoothedReviewerModel(t *testing.T) {
	records := []ScreeningRecord{
		{ID: "seed-include", Title: "crypto leakage microstructure"},
		{ID: "seed-exclude", Title: "plant catalyst photosynthesis"},
		{ID: "candidate-relevant", Title: "crypto microstructure signals"},
		{ID: "candidate-irrelevant", Title: "plant photosynthesis materials"},
		{ID: "candidate-boundary", Title: "validation methods"},
	}
	events := []DecisionEvent{
		{PaperID: "seed-include", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"},
		{PaperID: "seed-exclude", Stage: StageTitleAbstract, Decision: DecisionExclude, Reviewer: "ada"},
	}

	prioritized := PrioritizeModelRecords(records, events, StageTitleAbstract)

	if len(prioritized) != 3 {
		t.Fatalf("prioritized length = %d, want 3: %#v", len(prioritized), prioritized)
	}
	if prioritized[0].ID != "candidate-relevant" || prioritized[0].Score <= 0.5 {
		t.Fatalf("top model priority = %#v, want relevant probability", prioritized[0])
	}
	if prioritized[len(prioritized)-1].ID != "candidate-irrelevant" || prioritized[len(prioritized)-1].Score >= 0.5 {
		t.Fatalf("last model priority = %#v, want irrelevant probability", prioritized[len(prioritized)-1])
	}
}

func TestPrioritizeUncertaintyRecordsRanksBoundaryCasesFirst(t *testing.T) {
	records := []ScreeningRecord{
		{ID: "seed-include", Title: "crypto leakage"},
		{ID: "seed-exclude", Title: "plant catalyst"},
		{ID: "candidate-positive", Title: "crypto leakage"},
		{ID: "candidate-negative", Title: "plant catalyst"},
		{ID: "candidate-boundary", Title: "unseen validation"},
	}
	events := []DecisionEvent{
		{PaperID: "seed-include", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"},
		{PaperID: "seed-exclude", Stage: StageTitleAbstract, Decision: DecisionExclude, Reviewer: "ada"},
	}

	prioritized := PrioritizeUncertaintyRecords(records, events, StageTitleAbstract)

	if len(prioritized) != 3 {
		t.Fatalf("prioritized length = %d, want 3: %#v", len(prioritized), prioritized)
	}
	if prioritized[0].ID != "candidate-boundary" || prioritized[0].Uncertainty != 1 {
		t.Fatalf("first uncertainty priority = %#v, want boundary", prioritized[0])
	}
	if prioritized[1].Uncertainty >= prioritized[0].Uncertainty || prioritized[2].Uncertainty >= prioritized[0].Uncertainty {
		t.Fatalf("uncertainty scores not ranked: %#v", prioritized)
	}
}
