package webui

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

//go:embed static
var staticFS embed.FS

// Config configures the local research cockpit server.
type Config struct {
	// ProjectPath is the initial research project folder the dashboard reads
	// CLI-generated artifacts (library, screening, analysis, citation graph,
	// PDFs) from. It can be changed at runtime via the in-browser project
	// switcher. An empty path serves the shell and project create/open forms.
	ProjectPath string
}

// dashboardState holds the active research folder, which the in-browser project
// switcher can change while the server runs. A single dashboard process serves
// one active folder at a time; run multiple processes on different --addr ports
// to view several projects side by side.
type dashboardState struct {
	mu      sync.RWMutex
	project string
}

func (s *dashboardState) get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.project
}

func (s *dashboardState) set(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.project = path
}

// defaultSearchSources lists the connector sources offered on the search form.
func defaultSearchSources() []string {
	return []string{"openalex", "arxiv", "crossref"}
}

// Routes returns the sorted list of top-level routes the dashboard serves.
// It is used by `rforge ui --json` and tests to describe the dashboard surface
// without binding a port.
func Routes() []string {
	routes := []string{
		"/",
		"/acquisition",
		"/analysis",
		"/architecture",
		"/artifacts",
		"/connectors",
		"/dedupe",
		"/evidence",
		"/forge",
		"/library",
		"/map",
		"/notebook",
		"/oss",
		"/package",
		"/papers",
		"/parsing",
		"/privacy",
		"/projects",
		"/report",
		"/retrieve",
		"/screening",
		"/search",
		"/sources",
		"/workbenches",
	}
	sort.Strings(routes)
	return routes
}

// NewRouter builds the local research cockpit HTTP handler. View models are
// rebuilt per request from the active project folder so the dashboard always
// reflects current CLI-generated state. The CLI remains the authoritative
// automation path; this router only visualizes project state.
func NewRouter(cfg Config) http.Handler {
	state := &dashboardState{project: cfg.ProjectPath}
	mux := http.NewServeMux()

	mux.Handle("/", newRootHandler())
	mux.Handle("/projects", NewProjectHandler())
	mux.Handle("/projects/create", NewCreateProjectHandler())
	mux.Handle("/projects/open", NewOpenProjectHandler())
	mux.Handle("GET /projects/active", newActiveProjectHandler(state))
	mux.Handle("POST /projects/switch", newSwitchProjectHandler(state))
	mux.Handle("/search", NewSearchHandler(ui.NewSearchFormState(defaultSearchSources())))
	mux.Handle("/sources", NewSourcePlanningHandler())
	mux.Handle("/architecture", NewInformationArchitectureHandler(BuildDashboardInformationArchitecture()))
	mux.Handle("/privacy", NewPrivacyModelHandler(BuildDashboardPrivacyModel()))
	mux.Handle("/workbenches", NewWorkbenchIndexHandler(BuildWorkbenchIndexState()))
	mux.Handle("/notebook", newLabNotebookHandler(state.get))
	mux.Handle("/notebook/snapshot.json", newLabNotebookSnapshotHandler(state.get))
	mux.Handle("/parsing", NewParserConflictReviewHandler(BuildParserConflictReviewState()))
	mux.Handle("/map", newResearchMapHandler(state.get))
	mux.Handle("/map/snapshot.json", newResearchMapSnapshotHandler(state.get))
	mux.Handle("/acquisition", newAcquisitionQueueHandler(state.get))
	mux.Handle("/retrieve", newRetrievalTuningHandler(state.get))
	mux.Handle("/evidence", newEvidenceGridHandler(state.get))
	mux.Handle("/analysis", newAnalysisWorkbenchHandler(state.get))
	for _, route := range []string{"/report", "/package"} {
		mux.Handle(route, newGenericWorkbenchHandler(route))
	}
	mux.Handle("/connectors", newConnectorHealthHandler(state.get))
	mux.Handle("/dedupe", newDedupeReviewHandler(state.get))
	forgeHandler := newForgeHomeHandler(state.get)
	mux.Handle("/forge", forgeHandler)
	mux.Handle("/forge/refresh", forgeHandler)
	screeningHandler := newScreeningCockpitHandler(state.get)
	mux.Handle("/screening", screeningHandler)
	mux.Handle("/screening/refresh", screeningHandler)

	libraryHandler := newProjectLibraryHandler(state.get)
	mux.Handle("/library", libraryHandler)
	mux.Handle("/library/rows", libraryHandler)

	artifactsHandler := newProjectArtifactsHandler(state.get)
	mux.Handle("/artifacts", artifactsHandler)
	mux.Handle("/artifacts/refresh", artifactsHandler)
	mux.Handle("GET /artifacts/graph.json", newCitationGraphJSONHandler(state.get))

	mux.Handle("GET /papers", newPaperListHandler(state.get))
	mux.Handle("GET /papers/{id}", newPaperDetailHandler(state.get))
	mux.Handle("GET /papers/{id}/pdf", newPaperPDFHandler(state.get))

	if sub, err := fs.Sub(staticFS, "static"); err == nil {
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(sub))))
	}

	return mux
}

// newRootHandler serves the shell only for the exact "/" path and 404s any
// other unmatched path so unknown routes are not silently rendered as the shell.
func newRootHandler() http.Handler {
	shell := NewShellHandler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		shell.ServeHTTP(w, r)
	})
}

func newProjectLibraryHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vm, err := BuildLibraryViewModel(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NewLibraryHandler(vm).ServeHTTP(w, r)
	})
}

func newProjectArtifactsHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildArtifactDashboardState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NewArtifactsHandler(state).ServeHTTP(w, r)
	})
}

var activeProjectTemplate = template.Must(template.New("active-project").Parse(`<section aria-labelledby="active-project-title" class="rf-card">
  <h2 id="active-project-title">Active research folder</h2>
  <p id="active-project-path">{{if .}}{{.}}{{else}}No folder selected{{end}}</p>
  <form hx-post="/projects/switch" hx-target="#active-project-path" hx-swap="outerHTML">
    <label for="switch-path">Switch to folder</label>
    <input id="switch-path" name="path" required>
    <button type="submit">Open folder</button>
  </form>
</section>`))

func newActiveProjectHandler(state *dashboardState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = activeProjectTemplate.Execute(w, state.get())
	})
}

// newSwitchProjectHandler points the dashboard at a different research folder.
// It accepts any existing directory (a research folder need not be a fully
// initialized project), rejects missing paths, and triggers the HTMX clients to
// refresh the library, artifacts, and papers views.
func newSwitchProjectHandler(state *dashboardState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path := strings.TrimSpace(r.FormValue("path"))
		if path == "" {
			http.Error(w, "research folder path is required", http.StatusBadRequest)
			return
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			http.Error(w, "research folder does not exist", http.StatusBadRequest)
			return
		}
		state.set(path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "refresh-library, refresh-artifacts")
		_, _ = w.Write([]byte(`<p id="active-project-path">` + template.HTMLEscapeString(path) + `</p>`))
	})
}
