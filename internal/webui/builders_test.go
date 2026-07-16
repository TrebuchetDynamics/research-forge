package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// seedProject writes the canonical CLI project layout (library, screening, and
// analysis result) used by the cockpit builders.
func seedProject(t *testing.T) string {
	t.Helper()
	proj := t.TempDir()
	store, err := library.OpenStore(filepath.Join(proj, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: "Artificial photosynthesis catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/ap-1"}}); err != nil {
		t.Fatalf("seed paper: %v", err)
	}
	workflow, err := screening.Configure(screening.Options{ExclusionReasons: []string{"off-topic"}})
	if err != nil {
		t.Fatalf("configure screening: %v", err)
	}
	writeJSON(t, filepath.Join(proj, "data", "screening.workflow.json"), workflow)
	writeJSON(t, filepath.Join(proj, "data", "screening.events.json"), []screening.DecisionEvent{
		{PaperID: "10.1000/ap-1", Stage: screening.StageTitleAbstract, Decision: screening.DecisionInclude, Reviewer: "ada"},
	})
	writeJSON(t, filepath.Join(proj, "analysis", "run1-result.json"), analysis.AnalysisResult{Versions: map[string]string{"R": "fake"}})
	return proj
}

func TestBuildLibraryViewModelReadsProjectLibrary(t *testing.T) {
	proj := seedProject(t)
	vm, err := BuildLibraryViewModel(proj)
	if err != nil {
		t.Fatalf("BuildLibraryViewModel: %v", err)
	}
	if vm.Empty {
		t.Fatalf("library view model marked empty for populated project")
	}
	if len(vm.Rows) != 1 || vm.Rows[0].Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("rows = %#v, want one paper title", vm.Rows)
	}
}

func TestBuildLibraryViewModelEmptyWhenNoLibrary(t *testing.T) {
	vm, err := BuildLibraryViewModel(t.TempDir())
	if err != nil {
		t.Fatalf("BuildLibraryViewModel on empty project: %v", err)
	}
	if !vm.Empty || len(vm.Rows) != 0 {
		t.Fatalf("expected empty library view model, got %#v", vm)
	}
}

func TestBuildArtifactDashboardStateFromProject(t *testing.T) {
	proj := seedProject(t)
	state, err := BuildArtifactDashboardState(proj)
	if err != nil {
		t.Fatalf("BuildArtifactDashboardState: %v", err)
	}
	if len(state.Papers.Rows) != 1 || state.Papers.Rows[0].Title != "Artificial photosynthesis catalyst review" {
		t.Fatalf("papers = %#v, want one paper", state.Papers.Rows)
	}
	if state.PRISMA.Included != 1 {
		t.Fatalf("PRISMA.Included = %d, want 1", state.PRISMA.Included)
	}
	if state.PRISMA.Records != 1 {
		t.Fatalf("PRISMA.Records = %d, want 1 library record", state.PRISMA.Records)
	}
	if !state.Analysis.Ready || state.Analysis.RunID != "run1" {
		t.Fatalf("analysis = %+v, want ready run1", state.Analysis)
	}
}

