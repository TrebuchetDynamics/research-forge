package knowledge

import (
	"fmt"
	"sort"
	"strings"
)

type GraphAnalysis struct {
	NodeCount    int
	EdgeCount    int
	CentralNodes []GraphNodeMetric
	Communities  []GraphCommunity
}

type GraphNodeMetric struct {
	ID               string
	Label            string
	Kind             string
	Degree           int
	DegreeCentrality float64
	Betweenness      float64
}

type GraphCommunity struct {
	ID      int
	NodeIDs []string
}

func AnalyzeProjectKnowledgeGraph(graph ProjectKnowledgeGraph) GraphAnalysis {
	adj := graphAdjacency(graph)
	metrics := make([]GraphNodeMetric, 0, len(graph.Nodes))
	between := betweennessCentrality(adj)
	denom := float64(len(graph.Nodes) - 1)
	if denom < 1 {
		denom = 1
	}
	for _, node := range graph.Nodes {
		degree := len(adj[node.ID])
		metrics = append(metrics, GraphNodeMetric{ID: node.ID, Label: node.Label, Kind: node.Kind, Degree: degree, DegreeCentrality: float64(degree) / denom, Betweenness: between[node.ID]})
	}
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Degree != metrics[j].Degree {
			return metrics[i].Degree > metrics[j].Degree
		}
		if metrics[i].Betweenness != metrics[j].Betweenness {
			return metrics[i].Betweenness > metrics[j].Betweenness
		}
		return metrics[i].ID < metrics[j].ID
	})
	return GraphAnalysis{NodeCount: len(graph.Nodes), EdgeCount: len(graph.Edges), CentralNodes: metrics, Communities: connectedCommunities(adj)}
}

func BuildKnowledgeGraphReport(graph ProjectKnowledgeGraph) string {
	analysis := AnalyzeProjectKnowledgeGraph(graph)
	var b strings.Builder
	fmt.Fprintf(&b, "# Paper knowledge graph report\n\n")
	fmt.Fprintf(&b, "Nodes: %d\n\nEdges: %d\n\n", analysis.NodeCount, analysis.EdgeCount)
	b.WriteString("## Central nodes\n\n")
	b.WriteString("These are central graph nodes, not automatically overloaded or authoritative nodes. In paper graphs they may be foundational papers, broad concepts, or hub citations.\n\n")
	b.WriteString("| Node | Kind | Degree | Degree centrality | Betweenness |\n|---|---:|---:|---:|---:|\n")
	for i, node := range analysis.CentralNodes {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %.3f | %.3f |\n", escapeMarkdown(firstNonEmpty(node.Label, node.ID)), node.Kind, node.Degree, node.DegreeCentrality, node.Betweenness)
	}
	b.WriteString("\n## Communities\n\n")
	for _, community := range analysis.Communities {
		labels := []string{}
		for _, id := range community.NodeIDs {
			labels = append(labels, escapeMarkdown(labelForNode(graph, id)))
		}
		if len(labels) > 12 {
			labels = append(labels[:12], "…")
		}
		fmt.Fprintf(&b, "- Community %d (%d nodes): %s\n", community.ID, len(community.NodeIDs), strings.Join(labels, ", "))
	}
	b.WriteString("\n## Shortest paths\n\n")
	from, to := reportPathAnchors(analysis.CentralNodes)
	if from == "" || to == "" {
		b.WriteString("No anchor pair available.\n")
	} else if path, ok := ShortestPathIDs(graph, from, to); ok {
		parts := []string{}
		for _, id := range path {
			parts = append(parts, escapeMarkdown(labelForNode(graph, id)))
		}
		fmt.Fprintf(&b, "Selected anchor path: %s\n", strings.Join(parts, " → "))
	} else {
		fmt.Fprintf(&b, "No path found between %s and %s.\n", escapeMarkdown(labelForNode(graph, from)), escapeMarkdown(labelForNode(graph, to)))
	}
	return b.String()
}

