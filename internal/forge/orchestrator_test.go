package forge

import "testing"

func TestWorkflowOrchestratorBuildsResumableDAGCheckpointsAndRestartSafety(t *testing.T) {
	project := t.TempDir()
	plan := DefaultWorkflowDAG("catalysts")
	if len(plan.Steps) != 10 || plan.Steps[0].ID != "discovery" || plan.Steps[len(plan.Steps)-1].ID != "report" {
		t.Fatalf("plan = %#v", plan.Steps)
	}
	run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 3, Actor: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Checkpoints) != 3 || !run.Checkpoints[2].Completed {
		t.Fatalf("run = %#v", run)
	}
	resumed, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resumed.Checkpoints) != len(plan.Steps) {
		t.Fatalf("resumed = %#v", resumed)
	}
	again, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if again.RestartSafeSkipped != len(plan.Steps) {
		t.Fatalf("restart safety = %#v", again)
	}
	events, err := ProvenanceEvents(project)
	if err != nil || len(events) == 0 {
		t.Fatalf("events=%#v err=%v", events, err)
	}
}
