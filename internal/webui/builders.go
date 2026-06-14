package webui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

// BuildLibraryViewModel reads a CLI-generated project's library into the cockpit
// library view model. A project without a library yields an empty view model
// and is not treated as an error.
func BuildLibraryViewModel(projectPath string) (ui.LibraryViewModel, error) {
	libPath := filepath.Join(projectPath, "data", "library.json")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return ui.NewLibraryViewModel(nil), nil
	}
	store, err := library.OpenStore(libPath)
	if err != nil {
		return ui.LibraryViewModel{}, err
	}
	papers, err := store.List()
	if err != nil {
		return ui.LibraryViewModel{}, err
	}
	rows := make([]ui.PaperRow, 0, len(papers))
	for _, paper := range papers {
		rows = append(rows, ui.PaperRow{Title: paper.Title})
	}
	return ui.NewLibraryViewModel(rows), nil
}

// BuildArtifactDashboardState assembles the artifacts cockpit view from a
// CLI-generated project workspace: imported papers, screening-derived PRISMA
// counts, and meta-analysis readiness.
func BuildArtifactDashboardState(projectPath string) (ArtifactDashboardState, error) {
	papers, err := BuildLibraryViewModel(projectPath)
	if err != nil {
		return ArtifactDashboardState{}, err
	}
	prisma, err := buildPRISMAFlowState(projectPath, len(papers.Rows))
	if err != nil {
		return ArtifactDashboardState{}, err
	}
	return ArtifactDashboardState{
		Papers:   papers,
		PRISMA:   prisma,
		Analysis: buildAnalysisViewModel(projectPath),
	}, nil
}

// buildPRISMAFlowState replays the project's stored screening decisions into
// PRISMA flow counts. A project without a screening workflow yields just the
// record count.
func buildPRISMAFlowState(projectPath string, libraryCount int) (PRISMAFlowState, error) {
	state := PRISMAFlowState{Records: libraryCount}
	workflowData, err := os.ReadFile(filepath.Join(projectPath, "data", "screening.workflow.json"))
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	var workflow screening.Workflow
	if err := json.Unmarshal(workflowData, &workflow); err != nil {
		return state, err
	}
	var events []screening.DecisionEvent
	if eventsData, err := os.ReadFile(filepath.Join(projectPath, "data", "screening.events.json")); err == nil {
		_ = json.Unmarshal(eventsData, &events)
	}
	store := screening.NewMemoryStore(workflow)
	decided := map[string]bool{}
	for _, event := range events {
		_ = store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer})
		decided[event.PaperID] = true
	}
	state.Screened = len(decided)
	state.Included = store.PRISMACounts().Included
	return state, nil
}

// buildAnalysisViewModel reports whether the project has a stored meta-analysis
// result (analysis/<run>-result.json), naming the first run it finds.
func buildAnalysisViewModel(projectPath string) ui.AnalysisViewModel {
	entries, err := os.ReadDir(filepath.Join(projectPath, "analysis"))
	if err != nil {
		return ui.NewAnalysisViewModel("", false)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), "-result.json") {
			return ui.NewAnalysisViewModel(strings.TrimSuffix(entry.Name(), "-result.json"), true)
		}
	}
	return ui.NewAnalysisViewModel("", false)
}