func TestBuildArtifactDashboardStateDoesNotReadSymlinkedScreeningEvents(t *testing.T) {
	proj := seedProject(t)
	eventsPath := filepath.Join(proj, "data", "screening.events.json")
	if err := os.Remove(eventsPath); err != nil {
		t.Fatalf("remove project screening events fixture: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-screening-events.json")
	writeJSON(t, outsidePath, []screening.DecisionEvent{{
		PaperID: "10.1000/ap-1", Stage: screening.StageTitleAbstract,
		Decision: screening.DecisionInclude, Reviewer: "external-private-reviewer",
	}})
	if err := os.Symlink(outsidePath, eventsPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	state, err := BuildArtifactDashboardState(proj)
	if err != nil {
		t.Fatalf("BuildArtifactDashboardState: %v", err)
	}
	if state.PRISMA.Screened != 0 || state.PRISMA.Included != 0 {
		t.Fatalf("BuildArtifactDashboardState exposed external screening events: %#v", state.PRISMA)
	}
	info, err := os.Lstat(eventsPath)
	if err != nil {
		t.Fatalf("lstat screening events: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("artifact dashboard read replaced symlink: mode=%v", info.Mode())
	}
}

func TestArtifactDashboardFailsClosedOnMalformedScreeningEvents(t *testing.T) {
	proj := seedProject(t)
	eventsPath := filepath.Join(proj, "data", "screening.events.json")
	if err := os.WriteFile(eventsPath, []byte("["), 0o644); err != nil {
		t.Fatalf("write malformed screening events: %v", err)
	}

	if _, err := BuildArtifactDashboardState(proj); err == nil {
		t.Fatal("BuildArtifactDashboardState accepted malformed screening events")
	}
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/artifacts")
	if status != http.StatusInternalServerError {
		t.Fatalf("GET /artifacts with malformed events status = %d: %s", status, body)
	}
	if !strings.Contains(body, "parse screening events") {
		t.Fatalf("malformed artifacts response lacks parse context: %s", body)
	}
}

func TestArtifactDashboardFailsClosedOnInvalidScreeningDecision(t *testing.T) {
	proj := seedProject(t)
	writeJSON(t, filepath.Join(proj, "data", "screening.events.json"), []screening.DecisionEvent{
		{
			PaperID:  "10.1000/ap-1",
			Stage:    screening.StageTitleAbstract,
			Decision: screening.DecisionExclude,
			Reason:   "not-configured",
			Reviewer: "ada",
		},
	})

	if state, err := BuildArtifactDashboardState(proj); err == nil {
		t.Fatalf("BuildArtifactDashboardState accepted invalid screening decision: %#v", state.PRISMA)
	}
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/artifacts")
	if status != http.StatusInternalServerError {
		t.Fatalf("GET /artifacts with invalid screening decision status = %d: %s", status, body)
	}
	if !strings.Contains(body, "apply screening event") {
		t.Fatalf("invalid screening decision response lacks event context: %s", body)
	}
}

func TestBuildArtifactDashboardStateRejectsSymlinkedScreeningWorkflow(t *testing.T) {
	proj := seedProject(t)
	workflowPath := filepath.Join(proj, "data", "screening.workflow.json")
	if err := os.Remove(workflowPath); err != nil {
		t.Fatalf("remove project screening workflow fixture: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-screening-workflow.json")
	externalWorkflow, err := screening.Configure(screening.Options{ExclusionReasons: []string{"external-private-reason"}})
	if err != nil {
		t.Fatalf("configure external workflow: %v", err)
	}
	writeJSON(t, outsidePath, externalWorkflow)
	if err := os.Symlink(outsidePath, workflowPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if state, err := BuildArtifactDashboardState(proj); err == nil {
		t.Fatalf("BuildArtifactDashboardState accepted external screening workflow: %#v", state.PRISMA)
	}
	info, err := os.Lstat(workflowPath)
	if err != nil {
		t.Fatalf("lstat screening workflow: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("artifact dashboard workflow read replaced symlink: mode=%v", info.Mode())
	}
}

// TestE2EWebCockpitServesProjectViewModelsThroughHandlers ties CLI-produced
// project state to the cockpit: it builds the library and artifacts view models
// from a project workspace on disk and serves them through the internal/webui
// handlers, asserting the rendered HTML surfaces the project's papers, PRISMA
// counts, and meta-analysis readiness.
func TestE2EWebCockpitServesProjectViewModelsThroughHandlers(t *testing.T) {
	proj := seedProject(t)

	libVM, err := BuildLibraryViewModel(proj)
	if err != nil {
		t.Fatalf("BuildLibraryViewModel: %v", err)
	}
	libBody := renderHandler(t, NewLibraryHandler(libVM))
	if !strings.Contains(libBody, "Artificial photosynthesis catalyst review") {
		t.Fatalf("library page missing imported paper:\n%s", libBody)
	}

	state, err := BuildArtifactDashboardState(proj)
	if err != nil {
		t.Fatalf("BuildArtifactDashboardState: %v", err)
	}
	artBody := renderHandler(t, NewArtifactsHandler(state))
	for _, want := range []string{
		"CLI-generated artifacts",
		"Artificial photosynthesis catalyst review",
		"Records: 1",
		"Included: 1",
		"Run: run1",
	} {
		if !strings.Contains(artBody, want) {
			t.Fatalf("artifacts page missing %q:\n%s", want, artBody)
		}
	}
}

func renderHandler(t *testing.T, h http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("handler status = %d", rec.Code)
	}
	return rec.Body.String()
}
