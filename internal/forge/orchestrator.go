package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/security"
)

type WorkflowDAG struct {
	SchemaVersion string         `json:"schemaVersion"`
	Question      string         `json:"question"`
	Steps         []WorkflowStep `json:"steps"`
}

type WorkflowStep struct {
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	DependsOn        []string `json:"dependsOn,omitempty"`
	Command          string   `json:"command"`
	Inputs           []string `json:"inputs,omitempty"`
	Outputs          []string `json:"outputs,omitempty"`
	Checkpoint       string   `json:"checkpoint"`
	ProvenanceAction string   `json:"provenanceAction"`
}

type WorkflowRun struct {
	SchemaVersion      string               `json:"schemaVersion"`
	ProjectPath        string               `json:"projectPath"`
	Checkpoints        []WorkflowCheckpoint `json:"checkpoints"`
	RestartSafeSkipped int                  `json:"restartSafeSkipped"`
	UpdatedAt          string               `json:"updatedAt"`
}

type WorkflowCheckpoint struct {
	StepID      string   `json:"stepId"`
	Completed   bool     `json:"completed"`
	Inputs      []string `json:"inputs,omitempty"`
	Outputs     []string `json:"outputs,omitempty"`
	Checkpoint  string   `json:"checkpoint"`
	CompletedAt string   `json:"completedAt"`
}

type RunWorkflowOptions struct {
	MaxSteps int
	Actor    string
}

func DefaultWorkflowDAG(question string) WorkflowDAG {
	ids := []string{"discovery", "import", "dedupe", "legal_full_text_fetch", "parse", "index", "screen", "extract", "analyze", "report"}
	labels := []string{"Discovery", "Import", "Dedupe", "Legal full-text fetch", "Parse", "Index", "Screen", "Extract", "Analyze", "Report"}
	commands := []string{"rforge search import", "rforge library import", "rforge duplicate report", "rforge oa acquisition-queue", "rforge parse", "rforge index rebuild", "rforge screen active-run", "rforge extract add", "rforge analysis run", "rforge report build"}
	steps := []WorkflowStep{}
	for i, id := range ids {
		deps := []string{}
		if i > 0 {
			deps = []string{ids[i-1]}
		}
		steps = append(steps, WorkflowStep{ID: id, Label: labels[i], DependsOn: deps, Command: commands[i], Inputs: []string{"project"}, Outputs: []string{"data/forge-workflow/" + id + ".checkpoint.json"}, Checkpoint: "data/forge-workflow/" + id + ".checkpoint.json", ProvenanceAction: "workflow." + id})
	}
	return WorkflowDAG{SchemaVersion: schemaVersion, Question: question, Steps: steps}
}

