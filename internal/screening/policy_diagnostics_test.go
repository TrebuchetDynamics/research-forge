package screening

import "testing"

func TestPrioritizeBalancedExplorationExploitationRecords(t *testing.T) {
	records := []ScreeningRecord{{ID: "in", Title: "solar catalyst"}, {ID: "out", Title: "battery storage"}, {ID: "exploit", Title: "solar fuel catalyst"}, {ID: "explore", Title: "unknown frontier"}}
	events := []DecisionEvent{{PaperID: "in", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "r1"}, {PaperID: "out", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "r1"}}
	ranked := PrioritizeBalancedRecords(records, events, StageTitleAbstract, BalancedPolicy{ExploitationWeight: 0.7, ExplorationWeight: 0.3})
	if len(ranked) != 2 {
		t.Fatalf("ranked = %#v", ranked)
	}
	if ranked[0].ID != "exploit" || ranked[0].Policy != "balanced" || ranked[0].ExploitationScore == 0 || ranked[0].ExplorationScore == 0 {
		t.Fatalf("top ranked = %#v", ranked[0])
	}
}

func TestSimulateRecallEffortUsesRankedOutputAndRelevantSet(t *testing.T) {
	ranked := []PrioritizedRecord{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}}
	sim := SimulateRecallEffort(ranked, []string{"p2", "p3"}, []float64{0.5, 1.0})
	if sim.TotalRelevant != 2 || len(sim.Points) != 3 || len(sim.Targets) != 2 {
		t.Fatalf("simulation = %#v", sim)
	}
	if sim.Points[1].Recall != 0.5 || sim.Targets[0].ReachedAt != 2 || sim.Targets[1].ReachedAt != 3 {
		t.Fatalf("simulation details = %#v", sim)
	}
}

func TestActiveLearningSensitivityDiagnosticsComparePolicies(t *testing.T) {
	records := []ScreeningRecord{{ID: "in", Title: "solar catalyst"}, {ID: "out", Title: "battery storage"}, {ID: "candidate", Title: "solar fuel"}, {ID: "frontier", Title: "frontier method"}}
	events := []DecisionEvent{{PaperID: "in", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "r1"}, {PaperID: "out", Stage: StageTitleAbstract, Decision: DecisionExclude, Reason: "off-topic", Reviewer: "r1"}}
	diag, err := ActiveLearningSensitivityDiagnostics(ActiveLearningSensitivityInput{Records: records, Events: events, Stage: StageTitleAbstract, RelevantPaperIDs: []string{"candidate"}, Policies: []BalancedPolicy{{Name: "exploit", ExploitationWeight: 1}, {Name: "explore", ExplorationWeight: 1}}, TargetRecalls: []float64{1.0}})
	if err != nil {
		t.Fatalf("ActiveLearningSensitivityDiagnostics returned error: %v", err)
	}
	if diag.SchemaVersion != "1" || diag.InputHash == "" || len(diag.PolicyResults) != 2 || diag.SelectedPolicy == "" {
		t.Fatalf("diag = %#v", diag)
	}
	if diag.PolicyResults[0].Simulation.TotalRelevant != 1 || diag.PolicyResults[0].RankedCount == 0 {
		t.Fatalf("policy result = %#v", diag.PolicyResults[0])
	}
}
