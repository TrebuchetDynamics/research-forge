package webui

import (
	"encoding/json"
	"net/http"
)

// GraphJSONNode is one citation-graph node delivered to the interactive client.
// Stem/Href let the browser link a node to its /papers/{id} reading page without
// re-implementing the CLI's safe-stem normalization.
type GraphJSONNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Stem  string `json:"stem"`
	Href  string `json:"href"`
}

// GraphJSONEdge is one directed citation edge (citing -> referenced).
type GraphJSONEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// GraphJSON is the data payload the vendored citation-graph.js fetches and
// renders interactively. It is built from the same project citation graph the
// server-rendered SVG fallback uses.
type GraphJSON struct {
	Nodes []GraphJSONNode `json:"nodes"`
	Edges []GraphJSONEdge `json:"edges"`
}

// BuildCitationGraphJSON assembles the interactive-client graph payload from the
// project's exported citation graph (data/citation-graph.json).
func BuildCitationGraphJSON(projectPath string) (GraphJSON, error) {
	vm, err := buildCitationGraph(projectPath)
	if err != nil {
		return GraphJSON{}, err
	}
	graph := GraphJSON{
		Nodes: make([]GraphJSONNode, 0, len(vm.Nodes)),
		Edges: make([]GraphJSONEdge, 0, len(vm.Edges)),
	}
	for _, n := range vm.Nodes {
		stem := graphNodeStem(n.ID)
		graph.Nodes = append(graph.Nodes, GraphJSONNode{
			ID:    n.ID,
			Label: n.ID,
			Stem:  stem,
			Href:  "/papers/" + stem,
		})
	}
	for _, e := range vm.Edges {
		graph.Edges = append(graph.Edges, GraphJSONEdge{Source: e.Source, Target: e.Target})
	}
	return graph, nil
}

func newCitationGraphJSONHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		graph, err := BuildCitationGraphJSON(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(graph)
	})
}
