package webui

import (
	"html/template"
	"net/http"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

var shellTemplate = template.Must(template.New("shell").Parse(`<!doctype html>
<html lang="en" hx-boost="true">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ResearchForge</title>
  <script src="https://unpkg.com/htmx.org@1.9.12" integrity="sha384-2VbB9nQbS2QZ4WJjvQ8WbQbQwQvQwQvQwQvQwQvQwQvQwQvQwQvQwQvQwQvQw" crossorigin="anonymous"></script>
  <link rel="stylesheet" href="/assets/researchforge.css">
</head>
<body>
  <main class="rf-shell">
    <header>
      <p class="eyebrow">Local Go + HTMX workspace</p>
      <h1>ResearchForge</h1>
      <p>Project dashboard and CLI-generated artifacts will be served from shared Go application services.</p>
    </header>
    <section aria-labelledby="dashboard-title">
      <h2 id="dashboard-title">Project dashboard</h2>
      <p>Open or create a project after the implementation tracker slices land.</p>
    </section>
    <section aria-labelledby="artifacts-title">
      <h2 id="artifacts-title">CLI-generated artifacts</h2>
      <p>Papers, PRISMA diagrams, citation graphs, analyses, and reports remain reproducible from CLI output.</p>
    </section>
  </main>
</body>
</html>`))

var searchTemplate = template.Must(template.New("search").Parse(`<section aria-labelledby="search-title" class="rf-card">
  <h2 id="search-title">Search papers</h2>
  <form hx-get="/search/results" hx-target="#search-results" hx-indicator="#search-loading">
    <label for="query">Query</label>
    <input id="query" name="query" type="search" value="{{.Query}}">
    <fieldset>
      <legend>Sources</legend>
      {{range .Sources}}
      <label><input type="checkbox" name="source" value="{{.}}" checked> {{.}}</label>
      {{end}}
    </fieldset>
    <button type="submit">Search</button>
  </form>
  <p id="search-loading" class="htmx-indicator">Loading</p>
  <div id="search-results" role="status">No results yet</div>
</section>`))

var libraryTemplate = template.Must(template.New("library").Parse(`<section aria-labelledby="library-title" class="rf-card" hx-get="/library/rows" hx-trigger="refresh-library from:body">
  <h2 id="library-title">Library</h2>
  {{if .Empty}}
  <div role="status" class="empty-state">
    <p>No papers yet</p>
    <p>Import or search for papers to populate the library.</p>
  </div>
  {{else}}
  <div role="table" aria-label="Paper library">
    <div role="row">
      <strong role="columnheader">Paper title</strong>
    </div>
    {{range .Rows}}
    <div role="row">
      <span role="cell">{{.Title}}</span>
    </div>
    {{end}}
  </div>
  {{end}}
</section>`))

var projectTemplate = template.Must(template.New("project").Parse(`<section aria-labelledby="projects-title" class="rf-card">
  <h2 id="projects-title">Projects</h2>
  <form hx-post="/projects/create" hx-target="#project-status">
    <h3>Create project</h3>
    <label for="project-title">Title</label>
    <input id="project-title" name="title" required>
    <label for="project-create-path">Path</label>
    <input id="project-create-path" name="path" required>
    <button type="submit">Create project</button>
  </form>
  <form hx-post="/projects/open" hx-target="#project-status">
    <h3>Open project</h3>
    <label for="project-open-path">Path</label>
    <input id="project-open-path" name="path" required>
    <button type="submit">Open project</button>
  </form>
  <div id="project-status" role="status">No project open</div>
</section>`))

var projectStatusTemplate = template.Must(template.New("project-status").Parse(`<article class="rf-status">
  <h3>{{.Action}} project</h3>
  <dl>
    <dt>Title</dt><dd>{{.Project.Title}}</dd>
    <dt>Path</dt><dd>{{.Project.Path}}</dd>
    <dt>Storage</dt><dd>{{.Project.StorageMode}}</dd>
    <dt>Manifest</dt><dd>{{.Project.ManifestPath}}</dd>
  </dl>
</article>`))

var ossTemplate = template.Must(template.New("oss").Parse(`<section aria-labelledby="oss-title" class="rf-card" hx-get="/oss/repositories" hx-trigger="refresh-oss from:body">
  <h2 id="oss-title">OSS repository studies</h2>
  {{if .Repositories}}
  <div role="table" aria-label="OSS repository studies">
    <div role="row">
      <strong role="columnheader">Repository</strong>
    </div>
    {{range .Repositories}}
    <div role="row">
      <span role="cell">{{.Name}}</span>
    </div>
    {{end}}
  </div>
  {{else}}
  <div role="status" class="empty-state">
    <p>No repository studies yet</p>
    <p>Run an OSS repository study to compare repository evidence.</p>
  </div>
  {{end}}
</section>`))

// PRISMAFlowState summarizes CLI-generated screening counts for the web artifacts view.
type PRISMAFlowState struct {
	Records  int
	Screened int
	Included int
}

// ArtifactDashboardState combines CLI-generated outputs for the local web GUI.
type ArtifactDashboardState struct {
	Papers        ui.LibraryViewModel
	Analysis      ui.AnalysisViewModel
	CitationGraph ui.CitationGraphViewModel
	PRISMA        PRISMAFlowState
	Reports       ui.ReportViewModel
}

var artifactsTemplate = template.Must(template.New("artifacts").Parse(`<section aria-labelledby="artifacts-title" class="rf-card" hx-get="/artifacts/refresh" hx-trigger="refresh-artifacts from:body">
  <h2 id="artifacts-title">CLI-generated artifacts</h2>
  <section aria-labelledby="artifact-papers-title">
    <h3 id="artifact-papers-title">Papers</h3>
    {{if .Papers.Rows}}{{range .Papers.Rows}}<p>{{.Title}}</p>{{end}}{{else}}<p>No papers exported yet</p>{{end}}
  </section>
  <section aria-labelledby="artifact-analysis-title">
    <h3 id="artifact-analysis-title">Meta-analysis outputs</h3>
    {{if .Analysis.Ready}}<p>Ready: {{.Analysis.RunID}}</p>{{else}}<p>No analysis run ready</p>{{end}}
  </section>
  <section aria-labelledby="artifact-prisma-title">
    <h3 id="artifact-prisma-title">PRISMA diagram</h3>
    <p>Records: {{.PRISMA.Records}}</p>
    <p>Screened: {{.PRISMA.Screened}}</p>
    <p>Included: {{.PRISMA.Included}}</p>
  </section>
  <section aria-labelledby="artifact-citations-title">
    <h3 id="artifact-citations-title">Citation graph</h3>
    {{if .CitationGraph.Nodes}}
    {{range .CitationGraph.Nodes}}<p>{{.ID}}</p>{{end}}
    {{range .CitationGraph.Edges}}<p>{{.Source}} → {{.Target}}</p>{{end}}
    {{else}}<p>No citation graph exported yet</p>{{end}}
  </section>
  <section aria-labelledby="artifact-reports-title">
    <h3 id="artifact-reports-title">Report artifacts</h3>
    {{if .Reports.Formats}}{{range .Reports.Formats}}<p>{{.}}</p>{{end}}{{else}}<p>No report formats exported yet</p>{{end}}
  </section>
</section>`))

// NewShellHandler returns the dependency-light local web GUI shell handler.
func NewShellHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = shellTemplate.Execute(w, nil)
	})
}

