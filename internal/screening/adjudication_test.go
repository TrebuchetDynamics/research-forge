package screening

import "testing"

func TestAdjudicatedDecisionResolvesConflict(t *testing.T) {
	workflow, err := Configure(Options{ExclusionReasons: []string{"off-topic"}})
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}
	store := NewMemoryStore(workflow)
	_ = store.Decide(DecisionInput{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"})
	_ = store.Decide(DecisionInput{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "bob"})
	if conflicts := store.Conflicts(StageTitleAbstract); len(conflicts) != 1 || conflicts[0] != "paper-1" {
		t.Fatalf("conflicts before adjudication = %#v", conflicts)
	}
	_ = store.Decide(DecisionInput{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "carol", Adjudicated: true})
	if conflicts := store.Conflicts(StageTitleAbstract); len(conflicts) != 0 {
		t.Fatalf("conflicts after adjudication = %#v", conflicts)
	}
}
