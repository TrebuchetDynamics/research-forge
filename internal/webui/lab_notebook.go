package webui

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

type LabNotebookTimelineState struct {
	SchemaVersion   string             `json:"schemaVersion"`
	ProjectPath     string             `json:"projectPath"`
	TotalEvents     int                `json:"totalEvents"`
	HumanEvents     int                `json:"humanEvents"`
	AutomatedEvents int                `json:"automatedEvents"`
	Events          []LabNotebookEvent `json:"events"`
	SnapshotPath    string             `json:"snapshotPath"`
}

type LabNotebookEvent struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Actor     string         `json:"actor"`
	ActorKind string         `json:"actorKind"`
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	Inputs    map[string]any `json:"inputs,omitempty"`
	Outputs   map[string]any `json:"outputs,omitempty"`
	Warnings  []string       `json:"warnings,omitempty"`
}

func BuildLabNotebookTimelineState(projectPath string) (LabNotebookTimelineState, error) {
	events, err := provenance.Read(projectPath)
	if err != nil {
		events = nil
	}
	state := LabNotebookTimelineState{SchemaVersion: "1", ProjectPath: projectPath, SnapshotPath: "/notebook/snapshot.json"}
	for _, event := range events {
		kind := actorKind(event.Actor)
		entry := LabNotebookEvent{ID: event.ID, Timestamp: event.Timestamp, Actor: event.Actor, ActorKind: kind, Action: event.Action, Target: event.Target, Inputs: event.Inputs, Outputs: event.Outputs, Warnings: event.Warnings}
		state.Events = append(state.Events, entry)
		if kind == "human" {
			state.HumanEvents++
		} else {
			state.AutomatedEvents++
		}
	}
	sort.Slice(state.Events, func(i, j int) bool {
		if state.Events[i].Timestamp == state.Events[j].Timestamp {
			return state.Events[i].ID < state.Events[j].ID
		}
		return state.Events[i].Timestamp < state.Events[j].Timestamp
	})
	state.TotalEvents = len(state.Events)
	return state, nil
}

func NewLabNotebookHandler(state LabNotebookTimelineState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = labNotebookTemplate.Execute(w, state)
	})
}
func newLabNotebookHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildLabNotebookTimelineState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NewLabNotebookHandler(state).ServeHTTP(w, r)
	})
}
func newLabNotebookSnapshotHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildLabNotebookTimelineState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	})
}

func actorKind(actor string) string {
	actor = strings.ToLower(strings.TrimSpace(actor))
	if actor == "" || actor == "rforge" || actor == "cli" || strings.Contains(actor, "bot") || strings.Contains(actor, "automation") || strings.Contains(actor, "job") {
		return "automated"
	}
	return "human"
}
