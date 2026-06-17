package webui

type DashboardInformationArchitecture struct {
	SchemaVersion       string              `json:"schemaVersion"`
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
	return DashboardInformationArchitecture{SchemaVersion: "1", Routes: []DashboardRoute{
		{"/forge", "/forge/refresh", "ForgeHomeState", "server-rendered timeline", "internal/webui adapts project/provenance state"},
		{"/workbenches", "/workbenches", "WorkbenchIndexState", "server-rendered workbench index", "internal/webui owns navigation only"},
		{"/sources", "/sources", "protocol.SourcePlan", "plain GET form", "internal/protocol owns source planning"},
		{"/dedupe", "/dedupe", "DedupeReviewState", "cluster tables", "internal/library owns identity decisions"},
		{"/screening", "/screening/refresh", "ScreeningCockpitState", "screening tables", "internal/screening owns decisions/rankers"},
		{"/evidence", "/evidence", "generic WorkbenchCard", "static CLI checklist", "internal/evidence owns evidence logic"},
		{"/analysis", "/analysis", "generic WorkbenchCard", "static CLI checklist", "internal/analysis owns statistics"},
		{"/report", "/report", "generic WorkbenchCard", "static CLI checklist", "internal/report owns claim gates"},
		{"/package", "/package", "generic WorkbenchCard", "static CLI checklist", "internal/reviewpkg owns package audit/replay"},
		{"/connectors", "/connectors", "connectorHealthView", "connector alert table", "internal/protocol owns connector registry"},
	}, BackgroundJobs: []DashboardJob{
		{"source import", "manual CLI or future HTMX post", "data/jobs.json"},
		{"parser run", "manual CLI or future HTMX post", "data/jobs.json"},
		{"package replay", "manual CLI or future HTMX post", "data/jobs.json"},
	}, OwnershipBoundaries: []OwnershipBoundary{
		{"templates", "render existing artifacts and never decide scientific state"},
		{"partials", "must expose CLI-equivalent command and no-JS fallback"},
		{"background jobs", "job state is project-local and replayable"},
		{"privacy", "local-only paths, credentials, PDFs, notes, embeddings, and caches stay behind package/redaction gates"},
	}}
}

func (ia DashboardInformationArchitecture) HasRoute(path string) bool {
	for _, route := range ia.Routes {
		if route.Path == path {
			return true
		}
	}
	return false
}
