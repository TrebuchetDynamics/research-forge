package screening

import "testing"

func TestBuildActiveLearningRunPersistsASReviewStyleState(t *testing.T) {
	records := []ScreeningRecord{
		{ID: "seed-in", Title: "Solar catalyst", Abstract: "water splitting"},
		{ID: "seed-out", Title: "Excluded battery", Abstract: "storage"},
		{ID: "candidate", Title: "Solar water catalyst", Abstract: "fuel"},
	}
	events := []DecisionEvent{
		{PaperID: "seed-in", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "r1"},
		{PaperID: "seed-out", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "r2"},
		{PaperID: "conflict", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "r1"},
		{PaperID: "conflict", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "r2"},
		{PaperID: "resolved", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "r1", Adjudicated: true},
	}
	run, err := BuildActiveLearningRun(ActiveLearningRunInput{RunID: "run-1", Records: records, Events: events, Stage: StageTitleAbstract, RankingMethod: "active-learning", TargetRecall: 0.8})
	if err != nil {
		t.Fatalf("BuildActiveLearningRun returned error: %v", err)
	}
	if run.SchemaVersion != "1" || run.RunID != "run-1" || run.InputHash == "" || run.DecisionHash == "" {
		t.Fatalf("run metadata = %#v", run)
	}
	if len(run.SeedDecisions) < 2 || len(run.RankedOutput) != 1 || run.RankedOutput[0].ID != "candidate" {
		t.Fatalf("seed/ranked = %#v %#v", run.SeedDecisions, run.RankedOutput)
	}
	if run.RankingMethod != "active-learning" || run.ReviewerProgress.ScreenedRecords == 0 || run.StoppingDiagnostics.TargetRecall != 0.8 {
		t.Fatalf("diagnostics = %#v", run)
	}
	if run.AdjudicationState.Conflicts == 0 || run.AdjudicationState.Adjudicated == 0 || len(run.AdjudicationState.ConflictPaperIDs) == 0 {
		t.Fatalf("adjudication = %#v", run.AdjudicationState)
	}
}

func TestBuildActiveLearningRunSupportsModelAndUncertaintyMethods(t *testing.T) {
	records := []ScreeningRecord{{ID: "in", Title: "solar catalyst"}, {ID: "out", Title: "battery"}, {ID: "candidate", Title: "solar fuel"}}
	events := []DecisionEvent{{PaperID: "in", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "r1"}, {PaperID: "out", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "r1"}}
	for _, method := range []string{"model", "uncertainty"} {
		run, err := BuildActiveLearningRun(ActiveLearningRunInput{Records: records, Events: events, Stage: StageTitleAbstract, RankingMethod: method})
		if err != nil {
			t.Fatalf("method %s: %v", method, err)
		}
		if run.RankingMethod != method || len(run.RankedOutput) != 1 {
			t.Fatalf("run = %#v", run)
		}
	}
}
