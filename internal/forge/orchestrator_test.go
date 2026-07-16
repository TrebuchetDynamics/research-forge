package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

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

func TestRunWorkflowDAGRollsBackCheckpointWhenProvenanceAppendFails(t *testing.T) {
	project := t.TempDir()
	provenanceBlocker := filepath.Join(project, "provenance")
	if err := os.WriteFile(provenanceBlocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create provenance blocker: %v", err)
	}
	plan := DefaultWorkflowDAG("catalysts")
	plan.Steps = plan.Steps[:1]
	checkpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))

	run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
	if err == nil {
		t.Fatalf("run = %#v, want provenance append failure", run)
	}
	if _, statErr := os.Stat(checkpoint); !os.IsNotExist(statErr) {
		t.Errorf("failed workflow left checkpoint %s: %v", checkpoint, statErr)
	}

	if err := os.Remove(provenanceBlocker); err != nil {
		t.Fatalf("remove provenance blocker: %v", err)
	}
	retried, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
	if err != nil {
		t.Fatalf("retry workflow: %v", err)
	}
	if len(retried.Checkpoints) != 1 || retried.RestartSafeSkipped != 0 {
		t.Errorf("retry = %#v, want one newly completed checkpoint", retried)
	}
	events, err := ProvenanceEvents(project)
	if err != nil || len(events) != 1 {
		t.Errorf("events = %#v, error = %v, want one retry provenance event", events, err)
	}
}

func TestRunWorkflowDAGRejectsInvalidExistingCheckpointsBeforeWriting(t *testing.T) {
	validCheckpoint := func(step WorkflowStep) WorkflowCheckpoint {
		return WorkflowCheckpoint{StepID: step.ID, Completed: true, Inputs: step.Inputs, Outputs: step.Outputs, Checkpoint: step.Checkpoint, CompletedAt: "2026-01-01T00:00:00Z"}
	}
	tests := []struct {
		name  string
		setup func(t *testing.T, path string, step WorkflowStep)
	}{
		{
			name: "directory",
			setup: func(t *testing.T, path string, _ WorkflowStep) {
				t.Helper()
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatalf("create checkpoint directory: %v", err)
				}
			},
		},
		{
			name: "malformed JSON",
			setup: func(t *testing.T, path string, _ WorkflowStep) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("create checkpoint parent: %v", err)
				}
				if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
					t.Fatalf("write malformed checkpoint: %v", err)
				}
			},
		},
		{
			name: "different step",
			setup: func(t *testing.T, path string, step WorkflowStep) {
				t.Helper()
				checkpoint := validCheckpoint(step)
				checkpoint.StepID = "other"
				if err := writeJSON(path, checkpoint); err != nil {
					t.Fatalf("write mismatched checkpoint: %v", err)
				}
			},
		},
		{
			name: "incomplete",
			setup: func(t *testing.T, path string, step WorkflowStep) {
				t.Helper()
				checkpoint := validCheckpoint(step)
				checkpoint.Completed = false
				if err := writeJSON(path, checkpoint); err != nil {
					t.Fatalf("write incomplete checkpoint: %v", err)
				}
			},
		},
		{
			name: "different path",
			setup: func(t *testing.T, path string, step WorkflowStep) {
				t.Helper()
				checkpoint := validCheckpoint(step)
				checkpoint.Checkpoint = "data/forge-workflow/other.checkpoint.json"
				if err := writeJSON(path, checkpoint); err != nil {
					t.Fatalf("write stale-path checkpoint: %v", err)
				}
			},
		},
		{
			name: "different outputs",
			setup: func(t *testing.T, path string, step WorkflowStep) {
				t.Helper()
				checkpoint := validCheckpoint(step)
				checkpoint.Outputs = []string{"other"}
				if err := writeJSON(path, checkpoint); err != nil {
					t.Fatalf("write stale-output checkpoint: %v", err)
				}
			},
		},
		{
			name: "invalid completion time",
			setup: func(t *testing.T, path string, step WorkflowStep) {
				t.Helper()
				checkpoint := validCheckpoint(step)
				checkpoint.CompletedAt = "not-a-time"
				if err := writeJSON(path, checkpoint); err != nil {
					t.Fatalf("write invalid-time checkpoint: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			plan := DefaultWorkflowDAG("catalysts")
			plan.Steps = plan.Steps[:1]
			checkpointPath := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
			tt.setup(t, checkpointPath, plan.Steps[0])

			run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
			if err == nil || !strings.Contains(err.Error(), "invalid workflow checkpoint") {
				t.Errorf("run = %#v, error = %v, want invalid checkpoint rejection", run, err)
			}
			runPath := filepath.Join(project, "data", "forge-workflow", "run.json")
			if _, statErr := os.Stat(runPath); !os.IsNotExist(statErr) {
				t.Errorf("invalid checkpoint workflow wrote run record %s: %v", runPath, statErr)
			}
		})
	}
}

