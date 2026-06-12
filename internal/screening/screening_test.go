package screening

import "testing"

func TestConfigureWorkflowValidatesReasonsStagesAndDecisions(t *testing.T) {
	workflow, err := Configure(Options{ExclusionReasons: []string{"wrong population", "not empirical"}})
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}
	if len(workflow.Stages) != 3 || workflow.Stages[0] != StageTitleAbstract || workflow.Decisions[2] != DecisionUncertain {
		t.Fatalf("workflow = %#v", workflow)
	}
	if err := workflow.ValidateReason("wrong population"); err != nil {
		t.Fatalf("ValidateReason returned error: %v", err)
	}
	if err := workflow.ValidateReason("unknown"); err == nil {
		t.Fatalf("ValidateReason returned nil error for unknown reason")
	}
}

func TestDecisionHistoryReviewerAttributionQueuesConflictsAndPrismaCounts(t *testing.T) {
	workflow, _ := Configure(Options{ExclusionReasons: []string{"wrong population"}})
	store := NewMemoryStore(workflow)
	if err := store.Decide(DecisionInput{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"}); err != nil {
		t.Fatalf("Decide include returned error: %v", err)
	}
	if err := store.Decide(DecisionInput{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "wrong population", Reviewer: "grace"}); err != nil {
		t.Fatalf("Decide exclude returned error: %v", err)
	}
	if err := store.Decide(DecisionInput{PaperID: "paper-2", Stage: StageFullText, Decision: DecisionUncertain, Reviewer: "ada"}); err != nil {
		t.Fatalf("Decide uncertain returned error: %v", err)
	}
	if len(store.History("paper-1")) != 2 || store.History("paper-1")[0].Reviewer != "ada" {
		t.Fatalf("history = %#v", store.History("paper-1"))
	}
	conflicts := store.Conflicts(StageTitleAbstract)
	if len(conflicts) != 1 || conflicts[0] != "paper-1" {
		t.Fatalf("conflicts = %#v", conflicts)
	}
	uncertain := store.Queue(QueueFilter{Stage: StageFullText, Decision: DecisionUncertain})
	if len(uncertain) != 1 || uncertain[0] != "paper-2" {
		t.Fatalf("uncertain = %#v", uncertain)
	}
	counts := store.PRISMACounts()
	if counts.Included != 1 || counts.Excluded != 1 || counts.Uncertain != 1 {
		t.Fatalf("counts = %#v", counts)
	}
}
