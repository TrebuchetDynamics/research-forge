package citations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type AccessibleGraphOptions struct {
	Filter string
}

type AccessibleGraphView struct {
	SchemaVersion      string                     `json:"schemaVersion"`
	Filter             string                     `json:"filter,omitempty"`
	Summary            GraphAccessibleSummary     `json:"summary"`
	Report             GraphReport                `json:"report"`
	KeyboardNavigation []string                   `json:"keyboardNavigation"`
	NodeRows           []AccessibleNodeRow        `json:"nodeRows"`
	EdgeRows           []AccessibleEdgeRow        `json:"edgeRows"`
	DomainTopicRows    []AccessibleDomainTopicRow `json:"domainTopicRows,omitempty"`
}

type GraphAccessibleSummary struct {
	NodeCount        int `json:"nodeCount"`
	EdgeCount        int `json:"edgeCount"`
	FilteredNodes    int `json:"filteredNodes"`
	FilteredEdges    int `json:"filteredEdges"`
	DomainTopicCount int `json:"domainTopicCount"`
}

type AccessibleNodeRow struct {
	NodeID    string `json:"nodeId"`
	InDegree  int    `json:"inDegree"`
	OutDegree int    `json:"outDegree"`
}

type AccessibleEdgeRow struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type AccessibleDomainTopicRow struct {
	TopicID                string `json:"topicId"`
	Label                  string `json:"label"`
	RepresentativePapers   string `json:"representativePapers"`
	RepresentativePassages string `json:"representativePassages"`
	CitationLinks          int    `json:"citationLinks"`
}

func BuildAccessibleGraphView(graphData []byte, domain DomainMapArtifact, opts AccessibleGraphOptions) (AccessibleGraphView, error) {
	var exported exportGraph
	if err := json.Unmarshal(graphData, &exported); err != nil {
		return AccessibleGraphView{}, err
	}
	filter := strings.TrimSpace(opts.Filter)
	in := inDegree(exported.Edges)
	out := outDegree(exported.Edges)
	nodeIDs := map[string]bool{}
	for _, node := range exported.Nodes {
		if strings.TrimSpace(node.ID) != "" {
			nodeIDs[node.ID] = true
		}
	}
	for _, edge := range exported.Edges {
		nodeIDs[edge.Source] = true
		nodeIDs[edge.Target] = true
	}
	view := AccessibleGraphView{SchemaVersion: "1", Filter: filter, KeyboardNavigation: []string{"Tab moves through filters, node rows, edge rows, and export links.", "Use browser find to jump to a paper, topic, or identifier.", "Each table has headers and can be copied or exported without JavaScript."}}
	for _, id := range sortedKeys(nodeIDs) {
		if filter != "" && !strings.Contains(strings.ToLower(id), strings.ToLower(filter)) {
			continue
		}
		view.NodeRows = append(view.NodeRows, AccessibleNodeRow{NodeID: id, InDegree: in[id], OutDegree: out[id]})
	}
	for _, edge := range exported.Edges {
		if filter != "" && !strings.Contains(strings.ToLower(edge.Source+" "+edge.Target), strings.ToLower(filter)) {
			continue
		}
		view.EdgeRows = append(view.EdgeRows, AccessibleEdgeRow{Source: edge.Source, Target: edge.Target})
	}
	for _, topic := range domain.Topics {
		row := AccessibleDomainTopicRow{TopicID: topic.TopicID, Label: topic.Label, RepresentativePapers: joinRepresentativePapers(topic.RepresentativePapers), RepresentativePassages: joinRepresentativePassages(topic.RepresentativePassages), CitationLinks: len(topic.CitationGraphLinks)}
		if filter != "" && !strings.Contains(strings.ToLower(row.TopicID+" "+row.Label+" "+row.RepresentativePapers+" "+row.RepresentativePassages), strings.ToLower(filter)) {
			continue
		}
		view.DomainTopicRows = append(view.DomainTopicRows, row)
	}
	sort.Slice(view.EdgeRows, func(i, j int) bool {
		if view.EdgeRows[i].Source == view.EdgeRows[j].Source {
			return view.EdgeRows[i].Target < view.EdgeRows[j].Target
		}
		return view.EdgeRows[i].Source < view.EdgeRows[j].Source
	})
	view.Summary = GraphAccessibleSummary{NodeCount: len(nodeIDs), EdgeCount: len(exported.Edges), FilteredNodes: len(view.NodeRows), FilteredEdges: len(view.EdgeRows), DomainTopicCount: len(view.DomainTopicRows)}
	if report, err := BuildGraphReport(graphData); err == nil {
		view.Report = report
	}
	return view, nil
}

func AccessibleGraphMarkdown(view AccessibleGraphView) string {
	var b strings.Builder
	b.WriteString("# Accessible graph view\n\n")
	fmt.Fprintf(&b, "- Nodes: %d\n- Edges: %d\n- Filtered nodes: %d\n- Filtered edges: %d\n- Domain topics: %d\n\n", view.Summary.NodeCount, view.Summary.EdgeCount, view.Summary.FilteredNodes, view.Summary.FilteredEdges, view.Summary.DomainTopicCount)
	b.WriteString("## Graph summary\n\n")
	b.WriteString(GraphReportMarkdown(view.Report))
	b.WriteString("\n## Keyboard navigation\n\n")
	for _, item := range view.KeyboardNavigation {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	b.WriteString("\n## Filtered node table\n\n| Node | In degree | Out degree |\n| --- | ---: | ---: |\n")
	for _, row := range view.NodeRows {
		fmt.Fprintf(&b, "| `%s` | %d | %d |\n", row.NodeID, row.InDegree, row.OutDegree)
	}
	b.WriteString("\n## Edge list\n\n| Source | Target |\n| --- | --- |\n")
	for _, row := range view.EdgeRows {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", row.Source, row.Target)
	}
	b.WriteString("\n## Domain topics\n\n| Topic | Label | Representative papers | Representative passages | Citation links |\n| --- | --- | --- | --- | ---: |\n")
	for _, row := range view.DomainTopicRows {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %d |\n", row.TopicID, row.Label, row.RepresentativePapers, row.RepresentativePassages, row.CitationLinks)
	}
	return b.String()
}

func joinRepresentativePapers(papers []RepresentativePaper) string {
	parts := []string{}
	for _, p := range papers {
		if p.Title != "" {
			parts = append(parts, p.PaperID+" "+p.Title)
		} else {
			parts = append(parts, p.PaperID)
		}
	}
	return strings.Join(parts, "; ")
}
func joinRepresentativePassages(passages []RepresentativePassage) string {
	parts := []string{}
	for _, p := range passages {
		parts = append(parts, p.PaperID+":"+p.PassageID)
	}
	return strings.Join(parts, "; ")
}
