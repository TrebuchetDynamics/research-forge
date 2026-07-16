package webui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/knowledge"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	reportpkg "github.com/TrebuchetDynamics/research-forge/internal/report"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

// BuildForgeHomeState assembles the Forge home timeline from project-local
// state, provenance, background jobs, blocked review gates, and next actions.
func BuildForgeHomeState(projectPath string) (ForgeHomeState, error) {
	state := ForgeHomeState{ActiveProject: projectPath, CurrentState: "question_draft"}
	if strings.TrimSpace(projectPath) == "" {
		state.BlockedReviewGates = []ForgeGate{{Gate: "project", Reason: "open or create a project"}}
		state.NextSafeActions = []ForgeNextAction{{Label: "Create project", CLI: "rforge project create <path> --title <title>"}}
		return state, nil
	}
	if proj, err := project.Inspect(projectPath); err == nil {
		state.ProjectTitle = proj.Title
	}
	var stored struct {
		CurrentState string `json:"currentState"`
		State        string `json:"state"`
	}
	if data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "forge-state.json")); err == nil {
		_ = json.Unmarshal(data, &stored)
		if stored.CurrentState != "" {
			state.CurrentState = stored.CurrentState
		} else if stored.State != "" {
			state.CurrentState = stored.State
		}
	}
	if events, err := provenance.Read(projectPath); err == nil {
		state.ProvenanceEvents = events
	}
	if data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "jobs.json")); err == nil {
		_ = json.Unmarshal(data, &state.BackgroundJobs)
	}
	state.BlockedReviewGates = forgeBlockedGates(projectPath, state.CurrentState)
	state.NextSafeActions = forgeNextActions(state.CurrentState)
	return state, nil
}

func forgeBlockedGates(projectPath, currentState string) []ForgeGate {
	gates := []ForgeGate{}
	missing := func(rel string) bool {
		return !forgeArtifactExists(projectPath, rel)
	}
	switch currentState {
	case "source_plan":
		if missing("data/source-plans") && missing("data/source-plan.json") {
			gates = append(gates, ForgeGate{Gate: "network/API approval", Reason: "source plan artifact or approval is missing"})
		}
	case "screening":
		if missing("data/screening-audit-bundle.json") {
			gates = append(gates, ForgeGate{Gate: "screening approval", Reason: "screening audit bundle is missing"})
		}
	case "report_build":
		if missing("data/claim-panel.json") {
			gates = append(gates, ForgeGate{Gate: "claim approval", Reason: "claim traceability panel is missing"})
		}
	case "package_export":
		if missing("review.rforgepkg") {
			gates = append(gates, ForgeGate{Gate: "package approval", Reason: "review package has not been exported"})
		}
	}
	return gates
}

func forgeArtifactExists(projectPath, rel string) bool {
	name := filepath.FromSlash(rel)
	file, err := os.OpenInRoot(projectPath, name)
	if err != nil {
		return false
	}
	openedInfo, statErr := file.Stat()
	visibleInfo, lstatErr := os.Lstat(filepath.Join(projectPath, name))
	closeErr := file.Close()
	if statErr != nil || lstatErr != nil || closeErr != nil {
		return false
	}
	return (openedInfo.Mode().IsRegular() || openedInfo.IsDir()) && os.SameFile(openedInfo, visibleInfo)
}

func forgeNextActions(currentState string) []ForgeNextAction {
	switch currentState {
	case "question_draft":
		return []ForgeNextAction{{Label: "Compile protocol", CLI: "rforge protocol compile --type pico --question <question>"}}
	case "source_plan":
		return []ForgeNextAction{{Label: "Preview source plan", CLI: "rforge protocol plan-sources --question <question>"}, {Label: "Record source approval", CLI: "rforge forge approve --gate source_plan"}}
	case "screening":
		return []ForgeNextAction{{Label: "Review screening progress", CLI: "rforge screen progress --stage title_abstract"}, {Label: "Export screening audit", CLI: "rforge screen audit-bundle --stage title_abstract --out data/screening-audit-bundle.json"}}
	case "report_build":
		return []ForgeNextAction{{Label: "Build claim panel", CLI: "rforge report claim-panel --trace data/trace.json --out data/claim-panel.json"}}
	case "package_export":
		return []ForgeNextAction{{Label: "Create review package", CLI: "rforge package create --out review.rforgepkg"}, {Label: "Audit package", CLI: "rforge package audit review.rforgepkg"}}
	default:
		return []ForgeNextAction{{Label: "Inspect project", CLI: "rforge project inspect <path>"}}
	}
}