func RunWorkflowDAG(projectPath string, dag WorkflowDAG, opts RunWorkflowOptions) (WorkflowRun, error) {
	if projectPath == "" {
		return WorkflowRun{}, fmt.Errorf("project path is required")
	}
	if err := validateWorkflowDAG(projectPath, dag); err != nil {
		return WorkflowRun{}, err
	}
	existingCheckpoints, err := loadExistingWorkflowCheckpoints(projectPath, dag)
	if err != nil {
		return WorkflowRun{}, err
	}
	run := WorkflowRun{SchemaVersion: schemaVersion, ProjectPath: projectPath, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	completed := map[string]bool{}
	for _, step := range dag.Steps {
		if checkpoint, exists := existingCheckpoints[step.ID]; exists {
			completed[step.ID] = true
			run.RestartSafeSkipped++
			run.Checkpoints = append(run.Checkpoints, checkpoint)
			continue
		}
		for _, dep := range step.DependsOn {
			if !completed[dep] {
				return run, fmt.Errorf("dependency %s for step %s is not complete", dep, step.ID)
			}
		}
		if opts.MaxSteps > 0 && len(run.Checkpoints)-run.RestartSafeSkipped >= opts.MaxSteps {
			break
		}
		completedAt := time.Now().UTC()
		cp := WorkflowCheckpoint{StepID: step.ID, Completed: true, Inputs: step.Inputs, Outputs: step.Outputs, Checkpoint: step.Checkpoint, CompletedAt: completedAt.Format(time.RFC3339)}
		checkpointPath := filepath.Join(projectPath, filepath.FromSlash(step.Checkpoint))
		if err := writeJSON(checkpointPath, cp); err != nil {
			return run, err
		}
		if err := provenance.Append(projectPath, provenance.Event{SchemaVersion: schemaVersion, ID: "evt_" + completedAt.Format("20060102T150405Z") + "_workflow_" + step.ID, Timestamp: cp.CompletedAt, Actor: actor(opts.Actor), Action: step.ProvenanceAction, Target: step.ID, Inputs: map[string]any{"command": step.Command, "inputs": step.Inputs}, Outputs: map[string]any{"outputs": step.Outputs, "checkpoint": step.Checkpoint}, Warnings: []string{}}); err != nil {
			if removeErr := os.Remove(checkpointPath); removeErr != nil {
				return run, fmt.Errorf("append provenance: %v; roll back checkpoint: %w", err, removeErr)
			}
			return run, err
		}
		completed[step.ID] = true
		run.Checkpoints = append(run.Checkpoints, cp)
	}
	if err := writeJSON(filepath.Join(projectPath, "data", "forge-workflow", "run.json"), run); err != nil {
		return run, err
	}
	return run, nil
}

func loadExistingWorkflowCheckpoints(projectPath string, dag WorkflowDAG) (map[string]WorkflowCheckpoint, error) {
	existing := make(map[string]WorkflowCheckpoint, len(dag.Steps))
	for _, step := range dag.Steps {
		data, err := os.ReadFile(filepath.Join(projectPath, filepath.FromSlash(step.Checkpoint)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: %w", step.Checkpoint, err)
		}
		var checkpoint WorkflowCheckpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: %w", step.Checkpoint, err)
		}
		if checkpoint.StepID != step.ID {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: step ID %q does not match %q", step.Checkpoint, checkpoint.StepID, step.ID)
		}
		if !checkpoint.Completed {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: step is not complete", step.Checkpoint)
		}
		if checkpoint.Checkpoint != step.Checkpoint {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: recorded path %q does not match", step.Checkpoint, checkpoint.Checkpoint)
		}
		if !slices.Equal(checkpoint.Inputs, step.Inputs) || !slices.Equal(checkpoint.Outputs, step.Outputs) {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: inputs or outputs do not match step %q", step.Checkpoint, step.ID)
		}
		if _, err := time.Parse(time.RFC3339, checkpoint.CompletedAt); err != nil {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: invalid completion time: %w", step.Checkpoint, err)
		}
		existing[step.ID] = checkpoint
	}
	if len(existing) == 0 {
		return existing, nil
	}
	events, err := provenance.Read(projectPath)
	if err != nil {
		return nil, fmt.Errorf("validate workflow checkpoint provenance: %w", err)
	}
	for _, step := range dag.Steps {
		checkpoint, exists := existing[step.ID]
		if exists && !hasMatchingWorkflowProvenance(events, step, checkpoint) {
			return nil, fmt.Errorf("invalid workflow checkpoint %q: no matching provenance event", step.Checkpoint)
		}
	}
	return existing, nil
}

func hasMatchingWorkflowProvenance(events []provenance.Event, step WorkflowStep, checkpoint WorkflowCheckpoint) bool {
	for _, event := range events {
		if event.SchemaVersion == schemaVersion && event.Timestamp == checkpoint.CompletedAt && event.Action == step.ProvenanceAction && event.Target == step.ID && event.Inputs["command"] == step.Command && event.Outputs["checkpoint"] == step.Checkpoint {
			return true
		}
	}
	return false
}

func validateWorkflowDAG(projectPath string, dag WorkflowDAG) error {
	if dag.SchemaVersion != schemaVersion {
		return fmt.Errorf("unsupported workflow schema version %q", dag.SchemaVersion)
	}
	declared := map[string]bool{}
	checkpointOwners := map[string]string{}
	for _, step := range dag.Steps {
		if strings.TrimSpace(step.ID) == "" {
			return fmt.Errorf("workflow step ID is required")
		}
		if declared[step.ID] {
			return fmt.Errorf("duplicate workflow step ID %q", step.ID)
		}
		if strings.TrimSpace(step.Command) == "" {
			return fmt.Errorf("workflow step %q command is required", step.ID)
		}
		if strings.TrimSpace(step.ProvenanceAction) == "" {
			return fmt.Errorf("workflow step %q provenance action is required", step.ID)
		}
		if err := security.ValidatePathWithinRoot(projectPath, step.Checkpoint); err != nil {
			return fmt.Errorf("unsafe workflow checkpoint path %q: %w", step.Checkpoint, err)
		}
		checkpoint := filepath.Clean(filepath.FromSlash(step.Checkpoint))
		if owner, exists := checkpointOwners[checkpoint]; exists {
			return fmt.Errorf("duplicate workflow checkpoint path %q for steps %q and %q", filepath.ToSlash(checkpoint), owner, step.ID)
		}
		checkpointOwners[checkpoint] = step.ID
		for _, dep := range step.DependsOn {
			if !declared[dep] {
				return fmt.Errorf("dependency %s for step %s is not complete", dep, step.ID)
			}
		}
		declared[step.ID] = true
	}
	return nil
}
func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return filetxn.Replace(path, append(data, '\n'), 0o644)
}
