package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
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
	run := WorkflowRun{SchemaVersion: schemaVersion, ProjectPath: projectPath, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	completed := map[string]bool{}
	for _, step := range dag.Steps {
		if checkpointExists(projectPath, step.Checkpoint) {
			completed[step.ID] = true
			run.RestartSafeSkipped++
			run.Checkpoints = append(run.Checkpoints, WorkflowCheckpoint{StepID: step.ID, Completed: true, Inputs: step.Inputs, Outputs: step.Outputs, Checkpoint: step.Checkpoint})
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
		cp := WorkflowCheckpoint{StepID: step.ID, Completed: true, Inputs: step.Inputs, Outputs: step.Outputs, Checkpoint: step.Checkpoint, CompletedAt: time.Now().UTC().Format(time.RFC3339)}
		if err := writeJSON(filepath.Join(projectPath, filepath.FromSlash(step.Checkpoint)), cp); err != nil {
			return run, err
		}
		if err := provenance.Append(projectPath, provenance.Event{SchemaVersion: schemaVersion, ID: "evt_" + time.Now().UTC().Format("20060102T150405Z") + "_workflow_" + step.ID, Timestamp: time.Now().UTC().Format(time.RFC3339), Actor: actor(opts.Actor), Action: step.ProvenanceAction, Target: step.ID, Inputs: map[string]any{"command": step.Command, "inputs": step.Inputs}, Outputs: map[string]any{"outputs": step.Outputs, "checkpoint": step.Checkpoint}, Warnings: []string{}}); err != nil {
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

func checkpointExists(projectPath, rel string) bool {
	_, err := os.Stat(filepath.Join(projectPath, filepath.FromSlash(rel)))
	return err == nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