type PackageExportCenterState struct {
	ProjectPath          string
	PackageContents      []string
	RedactionResults     []string
	Checksums            []string
	Lockfiles            []string
	ExternalToolVersions []string
	ParserManifests      []string
	AnalysisArtifacts    []string
	ReportOutputs        []string
	ReviewerDecisionLogs []string
}

func BuildPackageExportCenterState(projectPath string) PackageExportCenterState {
	state := PackageExportCenterState{ProjectPath: projectPath, RedactionResults: []string{"documents/ excluded until shareability approval", "cache/ excluded as private local state"}, Checksums: []string{"checksums.sha256 preview for copied package files"}, ExternalToolVersions: []string{"rforge.lock.json and data/*.lock.json capture external-tool versions"}}
	if strings.TrimSpace(projectPath) == "" {
		return state
	}
	for _, rel := range []string{"rforge.project.toml", "rforge.lock.json", "data/provenance.jsonl", "data/forge-state.json", "data/connector-capabilities.json", "data/evidence.schemas.json", "data/evidence.items.json", "data/claim-trace.json"} {
		file, err := filetxn.OpenRegularInRoot(projectPath, filepath.FromSlash(rel))
		if err != nil {
			continue
		}
		if err := file.Close(); err != nil {
			continue
		}
		state.PackageContents = append(state.PackageContents, rel)
	}
	state.Lockfiles = appendExistingGlobs(projectPath, state.Lockfiles, "rforge.lock.json", "data/*.lock.json")
	state.ParserManifests = appendExistingGlobs(projectPath, state.ParserManifests, "data/parser-manifests/*")
	state.AnalysisArtifacts = appendExistingGlobs(projectPath, state.AnalysisArtifacts, "analysis/*")
	state.ReportOutputs = appendExistingGlobs(projectPath, state.ReportOutputs, "reports/*")
	state.ReviewerDecisionLogs = appendExistingGlobs(projectPath, state.ReviewerDecisionLogs, "data/identity-decisions.jsonl", "data/screening-audit.jsonl", "data/reviewer-decisions.jsonl")
	return state
}

func appendExistingGlobs(projectPath string, out []string, patterns ...string) []string {
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, filepath.FromSlash(pattern)))
		sort.Strings(matches)
		for _, match := range matches {
			rel, err := filepath.Rel(projectPath, match)
			if err != nil {
				continue
			}
			file, err := filetxn.OpenRegularInRoot(projectPath, rel)
			if err != nil {
				continue
			}
			if err := file.Close(); err != nil {
				continue
			}
			out = append(out, filepath.ToSlash(rel))
		}
	}
	return out
}

type ReportClaimPanelState struct {
	ProjectPath      string
	PanelPath        string
	Rows             []reportpkg.ClaimTracePanelRow
	BlockFinalExport bool
	Blockers         []string
}

func BuildReportClaimPanelState(projectPath string) ReportClaimPanelState {
	state := ReportClaimPanelState{ProjectPath: projectPath, PanelPath: "data/claim-panel.json"}
	if strings.TrimSpace(projectPath) == "" {
		return state
	}
	var panel reportpkg.ClaimTraceabilityPanel
	if data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "claim-panel.json")); err == nil {
		_ = json.Unmarshal(data, &panel)
		state.Rows = panel.Rows
		state.BlockFinalExport = panel.BlockFinalExport
		state.Blockers = panel.Blockers
	}
	return state
}

type AnalysisWorkbenchState struct {
	ProjectPath string
	Runs        []analysis.AnalysisRun
	Manifests   []analysis.AnalysisArtifactManifest
}