func TestRunWorkflowDAGRejectsCheckpointWithoutMatchingProvenanceBeforeWriting(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*provenance.Event)
	}{
		{name: "missing event"},
		{name: "wrong timestamp", mutate: func(event *provenance.Event) { event.Timestamp = "2026-01-02T00:00:00Z" }},
		{name: "wrong action", mutate: func(event *provenance.Event) { event.Action = "workflow.other" }},
		{name: "wrong target", mutate: func(event *provenance.Event) { event.Target = "other" }},
		{name: "wrong command", mutate: func(event *provenance.Event) { event.Inputs["command"] = "rforge other" }},
		{name: "wrong checkpoint", mutate: func(event *provenance.Event) { event.Outputs["checkpoint"] = "other.json" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			plan := DefaultWorkflowDAG("catalysts")
			plan.Steps = plan.Steps[:1]
			step := plan.Steps[0]
			checkpoint := WorkflowCheckpoint{StepID: step.ID, Completed: true, Inputs: step.Inputs, Outputs: step.Outputs, Checkpoint: step.Checkpoint, CompletedAt: "2026-01-01T00:00:00Z"}
			checkpointPath := filepath.Join(project, filepath.FromSlash(step.Checkpoint))
			if err := writeJSON(checkpointPath, checkpoint); err != nil {
				t.Fatalf("write checkpoint: %v", err)
			}
			if tt.mutate != nil {
				event := provenance.Event{SchemaVersion: schemaVersion, ID: "evt_test", Timestamp: checkpoint.CompletedAt, Actor: "tester", Action: step.ProvenanceAction, Target: step.ID, Inputs: map[string]any{"command": step.Command, "inputs": step.Inputs}, Outputs: map[string]any{"outputs": step.Outputs, "checkpoint": step.Checkpoint}, Warnings: []string{}}
				tt.mutate(&event)
				if err := provenance.Append(project, event); err != nil {
					t.Fatalf("append mismatched provenance: %v", err)
				}
			}

			run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
			if err == nil || !strings.Contains(err.Error(), "provenance") {
				t.Errorf("run = %#v, error = %v, want checkpoint provenance rejection", run, err)
			}
			runPath := filepath.Join(project, "data", "forge-workflow", "run.json")
			if _, statErr := os.Stat(runPath); !os.IsNotExist(statErr) {
				t.Errorf("unproven checkpoint workflow wrote run record %s: %v", runPath, statErr)
			}
		})
	}
}

func TestRunWorkflowDAGRejectsUnsupportedSchemaWithoutSideEffects(t *testing.T) {
	project := t.TempDir()
	plan := DefaultWorkflowDAG("catalysts")
	plan.SchemaVersion = "999"

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 1, Actor: "tester"})
	if err == nil || !strings.Contains(err.Error(), `unsupported workflow schema version "999"`) {
		t.Errorf("run error = %v, want unsupported schema rejection", err)
	}
	checkpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
	if _, statErr := os.Stat(checkpoint); !os.IsNotExist(statErr) {
		t.Errorf("unsupported workflow wrote checkpoint %s: %v", checkpoint, statErr)
	}
}

