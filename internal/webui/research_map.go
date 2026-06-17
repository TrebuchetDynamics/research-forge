package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/knowledge"
)

type ResearchMapCockpitState struct {
	SchemaVersion         string                  `json:"schemaVersion"`
	ProjectPath           string                  `json:"projectPath"`
	Filter                string                  `json:"filter,omitempty"`
	Neighborhood          string                  `json:"neighborhood,omitempty"`
	ConceptMap            []ResearchMapItem       `json:"conceptMap"`
	CitationNeighborhoods []ResearchMapItem       `json:"citationNeighborhoods"`
	RetrievalClusters     []ResearchMapItem       `json:"retrievalClusters"`
	RetrievalHits         []ResearchMapItem       `json:"retrievalHits,omitempty"`
	ScreeningPriority     []ResearchMapItem       `json:"screeningPriority,omitempty"`
	ScreeningStatus       []ResearchMapItem       `json:"screeningStatus,omitempty"`
	ParserQuality         []ResearchMapItem       `json:"parserQuality,omitempty"`
	EvidenceCoverage      EvidenceCoverageSummary `json:"evidenceCoverage"`
	ProvenanceOverlays    []ResearchMapItem       `json:"provenanceOverlays,omitempty"`
	KeyboardAlternatives  []string                `json:"keyboardAlternatives"`
	SnapshotExportPath    string                  `json:"snapshotExportPath"`
}

type ResearchMapOptions struct {
	Filter, Neighborhood string
	IncludeProvenance    bool
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
	return BuildResearchMapCockpitStateWithOptions(projectPath, ResearchMapOptions{})
}

func BuildResearchMapCockpitStateWithOptions(projectPath string, opts ResearchMapOptions) (ResearchMapCockpitState, error) {
	graph, err := knowledge.BuildProjectKnowledgeGraphFromProject(projectPath)
	if err != nil {
		return ResearchMapCockpitState{}, err
	}
	filter := strings.ToLower(strings.TrimSpace(opts.Filter))
	state := ResearchMapCockpitState{SchemaVersion: "1", ProjectPath: projectPath, Filter: opts.Filter, Neighborhood: opts.Neighborhood, SnapshotExportPath: "/map/snapshot.json", KeyboardAlternatives: []string{"Tab through concept, citation, retrieval, and provenance tables", "Use /map/snapshot.json for screen-reader-friendly JSON export"}}
	for _, node := range graph.Nodes {
		if !researchMapMatch(node.Label+" "+node.ID, filter) {
			continue
		}
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
		if opts.Neighborhood != "" && !strings.Contains(edge.Source+" "+edge.Target, opts.Neighborhood) {
			continue
		}
		if edge.Kind == "cites" {
			state.CitationNeighborhoods = append(state.CitationNeighborhoods, ResearchMapItem{ID: edge.ID, Label: labelFor(graph, edge.Source), Detail: "cites " + labelFor(graph, edge.Target)})
		}
	}
	state.RetrievalHits = readResearchMapItems(filepath.Join(projectPath, "data", "retrieval-hits.json"))
	state.ScreeningPriority = readResearchMapItems(filepath.Join(projectPath, "data", "screening-priority.json"))
	state.ScreeningStatus = readResearchMapItems(filepath.Join(projectPath, "data", "screening-status.json"))
	state.ParserQuality = readParserQualityItems(filepath.Join(projectPath, "data", "parser-quality.json"))
	if opts.IncludeProvenance {
		for _, node := range graph.Nodes {
			if node.Kind == "provenance_event" && researchMapMatch(node.Label+" "+node.ID, filter) {
				state.ProvenanceOverlays = append(state.ProvenanceOverlays, ResearchMapItem{ID: node.ID, Label: node.Label, Detail: node.Properties["target"]})
			}
		}
		if len(state.ProvenanceOverlays) == 0 {
			state.ProvenanceOverlays = append(state.ProvenanceOverlays, ResearchMapItem{ID: "provenance:available", Label: "Project provenance", Detail: "Use /notebook for complete workflow event timeline"})
		}
	}
	sortItems(state.ConceptMap)
	sortItems(state.CitationNeighborhoods)
	sortItems(state.RetrievalClusters)
	sortItems(state.RetrievalHits)
	sortItems(state.ScreeningPriority)
	sortItems(state.ScreeningStatus)
	sortItems(state.ParserQuality)
	sortItems(state.ProvenanceOverlays)
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
		state, err := BuildResearchMapCockpitStateWithOptions(projectPath(), ResearchMapOptions{Filter: r.URL.Query().Get("filter"), Neighborhood: r.URL.Query().Get("neighborhood"), IncludeProvenance: r.URL.Query().Get("provenance") != ""})
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

func readResearchMapItems(path string) []ResearchMapItem {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var items []ResearchMapItem
	if err := json.Unmarshal(payload, &items); err == nil {
		return items
	}
	var generic []map[string]any
	if err := json.Unmarshal(payload, &generic); err != nil {
		return nil
	}
	for _, item := range generic {
		items = append(items, ResearchMapItem{ID: fmt.Sprint(item["id"]), Label: fmt.Sprint(item["label"]), Detail: fmt.Sprint(item["detail"])})
	}
	return items
}

func readParserQualityItems(path string) []ResearchMapItem {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var report struct {
		ParserRuns []struct {
			ParserName   string  `json:"parserName"`
			QualityScore float64 `json:"qualityScore"`
		} `json:"parserRuns"`
		Conflicts []struct {
			Field  string `json:"field"`
			Status string `json:"status"`
		} `json:"conflicts"`
	}
	if err := json.Unmarshal(payload, &report); err != nil {
		return nil
	}
	items := []ResearchMapItem{}
	for _, run := range report.ParserRuns {
		items = append(items, ResearchMapItem{ID: "parser:" + run.ParserName, Label: run.ParserName, Detail: fmt.Sprintf("quality %.2f", run.QualityScore)})
	}
	for _, conflict := range report.Conflicts {
		items = append(items, ResearchMapItem{ID: "parser-conflict:" + conflict.Field, Label: conflict.Field, Detail: conflict.Status})
	}
	return items
}

func researchMapMatch(value, filter string) bool {
	return filter == "" || strings.Contains(strings.ToLower(value), filter)
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
