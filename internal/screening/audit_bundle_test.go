package screening

import "testing"

func TestBuildReviewerAssignmentsConflictPanelUncertainQueueAndAuditBundle(t *testing.T) {
	records := []ScreeningRecord{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}}
	assignments := AssignReviewers(records, []string{"ada", "bob"}, 2)
	if len(assignments) != 6 || assignments[0].Reviewer != "ada" || assignments[1].Reviewer != "bob" || assignments[2].PaperID != "p2" {
		t.Fatalf("assignments = %#v", assignments)
	}
	events := []DecisionEvent{
		{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"},
		{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "bob"},
		{PaperID: "p2", Stage: StageTitleAbstract, Decision: DecisionUncertain, Reviewer: "ada"},
		{PaperID: "p3", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "carol", Adjudicated: true},
	}
	panel := BuildConflictAdjudicationPanel(events, StageTitleAbstract)
	if len(panel.Conflicts) != 1 || panel.Conflicts[0].PaperID != "p1" || len(panel.Adjudicated) != 1 {
		t.Fatalf("panel = %#v", panel)
	}
	uncertain := UncertainQueue(events, StageTitleAbstract)
	if len(uncertain) != 1 || uncertain[0].PaperID != "p2" {
		t.Fatalf("uncertain = %#v", uncertain)
	}
	bundle := BuildScreeningAuditBundle(ScreeningAuditBundleInput{Records: records, Events: events, Assignments: assignments, Stage: StageTitleAbstract, ActiveRun: ActiveLearningRun{RunID: "run-1"}})
	if bundle.SchemaVersion != "1" || bundle.InputHash == "" || bundle.DecisionHash == "" || len(bundle.Assignments) != 6 || bundle.Panel.Conflicts[0].PaperID != "p1" || bundle.ActiveRunRef != "run-1" {
		t.Fatalf("bundle = %#v", bundle)
	}
}