func TestRunWorkflowDAGRejectsCheckpointTraversalWithoutOutsideWrite(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	plan := DefaultWorkflowDAG("catalysts")
	plan.Steps[1].Checkpoint = "../escaped-checkpoint.json"

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 2, Actor: "tester"})
	if err == nil || !strings.Contains(err.Error(), `unsafe workflow checkpoint path "../escaped-checkpoint.json"`) {
		t.Errorf("run error = %v, want unsafe checkpoint rejection", err)
	}
	firstCheckpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
	if _, statErr := os.Stat(firstCheckpoint); !os.IsNotExist(statErr) {
		t.Errorf("unsafe workflow partially wrote first checkpoint %s: %v", firstCheckpoint, statErr)
	}
	escaped := filepath.Join(parent, "escaped-checkpoint.json")
	if _, statErr := os.Stat(escaped); !os.IsNotExist(statErr) {
		t.Errorf("unsafe workflow wrote outside project at %s: %v", escaped, statErr)
	}
}

func TestRunWorkflowDAGRejectsSymlinkedCheckpointParentWithoutOutsideWrite(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "project")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatalf("create project data: %v", err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatalf("create outside directory: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "data", "forge-workflow")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	plan := DefaultWorkflowDAG("catalysts")

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 1, Actor: "tester"})
	if err == nil || !strings.Contains(err.Error(), `unsafe workflow checkpoint path "data/forge-workflow/discovery.checkpoint.json"`) {
		t.Errorf("run error = %v, want symlinked checkpoint rejection", err)
	}
	escaped := filepath.Join(outside, "discovery.checkpoint.json")
	if _, statErr := os.Stat(escaped); !os.IsNotExist(statErr) {
		t.Errorf("unsafe workflow wrote through symlink at %s: %v", escaped, statErr)
	}
}

func TestRunWorkflowDAGDoesNotWriteThroughSymlinkedRunSummary(t *testing.T) {
	project := t.TempDir()
	workflowDir := filepath.Join(project, "data", "forge-workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("create workflow directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-run.json")
	outsideBefore := []byte("outside workflow run\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside run summary: %v", err)
	}
	runPath := filepath.Join(workflowDir, "run.json")
	if err := os.Symlink(outsidePath, runPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	plan := DefaultWorkflowDAG("catalysts")

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 1, Actor: "tester"})
	if err == nil {
		t.Errorf("workflow succeeded with symlinked run summary")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside run summary: %v", readErr)
	}
	if string(outsideAfter) != string(outsideBefore) {
		t.Errorf("workflow wrote through run summary symlink:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
	}
	info, statErr := os.Stat(outsidePath)
	if statErr != nil {
		t.Fatalf("stat outside run summary: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("outside run summary mode = %o, want 600", got)
	}
}

func TestRunWorkflowDAGRejectsInvalidDependencyBeforeWriting(t *testing.T) {
	project := t.TempDir()
	plan := DefaultWorkflowDAG("catalysts")
	plan.Steps[1].DependsOn = []string{"missing"}

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 2, Actor: "tester"})
	want := "dependency missing for step " + plan.Steps[1].ID + " is not complete"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("run error = %v, want %q", err, want)
	}
	firstCheckpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
	if _, statErr := os.Stat(firstCheckpoint); !os.IsNotExist(statErr) {
		t.Errorf("invalid workflow partially wrote first checkpoint %s: %v", firstCheckpoint, statErr)
	}
}

func TestRunWorkflowDAGRejectsDuplicateCheckpointPathsBeforeWriting(t *testing.T) {
	project := t.TempDir()
	plan := DefaultWorkflowDAG("catalysts")
	plan.Steps[1].Checkpoint = "data/forge-workflow/./discovery.checkpoint.json"

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{MaxSteps: 2, Actor: "tester"})
	if err == nil || !strings.Contains(err.Error(), "duplicate workflow checkpoint path") || !strings.Contains(err.Error(), plan.Steps[0].ID) || !strings.Contains(err.Error(), plan.Steps[1].ID) {
		t.Errorf("run error = %v, want duplicate checkpoint rejection for %s and %s", err, plan.Steps[0].ID, plan.Steps[1].ID)
	}
	firstCheckpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
	if _, statErr := os.Stat(firstCheckpoint); !os.IsNotExist(statErr) {
		t.Errorf("duplicate workflow wrote first checkpoint %s: %v", firstCheckpoint, statErr)
	}
}

