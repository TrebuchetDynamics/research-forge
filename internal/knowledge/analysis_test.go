package knowledge

import (
	"strings"
	"testing"
)

func TestAnalyzeProjectKnowledgeGraphFindsCentralNodesCommunitiesAndPaths(t *testing.T) {
	graph := ProjectKnowledgeGraph{SchemaVersion: "1", Nodes: []KnowledgeNode{
		{ID: "paper:a", Kind: "paper", Label: "Paper A"},
		{ID: "concept:bridge", Kind: "concept", Label: "Bridge concept"},
		{ID: "paper:b", Kind: "paper", Label: "Paper B"},
		{ID: "concept:solo", Kind: "concept", Label: "Solo concept"},
	}, Edges: []KnowledgeEdge{
		{Source: "paper:a", Target: "concept:bridge", Kind: "has_concept"},
		{Source: "concept:bridge", Target: "paper:b", Kind: "has_concept"},
	}}
	analysis := AnalyzeProjectKnowledgeGraph(graph)
	if analysis.NodeCount != 4 || analysis.EdgeCount != 2 || len(analysis.Communities) != 2 {
		t.Fatalf("analysis = %#v", analysis)
	}
	if analysis.CentralNodes[0].ID != "concept:bridge" || analysis.CentralNodes[0].Degree != 2 || analysis.CentralNodes[0].Betweenness == 0 {
		t.Fatalf("central nodes = %#v", analysis.CentralNodes)
	}
	path, ok := ShortestPathIDs(graph, "paper:a", "paper:b")
	if !ok || strings.Join(path, " -> ") != "paper:a -> concept:bridge -> paper:b" {
		t.Fatalf("path = %#v ok=%t", path, ok)
	}
}

func TestBuildKnowledgeGraphReportUsesCentralNodeLanguage(t *testing.T) {
	graph := ProjectKnowledgeGraph{SchemaVersion: "1", Nodes: []KnowledgeNode{
		{ID: "paper:a", Kind: "paper", Label: "Paper A"},
		{ID: "concept:bridge", Kind: "concept", Label: "Bridge concept"},
		{ID: "paper:b", Kind: "paper", Label: "Paper B"},
	}, Edges: []KnowledgeEdge{
		{Source: "paper:a", Target: "concept:bridge", Kind: "has_concept"},
		{Source: "concept:bridge", Target: "paper:b", Kind: "has_concept"},
	}}
	report := BuildKnowledgeGraphReport(graph)
	for _, want := range []string{"# Paper knowledge graph report", "Nodes: 3", "## Central nodes", "Bridge concept", "## Communities", "## Shortest paths", "Selected anchor path"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
	if strings.Contains(strings.ToLower(report), "god node") {
		t.Fatalf("report should avoid god-node terminology:\n%s", report)
	}
}
