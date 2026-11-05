package core

// Graph neighborhood types shared by citation-graph connectors. They live with
// the budget types because adapters outside this folder consume them.

// SemanticScholarGraphDirection selects which graph neighborhood to fetch.
type SemanticScholarGraphDirection string

const (
	SemanticScholarDirectionReferences SemanticScholarGraphDirection = "references"
	SemanticScholarDirectionCitations  SemanticScholarGraphDirection = "citations"
	SemanticScholarDirectionBoth       SemanticScholarGraphDirection = "both"
)

// SemanticScholarGraphQuery describes a Semantic Scholar citation graph expansion.
type SemanticScholarGraphQuery struct {
	PaperID   string
	Limit     int
	Direction SemanticScholarGraphDirection
}

// CitationEdge is a normalized citing paper -> referenced paper relationship.
type CitationEdge struct {
	SourceID string
	TargetID string
}

// CitationGraphExpansion is a normalized Semantic Scholar graph neighborhood.
type CitationGraphExpansion struct {
	StopReason   StopReason `json:"stop_reason"`
	RequestsUsed int        `json:"requests_used"`
	SeedID       string
	Edges        []CitationEdge
	Records      map[string]SourceRecord
	RawRef       string
}