func TestRunWorkflowDAGRejectsDuplicateStepIDsBeforeWriting(t *testing.T) {
	project := t.TempDir()
	plan := DefaultWorkflowDAG("catalysts")
	plan.Steps = plan.Steps[:2]
	plan.Steps[1].ID = plan.Steps[0].ID
	plan.Steps[1].DependsOn = []string{plan.Steps[0].ID}

	_, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
	if err == nil || !strings.Contains(err.Error(), `duplicate workflow step ID "discovery"`) {
		t.Errorf("run error = %v, want duplicate step ID rejection", err)
	}
	firstCheckpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
	if _, statErr := os.Stat(firstCheckpoint); !os.IsNotExist(statErr) {
		t.Errorf("duplicate-ID workflow wrote first checkpoint %s: %v", firstCheckpoint, statErr)
	}
}

func TestRunWorkflowDAGRejectsBlankStepIDBeforeWriting(t *testing.T) {
	for _, id := range []string{"", " \t"} {
		t.Run(id, func(t *testing.T) {
			project := t.TempDir()
			plan := DefaultWorkflowDAG("catalysts")
			plan.Steps = plan.Steps[:1]
			plan.Steps[0].ID = id

			run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
			if err == nil || !strings.Contains(err.Error(), "workflow step ID is required") {
				t.Errorf("run = %#v, error = %v, want blank step ID rejection", run, err)
			}
			checkpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
			if _, statErr := os.Stat(checkpoint); !os.IsNotExist(statErr) {
				t.Errorf("blank-ID workflow wrote checkpoint %s: %v", checkpoint, statErr)
			}
		})
	}
}

func TestRunWorkflowDAGRejectsBlankRequiredStepMetadataBeforeWriting(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*WorkflowStep)
		want   string
	}{
		{name: "empty command", mutate: func(step *WorkflowStep) { step.Command = "" }, want: "command is required"},
		{name: "whitespace command", mutate: func(step *WorkflowStep) { step.Command = " \t" }, want: "command is required"},
		{name: "empty provenance action", mutate: func(step *WorkflowStep) { step.ProvenanceAction = "" }, want: "provenance action is required"},
		{name: "whitespace provenance action", mutate: func(step *WorkflowStep) { step.ProvenanceAction = " \t" }, want: "provenance action is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			plan := DefaultWorkflowDAG("catalysts")
			plan.Steps = plan.Steps[:1]
			tt.mutate(&plan.Steps[0])

			run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("run = %#v, error = %v, want %q", run, err, tt.want)
			}
			checkpoint := filepath.Join(project, filepath.FromSlash(plan.Steps[0].Checkpoint))
			if _, statErr := os.Stat(checkpoint); !os.IsNotExist(statErr) {
				t.Errorf("blank-metadata workflow wrote checkpoint %s: %v", checkpoint, statErr)
			}
		})
	}
}

func TestRunWorkflowDAGRejectsEmptyCheckpointBeforeWriting(t *testing.T) {
	project := t.TempDir()
	plan := DefaultWorkflowDAG("catalysts")
	plan.Steps = plan.Steps[:1]
	plan.Steps[0].Checkpoint = ""

	run, err := RunWorkflowDAG(project, plan, RunWorkflowOptions{Actor: "tester"})
	if err == nil || !strings.Contains(err.Error(), `unsafe workflow checkpoint path ""`) {
		t.Errorf("run = %#v, error = %v, want empty checkpoint rejection", run, err)
	}
	runPath := filepath.Join(project, "data", "forge-workflow", "run.json")
	if _, statErr := os.Stat(runPath); !os.IsNotExist(statErr) {
		t.Errorf("empty-checkpoint workflow wrote run record %s: %v", runPath, statErr)
	}
}
