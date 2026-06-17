package screening

import "testing"

func TestScreeningAuditBundleContainsASReviewExportFields(t *testing.T) {
	records := []ScreeningRecord{{ID: "p1", Title: "Seed include"}, {ID: "p2", Title: "Candidate"}}
	events := []DecisionEvent{{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"}}
	run, err := BuildActiveLearningRun(ActiveLearningRunInput{RunID: "run-a", Records: records, Events: events, Stage: StageTitleAbstract, RankingMethod: "asreview", TargetRecall: 0.95})
	if err != nil {
		t.Fatal(err)
	}
	bundle := BuildScreeningAuditBundle(ScreeningAuditBundleInput{Records: records, Events: events, Stage: StageTitleAbstract, ActiveRun: run})
	if len(bundle.FrozenDataset) != 2 {
		t.Fatalf("missing frozen dataset: %#v", bundle.FrozenDataset)
	}
	if len(bundle.SeedLabels) != 1 || bundle.SeedLabels[0].PaperID != "p1" {
		t.Fatalf("seed labels = %#v", bundle.SeedLabels)
	}
	if len(bundle.RankingIterations) == 0 || bundle.RankingIterations[0].ID == "" {
		t.Fatalf("ranking iterations = %#v", bundle.RankingIterations)
	}
	if len(bundle.ReviewerActions) != 1 {
		t.Fatalf("reviewer actions = %#v", bundle.ReviewerActions)
	}
	if bundle.StoppingDiagnostics.TargetRecall != 0.95 {
		t.Fatalf("stopping = %#v", bundle.StoppingDiagnostics)
	}
	if len(bundle.RandomSeeds) == 0 || bundle.ModelMetadata.Name == "" || bundle.ModelMetadata.Version == "" {
		t.Fatalf("metadata/seeds missing: %#v", bundle)
	}
}