func BuildAnalysisWorkbenchState(projectPath string) AnalysisWorkbenchState {
	state := AnalysisWorkbenchState{ProjectPath: projectPath}
	if strings.TrimSpace(projectPath) == "" {
		return state
	}
	matches, _ := filepath.Glob(filepath.Join(projectPath, "analysis", "*.json"))
	for _, match := range matches {
		data, err := filetxn.ReadRegular(match)
		if err != nil {
			continue
		}
		var manifest analysis.AnalysisArtifactManifest
		if json.Unmarshal(data, &manifest) == nil && manifest.RunID != "" && (len(manifest.Plots) > 0 || manifest.Script.Path != "") {
			state.Manifests = append(state.Manifests, manifest)
			continue
		}
		var run analysis.AnalysisRun
		if json.Unmarshal(data, &run) == nil && run.ID != "" {
			state.Runs = append(state.Runs, run)
		}
	}
	return state
}

type EvidenceGridState struct {
	ProjectPath string
	GridPath    string
	Rows        []evidence.ExtractionGridRow
}

func BuildEvidenceGridState(projectPath string) EvidenceGridState {
	state := EvidenceGridState{ProjectPath: projectPath, GridPath: "data/evidence-grid.json"}
	if strings.TrimSpace(projectPath) == "" {
		return state
	}
	var grid evidence.ExtractionGrid
	if data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "evidence-grid.json")); err == nil {
		_ = json.Unmarshal(data, &grid)
		state.Rows = grid.Rows
	}
	return state
}

type RetrievalTuningState struct {
	ProjectPath          string
	Query                string
	Backends             []retrieval.RetrievalBackendBenchmark
	QueryResults         []retrieval.RetrievalBenchmarkQueryResult
	PrivacyNotes         []string
	ReproducibilityNotes []string
	CLIEquivalent        string
}

func BuildRetrievalTuningState(projectPath string) RetrievalTuningState {
	fixture := retrieval.DefaultRetrievalBenchmarkFixture()
	report, _ := retrieval.RunRetrievalBenchmark(fixture, 3)
	state := RetrievalTuningState{ProjectPath: projectPath, Query: "same query fixture set", Backends: report.Backends, QueryResults: report.QueryResults, PrivacyNotes: report.PrivacyNotes, ReproducibilityNotes: report.ReproducibilityNotes, CLIEquivalent: "rforge retrieve benchmark --out data/retrieval-benchmark.json && rforge retrieve tune-hybrid --queries queries.json --lexical lexical.json --vector vector.json --out data/hybrid-tuning.json"}
	return state
}

type AcquisitionQueueState struct {
	ProjectPath string
	QueuePath   string
	Items       []documents.LegalAcquisitionQueueItem
}

func BuildAcquisitionQueueState(projectPath string) AcquisitionQueueState {
	state := AcquisitionQueueState{ProjectPath: projectPath, QueuePath: "data/legal-acquisition-queue.json"}
	if strings.TrimSpace(projectPath) == "" {
		return state
	}
	var queue documents.LegalAcquisitionQueue
	if data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "legal-acquisition-queue.json")); err == nil {
		_ = json.Unmarshal(data, &queue)
		state.Items = queue.Items
	}
	return state
}

// BuildLibraryViewModel reads a CLI-generated project's library into the cockpit
// library view model. A project without a library yields an empty view model
// and is not treated as an error.
func BuildLibraryViewModel(projectPath string) (ui.LibraryViewModel, error) {
	if strings.TrimSpace(projectPath) == "" {
		return ui.NewLibraryViewModel(nil), nil
	}
	libPath := filepath.Join(projectPath, "data", "library.json")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return ui.NewLibraryViewModel(nil), nil
	}
	store, err := library.OpenStore(libPath)
	if err != nil {
		return ui.LibraryViewModel{}, err
	}
	papers, err := store.List()
	if err != nil {
		return ui.LibraryViewModel{}, err
	}
	rows := make([]ui.PaperRow, 0, len(papers))
	for _, paper := range papers {
		rows = append(rows, ui.PaperRow{Title: paper.Title})
	}
	return ui.NewLibraryViewModel(rows), nil
}

