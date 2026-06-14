package citations

import (
	"strings"
	"testing"
)

func TestBuildGraphReportSummarizesCitationGraph(t *testing.T) {
	graph := NewGraph()
	graph.AddCitation("paper-a", "ref-1")
	graph.AddCitation("paper-b", "ref-1")
	graph.AddCitation("paper-a", "ref-2")
	data, err := graph.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON returned error: %v", err)
	}

	report, err := BuildGraphReport(data)
	if err != nil {
		t.Fatalf("BuildGraphReport returned error: %v", err)
	}
	if report.NodeCount != 4 || report.EdgeCount != 3 || report.CoCitations != 1 || report.CouplingPairs != 1 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.TopCited) == 0 || report.TopCited[0].PaperID != "ref-1" || report.TopCited[0].Count != 2 {
		t.Fatalf("top cited = %#v", report.TopCited)
	}
	if len(report.TopCiting) == 0 || report.TopCiting[0].PaperID != "paper-a" || report.TopCiting[0].Count != 2 {
		t.Fatalf("top citing = %#v", report.TopCiting)
	}
}

func TestGraphReportMarkdownRendersTables(t *testing.T) {
	markdown := GraphReportMarkdown(GraphReport{NodeCount: 2, EdgeCount: 1, TopCited: []Degree{{PaperID: "ref-1", Count: 1}}})
	for _, want := range []string{"# Citation graph report", "- Nodes: 2", "## Top cited papers", "`ref-1`"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q:\n%s", want, markdown)
		}
	}
}