// NewSearchHandler renders the HTMX search screen from the shared search view model.
func NewSearchHandler(state ui.SearchFormState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = searchTemplate.Execute(w, state)
	})
}

// NewLibraryHandler renders the HTMX library screen from the shared library view model.
func NewLibraryHandler(state ui.LibraryViewModel) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = libraryTemplate.Execute(w, state)
	})
}

// NewProjectHandler renders the HTMX create/open project screen.
func NewProjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = projectTemplate.Execute(w, nil)
	})
}

// NewOSSHandler renders OSS repository studies from the shared OSS dashboard view model.
func NewOSSHandler(state ui.OSSDashboardViewModel) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = ossTemplate.Execute(w, state)
	})
}

// NewArtifactsHandler renders CLI-generated papers, analyses, diagrams, and reports.
func NewArtifactsHandler(state ArtifactDashboardState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = artifactsTemplate.Execute(w, state)
	})
}

// NewCreateProjectHandler creates a local project workspace from an HTMX form post.
func NewCreateProjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		proj, err := project.Create(r.FormValue("path"), project.CreateOptions{Title: r.FormValue("title")})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		renderProjectStatus(w, "Created", proj)
	})
}

// NewOpenProjectHandler opens an existing local project workspace from an HTMX form post.
func NewOpenProjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		proj, err := project.Inspect(r.FormValue("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		renderProjectStatus(w, "Opened", proj)
	})
}

func renderProjectStatus(w http.ResponseWriter, action string, proj project.Project) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Trigger", "project-opened")
	_ = projectStatusTemplate.Execute(w, struct {
		Action  string
		Project project.Project
	}{Action: action, Project: proj})
}
