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
