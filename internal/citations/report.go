package citations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// GraphReport is a deterministic summary of a citation graph artifact.
type GraphReport struct {
	NodeCount     int      `json:"nodeCount"`
	EdgeCount     int      `json:"edgeCount"`
	TopCited      []Degree `json:"topCited"`
	TopCiting     []Degree `json:"topCiting"`
	CoCitations   int      `json:"coCitations"`
	CouplingPairs int      `json:"couplingPairs"`
}

// Degree records a paper's graph degree for reporting.
type Degree struct {
	PaperID string `json:"paperId"`
	Count   int    `json:"count"`
}

// BuildGraphReport summarizes exported node/edge citation graph JSON.
func BuildGraphReport(data []byte) (GraphReport, error) {
	var exported exportGraph
	if err := json.Unmarshal(data, &exported); err != nil {
		return GraphReport{}, err
	}
	graph := NewGraph()
	for _, edge := range exported.Edges {
		graph.AddCitation(edge.Source, edge.Target)
	}
	nodes := map[string]bool{}
	for _, node := range exported.Nodes {
		if strings.TrimSpace(node.ID) != "" {
			nodes[node.ID] = true
		}
	}
	for _, edge := range exported.Edges {
		nodes[edge.Source] = true
		nodes[edge.Target] = true
	}
	return GraphReport{
		NodeCount:     len(nodes),
		EdgeCount:     len(exported.Edges),
		TopCited:      topDegrees(inDegree(exported.Edges), 5),
		TopCiting:     topDegrees(outDegree(exported.Edges), 5),
		CoCitations:   len(graph.CoCitationClusters()),
		CouplingPairs: len(graph.BibliographicCoupling()),
	}, nil
}

// GraphReportMarkdown renders a citation graph report as Markdown.
func GraphReportMarkdown(report GraphReport) string {
	var b strings.Builder
	b.WriteString("# Citation graph report\n\n")
	fmt.Fprintf(&b, "- Nodes: %d\n", report.NodeCount)
	fmt.Fprintf(&b, "- Edges: %d\n", report.EdgeCount)
	fmt.Fprintf(&b, "- Co-citation clusters: %d\n", report.CoCitations)
	fmt.Fprintf(&b, "- Bibliographic coupling pairs: %d\n\n", report.CouplingPairs)
	writeDegreeTable(&b, "Top cited papers", report.TopCited)
	writeDegreeTable(&b, "Top citing papers", report.TopCiting)
	return b.String()
}

func writeDegreeTable(b *strings.Builder, title string, degrees []Degree) {
	fmt.Fprintf(b, "## %s\n\n", title)
	b.WriteString("| Paper | Count |\n| --- | ---: |\n")
	if len(degrees) == 0 {
		b.WriteString("| none | 0 |\n\n")
		return
	}
	for _, degree := range degrees {
		fmt.Fprintf(b, "| `%s` | %d |\n", degree.PaperID, degree.Count)
	}
	b.WriteString("\n")
}

func inDegree(edges []exportEdge) map[string]int {
	degrees := map[string]int{}
	for _, edge := range edges {
		degrees[edge.Target]++
	}
	return degrees
}

func outDegree(edges []exportEdge) map[string]int {
	degrees := map[string]int{}
	for _, edge := range edges {
		degrees[edge.Source]++
	}
	return degrees
}

func topDegrees(degrees map[string]int, limit int) []Degree {
	out := make([]Degree, 0, len(degrees))
	for paperID, count := range degrees {
		out = append(out, Degree{PaperID: paperID, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].PaperID < out[j].PaperID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
