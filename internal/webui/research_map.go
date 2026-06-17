package webui

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/knowledge"
)

type ResearchMapCockpitState struct {
	SchemaVersion         string                  `json:"schemaVersion"`
	ProjectPath           string                  `json:"projectPath"`
	ConceptMap            []ResearchMapItem       `json:"conceptMap"`
	CitationNeighborhoods []ResearchMapItem       `json:"citationNeighborhoods"`
	RetrievalClusters     []ResearchMapItem       `json:"retrievalClusters"`
	EvidenceCoverage      EvidenceCoverageSummary `json:"evidenceCoverage"`
	SnapshotExportPath    string                  `json:"snapshotExportPath"`
}

type ResearchMapItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

type EvidenceCoverageSummary struct {
	Accepted  int `json:"accepted"`
	Suggested int `json:"suggested"`
	Other     int `json:"other"`
}

func BuildResearchMapCockpitState(projectPath string) (ResearchMapCockpitState, error) {
	graph, err := knowledge.BuildProjectKnowledgeGraphFromProject(projectPath)
	if err != nil {
		return ResearchMapCockpitState{}, err
	}
	state := ResearchMapCockpitState{SchemaVersion: "1", ProjectPath: projectPath, SnapshotExportPath: "/map/snapshot.json"}
	for _, node := range graph.Nodes {
		switch node.Kind {
		case "concept":
			state.ConceptMap = append(state.ConceptMap, ResearchMapItem{ID: node.ID, Label: node.Label, Detail: connectedPapers(graph, node.ID)})
		case "tag", "collection":
			state.RetrievalClusters = append(state.RetrievalClusters, ResearchMapItem{ID: node.ID, Label: node.Label, Detail: connectedPapers(graph, node.ID)})
		case "evidence":
			status := strings.ToLower(node.Properties["status"])
			if status == "accepted" {
				state.EvidenceCoverage.Accepted++
			} else if status == "suggested" {
				state.EvidenceCoverage.Suggested++
			} else {
				state.EvidenceCoverage.Other++
			}
		}
	}
	for _, edge := range graph.Edges {
		if edge.Kind == "cites" {
			state.CitationNeighborhoods = append(state.CitationNeighborhoods, ResearchMapItem{ID: edge.ID, Label: labelFor(graph, edge.Source), Detail: "cites " + labelFor(graph, edge.Target)})
		}
	}
	sortItems(state.ConceptMap)
	sortItems(state.CitationNeighborhoods)
	sortItems(state.RetrievalClusters)
	return state, nil
}

func NewResearchMapHandler(state ResearchMapCockpitState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = researchMapTemplate.Execute(w, state)
	})
}
func newResearchMapHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildResearchMapCockpitState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NewResearchMapHandler(state).ServeHTTP(w, r)
	})
}
func newResearchMapSnapshotHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildResearchMapCockpitState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	})
}

func connectedPapers(graph knowledge.ProjectKnowledgeGraph, nodeID string) string {
	papers := []string{}
	for _, edge := range graph.Edges {
		if edge.Target == nodeID && strings.HasPrefix(edge.Source, "paper:") {
			papers = append(papers, strings.TrimPrefix(edge.Source, "paper:"))
		}
	}
	sort.Strings(papers)
	return strings.Join(papers, ", ")
}
func labelFor(graph knowledge.ProjectKnowledgeGraph, id string) string {
	for _, node := range graph.Nodes {
		if node.ID == id {
			if node.Label != "" {
				return node.Label
			}
			return node.ID
		}
	}
	return id
}
func sortItems(items []ResearchMapItem) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}
