package citations

import (
	"encoding/json"
	"sort"
)

// Graph stores citation relationships as citing paper -> referenced paper.
type Graph struct {
	edges map[string]map[string]bool
}

// NewGraph creates an empty citation graph.
func NewGraph() *Graph { return &Graph{edges: map[string]map[string]bool{}} }

// AddCitation records that citingPaperID cites referencedPaperID.
func (g *Graph) AddCitation(citingPaperID, referencedPaperID string) {
	if g.edges[citingPaperID] == nil {
		g.edges[citingPaperID] = map[string]bool{}
	}
	g.edges[citingPaperID][referencedPaperID] = true
}

// Backward returns references cited by paperID.
func (g *Graph) Backward(paperID string) []string { return sortedKeys(g.edges[paperID]) }

// Forward returns papers that cite paperID.
func (g *Graph) Forward(paperID string) []string {
	var out []string
	for citing, refs := range g.edges {
		if refs[paperID] {
			out = append(out, citing)
		}
	}
	sort.Strings(out)
	return out
}

// CoCitationCluster groups papers citing the same reference.
type CoCitationCluster struct {
	ReferenceID    string
	CitingPaperIDs []string
}

// BibliographicCouplingPair groups papers sharing a reference.
type BibliographicCouplingPair struct {
	PaperA            string
	PaperB            string
	SharedReferenceID string
}

// CoCitationClusters returns simple same-reference clusters.
func (g *Graph) CoCitationClusters() []CoCitationCluster {
	refs := map[string][]string{}
	for citing, cited := range g.edges {
		for ref := range cited {
			refs[ref] = append(refs[ref], citing)
		}
	}
	keys := sortedKeys(refs)
	var clusters []CoCitationCluster
	for _, ref := range keys {
		if len(refs[ref]) > 1 {
			sort.Strings(refs[ref])
			clusters = append(clusters, CoCitationCluster{ReferenceID: ref, CitingPaperIDs: refs[ref]})
		}
	}
	return clusters
}

// BibliographicCoupling returns simple shared-reference pairs.
func (g *Graph) BibliographicCoupling() []BibliographicCouplingPair {
	var pairs []BibliographicCouplingPair
	clusters := g.CoCitationClusters()
	for _, cluster := range clusters {
		for i := 0; i < len(cluster.CitingPaperIDs); i++ {
			for j := i + 1; j < len(cluster.CitingPaperIDs); j++ {
				pairs = append(pairs, BibliographicCouplingPair{PaperA: cluster.CitingPaperIDs[i], PaperB: cluster.CitingPaperIDs[j], SharedReferenceID: cluster.ReferenceID})
			}
		}
	}
	return pairs
}

type exportGraph struct {
	Nodes []exportNode `json:"nodes"`
	Edges []exportEdge `json:"edges"`
}
type exportNode struct {
	ID string `json:"id"`
}
type exportEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ExportJSON writes a stable node/edge JSON graph.
func (g *Graph) ExportJSON() ([]byte, error) {
	nodesMap := map[string]bool{}
	var edges []exportEdge
	for source, targets := range g.edges {
		nodesMap[source] = true
		for target := range targets {
			nodesMap[target] = true
			edges = append(edges, exportEdge{Source: source, Target: target})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source == edges[j].Source {
			return edges[i].Target < edges[j].Target
		}
		return edges[i].Source < edges[j].Source
	})
	var nodes []exportNode
	for _, id := range sortedKeys(nodesMap) {
		nodes = append(nodes, exportNode{ID: id})
	}
	data, err := json.MarshalIndent(exportGraph{Nodes: nodes, Edges: edges}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
