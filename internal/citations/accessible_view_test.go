package citations

import (
	"strings"
	"testing"
)

func TestBuildAccessibleGraphViewFiltersNodesEdgesDomainTopicsAndRendersKeyboardMarkdown(t *testing.T) {
	graph := NewGraph()
	graph.AddCitation("paper-1", "paper-2")
	graph.AddCitation("paper-3", "paper-1")
	graphData, err := graph.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	domain := DomainMapArtifact{Topics: []DomainTopic{{TopicID: "solar", Label: "Solar fuels", RepresentativePapers: []RepresentativePaper{{PaperID: "paper-1", Title: "Solar catalyst"}}, CitationGraphLinks: []CitationGraphLink{{SourceID: "paper-1", TargetID: "paper-2", Relation: "cites"}}}}}
	view, err := BuildAccessibleGraphView(graphData, domain, AccessibleGraphOptions{Filter: "paper-1"})
	if err != nil {
		t.Fatalf("BuildAccessibleGraphView returned error: %v", err)
	}
	if view.Summary.NodeCount != 3 || view.Summary.EdgeCount != 2 || view.Filter != "paper-1" || view.Report.NodeCount != 3 || view.Report.EdgeCount != 2 {
		t.Fatalf("summary/filter = %#v", view)
	}
	if len(view.NodeRows) != 1 || view.NodeRows[0].NodeID != "paper-1" || view.NodeRows[0].InDegree != 1 || view.NodeRows[0].OutDegree != 1 {
		t.Fatalf("node rows = %#v", view.NodeRows)
	}
	if len(view.EdgeRows) != 2 {
		t.Fatalf("edge rows = %#v", view.EdgeRows)
	}
	if len(view.DomainTopicRows) != 1 || view.DomainTopicRows[0].TopicID != "solar" {
		t.Fatalf("domain rows = %#v", view.DomainTopicRows)
	}
	markdown := AccessibleGraphMarkdown(view)
	for _, want := range []string{"# Accessible graph view", "## Graph summary", "## Keyboard navigation", "| Node | In degree | Out degree |", "| Source | Target |", "Solar fuels"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q:\n%s", want, markdown)
		}
	}
}
