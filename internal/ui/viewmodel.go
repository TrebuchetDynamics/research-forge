package ui

type DashboardState struct {
	Title        string
	LibraryCount int
	Loading      bool
	Error        string
}

func NewDashboardState(title string, libraryCount int) DashboardState {
	return DashboardState{Title: title, LibraryCount: libraryCount}
}

type SearchFormState struct {
	Sources []string
	Query   string
	Loading bool
	Error   string
}

func NewSearchFormState(sources []string) SearchFormState { return SearchFormState{Sources: sources} }

type PaperRow struct{ Title string }
type LibraryViewModel struct {
	Rows  []PaperRow
	Empty bool
}

func NewLibraryViewModel(rows []PaperRow) LibraryViewModel {
	return LibraryViewModel{Rows: rows, Empty: len(rows) == 0}
}

type OSSRow struct{ Name string }
type OSSDashboardViewModel struct{ Repositories []OSSRow }

func NewOSSDashboardViewModel(rows []OSSRow) OSSDashboardViewModel {
	return OSSDashboardViewModel{Repositories: rows}
}

type GraphNode struct{ ID string }
type GraphEdge struct {
	Source string
	Target string
}
type CitationGraphViewModel struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

func NewCitationGraphViewModel(nodes []GraphNode, edges []GraphEdge) CitationGraphViewModel {
	return CitationGraphViewModel{Nodes: nodes, Edges: edges}
}

type ScreeningRow struct{ PaperID string }
type ScreeningViewModel struct{ Rows []ScreeningRow }

func NewScreeningViewModel(rows []ScreeningRow) ScreeningViewModel {
	return ScreeningViewModel{Rows: rows}
}

type EvidenceRow struct {
	PaperID    string
	SupportRef string
}
type EvidenceViewModel struct{ Rows []EvidenceRow }

func NewEvidenceViewModel(rows []EvidenceRow) EvidenceViewModel { return EvidenceViewModel{Rows: rows} }

type AnalysisViewModel struct {
	RunID string
	Ready bool
}

func NewAnalysisViewModel(runID string, ready bool) AnalysisViewModel {
	return AnalysisViewModel{RunID: runID, Ready: ready}
}

type ReportViewModel struct{ Formats []string }

func NewReportViewModel(formats []string) ReportViewModel { return ReportViewModel{Formats: formats} }
