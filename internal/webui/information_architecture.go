package webui

type DashboardInformationArchitecture struct {
	SchemaVersion       string              `json:"schemaVersion"`
	Diagram             []string            `json:"diagram"`
	Routes              []DashboardRoute    `json:"routes"`
	BackgroundJobs      []DashboardJob      `json:"backgroundJobs"`
	OwnershipBoundaries []OwnershipBoundary `json:"ownershipBoundaries"`
}

type DashboardRoute struct {
	Path         string `json:"path"`
	Partial      string `json:"partial"`
	ViewModel    string `json:"viewModel"`
	NoJSFallback string `json:"noJsFallback"`
	Owner        string `json:"owner"`
}

type DashboardJob struct{ Name, Trigger, StatusArtifact string }
type OwnershipBoundary struct{ Area, Boundary string }

func BuildDashboardInformationArchitecture() DashboardInformationArchitecture {
	routes := []DashboardRoute{
		{"/forge", "/forge/refresh", "ForgeHomeState", "server-rendered timeline and CLI action list", "internal/webui adapts project/provenance state"},
		{"/workbenches", "/workbenches", "WorkbenchIndexState", "server-rendered workbench index", "internal/webui owns navigation only"},
		{"/sources", "/sources", "protocol.SourcePlan", "plain GET form", "internal/protocol owns source planning"},
		{"/notebook", "/notebook/snapshot.json", "LabNotebookTimelineState", "ordered event list plus JSON snapshot", "internal/provenance owns workflow events"},
		{"/parsing", "/parsing", "ParserConflictReviewState", "field-by-field parser comparison table", "internal/parsing owns parser arbitration"},
		{"/map", "/map/snapshot.json", "ResearchMapCockpitState", "no-JS concept/citation/retrieval/evidence tables", "internal/knowledge owns graph construction"},
		{"/acquisition", "/acquisition", "AcquisitionQueueState", "legal acquisition queue table", "internal/documents owns acquisition policy"},
		{"/retrieve", "/retrieve", "RetrievalTuningState", "backend comparison tables", "internal/retrieval owns ranking and benchmarks"},
		{"/dedupe", "/dedupe", "DedupeReviewState", "cluster tables", "internal/library owns identity decisions"},
		{"/screening", "/screening/refresh", "ScreeningCockpitState", "screening priority/conflict/diagnostic tables", "internal/screening owns decisions/rankers"},
		{"/evidence", "/evidence", "EvidenceGridState", "evidence extraction grid", "internal/evidence owns evidence logic"},
		{"/analysis", "/analysis", "AnalysisWorkbenchState", "meta-analysis artifact and diagnostics tables", "internal/analysis owns statistics"},
		{"/report", "/report", "ReportClaimPanelState", "claim traceability table", "internal/report owns claim gates"},
		{"/package", "/package", "PackageExportCenterState", "package preview tables", "internal/reviewpkg owns package audit/replay"},
		{"/connectors", "/connectors", "connectorHealthView", "connector alert and capability tables", "internal/protocol owns connector registry"},
		{"/privacy", "/privacy", "DashboardPrivacyModel", "privacy classification tables", "internal/webui documents policy only"},
		{"/architecture", "/architecture", "DashboardInformationArchitecture", "this route/ownership diagram", "internal/webui documents dashboard boundaries"},
	}
	return DashboardInformationArchitecture{SchemaVersion: "1", Diagram: dashboardDiagram(routes), Routes: routes, BackgroundJobs: []DashboardJob{
		{"source import", "manual CLI or future HTMX post", "data/jobs.json"},
		{"source live-smoke", "opt-in CLI", "data/source-live-smoke-snapshots/latest.json"},
		{"parser run", "manual CLI or future HTMX post", "data/jobs.json and data/parser-manifests/*"},
		{"retrieval benchmark", "manual CLI", "data/retrieval-benchmark.json"},
		{"analysis run", "manual CLI", "analysis/*"},
		{"package replay", "manual CLI or future HTMX post", "data/jobs.json"},
	}, OwnershipBoundaries: []OwnershipBoundary{
		{"templates", "render existing artifacts and never decide scientific state"},
		{"partials", "must expose CLI-equivalent command and no-JS fallback"},
		{"background jobs", "job state is project-local and replayable"},
		{"domain logic", "CLI/internal packages own research, screening, evidence, analysis, report, and package semantics"},
		{"privacy", "local-only paths, credentials, PDFs, notes, embeddings, and caches stay behind package/redaction gates"},
	}}
}

func dashboardDiagram(routes []DashboardRoute) []string {
	lines := []string{"flowchart TD", "  CLI[rforge CLI artifacts] --> Web[local HTMX dashboard]"}
	for _, route := range routes {
		lines = append(lines, "  Web --> "+route.Path+"[\""+route.Path+" / "+route.ViewModel+"\"]")
	}
	lines = append(lines, "  Web --> Jobs[data/jobs.json background job status]", "  Web --> Privacy[package/redaction privacy gates]")
	return lines
}

func (ia DashboardInformationArchitecture) HasRoute(path string) bool {
	for _, route := range ia.Routes {
		if route.Path == path {
			return true
		}
	}
	return false
}
