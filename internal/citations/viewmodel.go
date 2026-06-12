package citations

// ResearchLineageViewModel is a UI-ready citation lineage summary.
type ResearchLineageViewModel struct {
	PaperID    string
	References []string
	CitedBy    []string
}

// NewResearchLineageViewModel summarizes backward and forward citation links for a paper.
func NewResearchLineageViewModel(graph *Graph, paperID string) ResearchLineageViewModel {
	return ResearchLineageViewModel{PaperID: paperID, References: graph.Backward(paperID), CitedBy: graph.Forward(paperID)}
}