// DedupeReviewState powers the visual identity-cluster review screen.
type DedupeReviewState struct {
	ProjectPath string
	Records     []library.PaperRecord
	Clusters    []library.IdentityCluster
	Conflicts   []library.IdentityConflictRecord
	DecisionLog library.IdentityDecisionLog
	PRISMA      PRISMAFlowState
	AuditEvents []provenance.Event
}

// BuildDedupeReviewState reads identity clusters, reversible decision history,
// conflict records, PRISMA counts, and audit provenance for the dedupe cockpit.
func BuildDedupeReviewState(projectPath string) (DedupeReviewState, error) {
	state := DedupeReviewState{ProjectPath: projectPath, DecisionLog: library.IdentityDecisionLog{SchemaVersion: "1"}}
	if strings.TrimSpace(projectPath) == "" {
		return state, nil
	}
	libPath := filepath.Join(projectPath, "data", "library.json")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return state, nil
	}
	store, err := library.OpenStore(libPath)
	if err != nil {
		return state, err
	}
	records, err := store.List()
	if err != nil {
		return state, err
	}
	state.Records = records
	report := library.ResolveIdentityClusters(records)
	state.Clusters = report.Clusters
	state.Conflicts = library.DetectIdentityConflicts(report, records)
	if log, err := library.ReadIdentityDecisionLog(filepath.Join(projectPath, "data", "identity-decisions.jsonl")); err == nil {
		state.DecisionLog = log
		state.Conflicts = append(state.Conflicts, log.Conflicts...)
	}
	prisma, err := buildPRISMAFlowState(projectPath, len(records))
	if err != nil {
		return state, err
	}
	state.PRISMA = prisma
	if events, err := provenance.Read(projectPath); err == nil {
		for _, event := range events {
			if strings.HasPrefix(event.Action, "identity.") || strings.HasPrefix(event.Action, "duplicate.") {
				state.AuditEvents = append(state.AuditEvents, event)
			}
		}
	}
	return state, nil
}

const screeningAuditBundlePath = "data/screening-audit-bundle.json"

func openScreeningAuditBundle(projectPath string) (*os.File, bool) {
	file, err := filetxn.OpenRegularInRoot(projectPath, filepath.FromSlash(screeningAuditBundlePath))
	if err != nil {
		return nil, false
	}
	reject := func() (*os.File, bool) {
		_ = file.Close()
		return nil, false
	}
	var bundle screening.ScreeningAuditBundle
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&bundle); err != nil || bundle.SchemaVersion != "1" || bundle.Stage == "" {
		return reject()
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return reject()
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return reject()
	}
	return file, true
}

func normalizeAndValidateScreeningEvents(projectPath string, events []screening.DecisionEvent) error {
	if len(events) == 0 {
		return nil
	}
	workflowData, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "screening.workflow.json"))
	if err != nil {
		return fmt.Errorf("read screening workflow: %w", err)
	}
	var workflow screening.Workflow
	if err := json.Unmarshal(workflowData, &workflow); err != nil {
		return fmt.Errorf("parse screening workflow: %w", err)
	}
	store := screening.NewMemoryStore(workflow)
	for i := range events {
		events[i] = screening.NormalizeDecisionEvent(events[i])
		event := events[i]
		if err := store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer, Adjudicated: event.Adjudicated}); err != nil {
			return fmt.Errorf("validate screening event %d: %w", i+1, err)
		}
	}
	return nil
}

