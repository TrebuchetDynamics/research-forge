package webui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
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
	graph, err := buildCitationGraph(projectPath)
	if err != nil {
		return ArtifactDashboardState{}, err
	}
	return ArtifactDashboardState{
		Papers:         papers,
		PRISMA:         prisma,
		Analysis:       buildAnalysisViewModel(projectPath),
		AnalysisDetail: buildAnalysisDetail(projectPath),
		CitationGraph:  graph,
	}, nil
}

// buildCitationGraph loads the project's exported citation graph
// (data/citation-graph.json, the stable nodes/edges export format) into the
// cockpit citation-graph view model. A project without a graph yields an empty,
// non-error model.
func buildCitationGraph(projectPath string) (ui.CitationGraphViewModel, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, "data", "citation-graph.json"))
	if os.IsNotExist(err) {
		return ui.CitationGraphViewModel{}, nil
	}
	if err != nil {
		return ui.CitationGraphViewModel{}, err
	}
	var export struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return ui.CitationGraphViewModel{}, err
	}
	nodes := make([]ui.GraphNode, 0, len(export.Nodes))
	for _, n := range export.Nodes {
		nodes = append(nodes, ui.GraphNode{ID: n.ID})
	}
	edges := make([]ui.GraphEdge, 0, len(export.Edges))
	for _, e := range export.Edges {
		edges = append(edges, ui.GraphEdge{Source: e.Source, Target: e.Target})
	}
	return ui.NewCitationGraphViewModel(nodes, edges), nil
}

// buildAnalysisDetail loads the project's stored meta-analysis result
// (analysis/<run>-result.json) into a readable detail view: heterogeneity
// metrics, plot availability, and any runner warnings. A project without a
// stored result yields a not-ready detail.
func buildAnalysisDetail(projectPath string) AnalysisDetail {
	dir := filepath.Join(projectPath, "analysis")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return AnalysisDetail{}
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-result.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return AnalysisDetail{}
		}
		var result analysis.AnalysisResult
		if err := json.Unmarshal(data, &result); err != nil {
			return AnalysisDetail{}
		}
		return AnalysisDetail{
			Ready:         true,
			RunID:         strings.TrimSuffix(entry.Name(), "-result.json"),
			I2:            result.Metrics.I2,
			Tau2:          result.Metrics.Tau2,
			Q:             result.Metrics.Q,
			HasForestPlot: strings.TrimSpace(result.ForestPlot.Path) != "",
			HasFunnelPlot: strings.TrimSpace(result.FunnelPlot.Path) != "",
			Warnings:      result.Warnings,
		}
	}
	return AnalysisDetail{}
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