func ShortestPathIDs(graph ProjectKnowledgeGraph, fromID, toID string) ([]string, bool) {
	fromID = strings.TrimSpace(fromID)
	toID = strings.TrimSpace(toID)
	adj := graphAdjacency(graph)
	if !hasAdjNode(adj, fromID) || !hasAdjNode(adj, toID) {
		return nil, false
	}
	queue := []string{fromID}
	seen := map[string]bool{fromID: true}
	prev := map[string]string{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == toID {
			path := []string{id}
			for id != fromID {
				id = prev[id]
				path = append(path, id)
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path, true
		}
		neighbors := sortedNeighbors(adj[id])
		for _, next := range neighbors {
			if !seen[next] {
				seen[next] = true
				prev[next] = id
				queue = append(queue, next)
			}
		}
	}
	return nil, false
}

func graphAdjacency(graph ProjectKnowledgeGraph) map[string]map[string]bool {
	adj := map[string]map[string]bool{}
	for _, node := range graph.Nodes {
		adj[node.ID] = map[string]bool{}
	}
	for _, edge := range graph.Edges {
		if _, ok := adj[edge.Source]; !ok {
			adj[edge.Source] = map[string]bool{}
		}
		if _, ok := adj[edge.Target]; !ok {
			adj[edge.Target] = map[string]bool{}
		}
		adj[edge.Source][edge.Target] = true
		adj[edge.Target][edge.Source] = true
	}
	return adj
}

func betweennessCentrality(adj map[string]map[string]bool) map[string]float64 {
	cb := map[string]float64{}
	nodes := make([]string, 0, len(adj))
	for id := range adj {
		nodes = append(nodes, id)
		cb[id] = 0
	}
	sort.Strings(nodes)
	for _, s := range nodes {
		stack := []string{}
		pred := map[string][]string{}
		sigma := map[string]float64{s: 1}
		dist := map[string]int{s: 0}
		queue := []string{s}
		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			stack = append(stack, v)
			for _, w := range sortedNeighbors(adj[v]) {
				if _, ok := dist[w]; !ok {
					dist[w] = dist[v] + 1
					queue = append(queue, w)
				}
				if dist[w] == dist[v]+1 {
					sigma[w] += sigma[v]
					pred[w] = append(pred[w], v)
				}
			}
		}
		delta := map[string]float64{}
		for len(stack) > 0 {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, v := range pred[w] {
				if sigma[w] != 0 {
					delta[v] += (sigma[v] / sigma[w]) * (1 + delta[w])
				}
			}
			if w != s {
				cb[w] += delta[w]
			}
		}
	}
	for id := range cb {
		cb[id] /= 2
	}
	return cb
}

func connectedCommunities(adj map[string]map[string]bool) []GraphCommunity {
	seen := map[string]bool{}
	ids := make([]string, 0, len(adj))
	for id := range adj {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	communities := []GraphCommunity{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		queue := []string{id}
		seen[id] = true
		component := []string{}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			component = append(component, cur)
			for _, next := range sortedNeighbors(adj[cur]) {
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
		sort.Strings(component)
		communities = append(communities, GraphCommunity{ID: len(communities) + 1, NodeIDs: component})
	}
	sort.Slice(communities, func(i, j int) bool {
		if len(communities[i].NodeIDs) != len(communities[j].NodeIDs) {
			return len(communities[i].NodeIDs) > len(communities[j].NodeIDs)
		}
		return communities[i].NodeIDs[0] < communities[j].NodeIDs[0]
	})
	for i := range communities {
		communities[i].ID = i + 1
	}
	return communities
}

func sortedNeighbors(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func hasAdjNode(adj map[string]map[string]bool, id string) bool {
	_, ok := adj[id]
	return ok
}

func labelForNode(graph ProjectKnowledgeGraph, id string) string {
	for _, node := range graph.Nodes {
		if node.ID == id {
			return firstNonEmpty(node.Label, node.ID)
		}
	}
	return id
}

func reportPathAnchors(nodes []GraphNodeMetric) (string, string) {
	if len(nodes) < 2 {
		return "", ""
	}
	first := nodes[0].ID
	for _, node := range nodes[1:] {
		if node.ID != first {
			return first, node.ID
		}
	}
	return "", ""
}

func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