// BuildScreeningCockpitState reads screening decisions and library records into
// the HTMX screening cockpit: active-learning queue, uncertainty/exploration
// flags, progress metrics, stopping diagnostics, and audit-bundle links.
func BuildScreeningCockpitState(projectPath string) (ScreeningCockpitState, error) {
	state := ScreeningCockpitState{ProjectPath: projectPath, Stage: screening.StageTitleAbstract, AuditBundlePath: screeningAuditBundlePath}
	if strings.TrimSpace(projectPath) == "" {
		return state, nil
	}
	records, err := buildScreeningRecords(projectPath)
	if err != nil {
		return state, err
	}
	state.TotalRecords = len(records)
	var events []screening.DecisionEvent
	if data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "screening.events.json")); err == nil {
		if err := json.Unmarshal(data, &events); err != nil {
			return state, fmt.Errorf("parse screening events: %w", err)
		}
	}
	if err := normalizeAndValidateScreeningEvents(projectPath, events); err != nil {
		return state, err
	}
	run, err := screening.BuildActiveLearningRun(screening.ActiveLearningRunInput{Records: records, Events: events, Stage: state.Stage, RankingMethod: "active-learning", TargetRecall: 0.95})
	if err == nil {
		state.ActiveLearningQueue = run.RankedOutput
		state.ActiveRunID = run.RunID
	}
	state.UncertaintyQueue = screening.PrioritizeUncertaintyRecords(records, events, state.Stage)
	state.UncertainQueue = screening.UncertainQueue(events, state.Stage)
	state.Progress = screening.Progress(events, state.Stage, len(records))
	state.Stopping = screening.StoppingCriteria(events, state.Stage, 0.95, len(records))
	if file, ok := openScreeningAuditBundle(projectPath); ok {
		if err := file.Close(); err == nil {
			state.HasAuditBundle = true
		}
	}
	return state, nil
}

func buildScreeningRecords(projectPath string) ([]screening.ScreeningRecord, error) {
	libPath := filepath.Join(projectPath, "data", "library.json")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return nil, nil
	}
	store, err := library.OpenStore(libPath)
	if err != nil {
		return nil, err
	}
	papers, err := store.List()
	if err != nil {
		return nil, err
	}
	records := make([]screening.ScreeningRecord, 0, len(papers))
	for _, paper := range papers {
		records = append(records, screening.ScreeningRecord{ID: webScreeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
	}
	return records, nil
}

func webScreeningPaperID(paper library.PaperRecord) string {
	if strings.TrimSpace(paper.Identifiers.DOI) != "" {
		return paper.Identifiers.DOI
	}
	if strings.TrimSpace(paper.Identifiers.PMID) != "" {
		return paper.Identifiers.PMID
	}
	if strings.TrimSpace(paper.Identifiers.ArXivID) != "" {
		return paper.Identifiers.ArXivID
	}
	return paper.Title
}

// BuildArtifactDashboardState assembles the artifacts cockpit view from a
// CLI-generated project workspace: imported papers, screening-derived PRISMA
// counts, and meta-analysis readiness.
func BuildArtifactDashboardState(projectPath string) (ArtifactDashboardState, error) {
	papers, err := BuildLibraryViewModel(projectPath)
	if err != nil {
		return ArtifactDashboardState{}, err
	}
	prisma, err := buildPRISMAFlowState(projectPath, len(papers.Rows))
	if err != nil {
		return ArtifactDashboardState{}, err
	}
	graph, err := buildCitationGraph(projectPath)
	if err != nil {
		return ArtifactDashboardState{}, err
	}
	analysisDetail := buildAnalysisDetail(projectPath)
	return ArtifactDashboardState{
		Papers:         papers,
		PRISMA:         prisma,
		Analysis:       ui.NewAnalysisViewModel(analysisDetail.RunID, analysisDetail.Ready),
		AnalysisDetail: analysisDetail,
		CitationGraph:  graph,
	}, nil
}

// buildCitationGraph loads the project's exported citation graph
// (data/citation-graph.json, the stable nodes/edges export format) into the
// cockpit citation-graph view model. A project without a graph yields an empty,
// non-error model.
func buildCitationGraph(projectPath string) (ui.CitationGraphViewModel, error) {
	if strings.TrimSpace(projectPath) == "" {
		return ui.CitationGraphViewModel{}, nil
	}
	if graph, err := knowledge.ReadProjectKnowledgeGraphArtifact(projectPath); err == nil {
		return citationGraphFromKnowledgeGraph(graph), nil
	} else if !os.IsNotExist(err) {
		return ui.CitationGraphViewModel{}, err
	}
	data, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "citation-graph.json"))
	if os.IsNotExist(err) {
		return ui.CitationGraphViewModel{}, nil
	}
	if err != nil {
		return ui.CitationGraphViewModel{}, err
	}
	var export struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return ui.CitationGraphViewModel{}, err
	}
	nodes := make([]ui.GraphNode, 0, len(export.Nodes))
	for _, n := range export.Nodes {
		nodes = append(nodes, ui.GraphNode{ID: n.ID})
	}
	edges := make([]ui.GraphEdge, 0, len(export.Edges))
	for _, e := range export.Edges {
		edges = append(edges, ui.GraphEdge{Source: e.Source, Target: e.Target})
	}
	return ui.NewCitationGraphViewModel(nodes, edges), nil
}

func citationGraphFromKnowledgeGraph(graph knowledge.ProjectKnowledgeGraph) ui.CitationGraphViewModel {
	nodeIDs := map[string]bool{}
	for _, node := range graph.Nodes {
		if node.Kind == "paper" {
			nodeIDs[strings.TrimPrefix(node.ID, "paper:")] = true
		}
	}
	edges := []ui.GraphEdge{}
	for _, edge := range graph.Edges {
		if edge.Kind != "cites" {
			continue
		}
		source := strings.TrimPrefix(edge.Source, "paper:")
		target := strings.TrimPrefix(edge.Target, "paper:")
		nodeIDs[source] = true
		nodeIDs[target] = true
		edges = append(edges, ui.GraphEdge{Source: source, Target: target})
	}
	nodes := make([]ui.GraphNode, 0, len(nodeIDs))
	for id := range nodeIDs {
		nodes = append(nodes, ui.GraphNode{ID: id})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source == edges[j].Source {
			return edges[i].Target < edges[j].Target
		}
		return edges[i].Source < edges[j].Source
	})
	return ui.NewCitationGraphViewModel(nodes, edges)
}

// buildAnalysisDetail loads the project's stored meta-analysis result
// (analysis/<run>-result.json) into a readable detail view: heterogeneity
// metrics, plot availability, and any runner warnings. A project without a
// stored result yields a not-ready detail.
func buildAnalysisDetail(projectPath string) AnalysisDetail {
	if strings.TrimSpace(projectPath) == "" {
		return AnalysisDetail{}
	}
	dir := filepath.Join(projectPath, "analysis")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return AnalysisDetail{}
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-result.json") {
			continue
		}
		data, err := filetxn.ReadRegular(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var result analysis.AnalysisResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		return AnalysisDetail{
			Ready:         true,
			RunID:         strings.TrimSuffix(entry.Name(), "-result.json"),
			I2:            result.Metrics.I2,
			Tau2:          result.Metrics.Tau2,
			Q:             result.Metrics.Q,
			HasForestPlot: strings.TrimSpace(result.ForestPlot.Path) != "",
			HasFunnelPlot: strings.TrimSpace(result.FunnelPlot.Path) != "",
			Warnings:      result.Warnings,
		}
	}
	return AnalysisDetail{}
}

// buildPRISMAFlowState replays the project's stored screening decisions into
// PRISMA flow counts. A project without a screening workflow yields just the
// record count.
func buildPRISMAFlowState(projectPath string, libraryCount int) (PRISMAFlowState, error) {
	state := PRISMAFlowState{Records: libraryCount}
	if strings.TrimSpace(projectPath) == "" {
		return state, nil
	}
	workflowData, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "screening.workflow.json"))
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	var workflow screening.Workflow
	if err := json.Unmarshal(workflowData, &workflow); err != nil {
		return state, err
	}
	var events []screening.DecisionEvent
	if eventsData, err := filetxn.ReadRegular(filepath.Join(projectPath, "data", "screening.events.json")); err == nil {
		if err := json.Unmarshal(eventsData, &events); err != nil {
			return state, fmt.Errorf("parse screening events: %w", err)
		}
	}
	store := screening.NewMemoryStore(workflow)
	decided := map[string]bool{}
	for i, event := range events {
		event = screening.NormalizeDecisionEvent(event)
		if err := store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer}); err != nil {
			return state, fmt.Errorf("apply screening event %d: %w", i+1, err)
		}
		decided[event.PaperID] = true
	}
	state.Screened = len(decided)
	state.Included = store.PRISMACounts().Included
	return state, nil
}
