package webui

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/protocol"
	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

var shellTemplate = template.Must(template.New("shell").Parse(`<!doctype html>
<html lang="en" hx-boost="true">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ResearchForge</title>
  <script src="/assets/htmx.min.js" defer></script>
  <link rel="stylesheet" href="/assets/researchforge.css">
</head>
<body>
  <main class="rf-shell">
    <header>
      <p class="eyebrow">Local Go + HTMX workspace</p>
      <h1>ResearchForge</h1>
      <p>Read papers, browse CLI-generated artifacts, and explore the knowledge graph for the open research project.</p>
      <nav aria-label="Dashboard sections">
        <ul class="rf-nav">
          <li><a href="/papers">Papers</a></li>
          <li><a href="/library">Library</a></li>
          <li><a href="/artifacts">Artifacts</a></li>
          <li><a href="/oss">OSS studies</a></li>
          <li><a href="/search">Search</a></li>
          <li><a href="/sources">Source plan</a></li>
          <li><a href="/projects">Projects</a></li>
        </ul>
      </nav>
    </header>
    <div hx-get="/projects/active" hx-trigger="load"></div>
    <section aria-labelledby="dashboard-title">
      <h2 id="dashboard-title">Project dashboard</h2>
      <p>Open a research folder, then read every parsed paper in the browser and review its analysis, PRISMA flow, and citation graph.</p>
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

var sourcePlanningTemplate = template.Must(template.New("source-planning").Parse(`<section aria-labelledby="source-planning-title" class="rf-card">
  <h2 id="source-planning-title">Source planning cockpit</h2>
  <form method="get" action="/sources">
    <label for="source-question">Research question</label>
    <input id="source-question" name="question" value="{{.Question}}" required>
    <label for="source-type">Framework</label>
    <select id="source-type" name="type">
      <option value="freeform">freeform</option>
      <option value="pico">pico</option>
      <option value="peco">peco</option>
      <option value="spider">spider</option>
    </select>
    <button type="submit">Preview sources</button>
  </form>
  <p>CLI equivalent: <code>rforge protocol plan-sources --question '{{.Question}}'</code></p>
  <p><strong>Reviewer approval required</strong> before network calls, imports, downloads, or package inclusion.</p>
  <div role="table" aria-label="Source plan preview">
    <div role="row"><strong role="columnheader">Source</strong> <strong role="columnheader">Kind</strong> <strong role="columnheader">Dry run</strong> <strong role="columnheader">Privacy/Auth</strong></div>
    {{range .Sources}}
    <div role="row">
      <span role="cell">{{.Label}}</span>
      <span role="cell">{{.SourceKind}}</span>
      <span role="cell">{{.DryRunEstimate}}</span>
      <span role="cell">{{.PrivacyWarning}} Auth: {{.AuthRequirement}}</span>
      <code role="cell">{{.CLICommand}}</code>
    </div>
    {{end}}
  </div>
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

// AnalysisDetail surfaces a stored meta-analysis result for the cockpit:
// heterogeneity metrics, plot availability, and runner warnings.
type AnalysisDetail struct {
	Ready         bool
	RunID         string
	I2            float64
	Tau2          float64
	Q             float64
	HasForestPlot bool
	HasFunnelPlot bool
	Warnings      []string
}

// ArtifactDashboardState combines CLI-generated outputs for the local web GUI.
type ArtifactDashboardState struct {
	Papers         ui.LibraryViewModel
	Analysis       ui.AnalysisViewModel
	AnalysisDetail AnalysisDetail
	CitationGraph  ui.CitationGraphViewModel
	PRISMA         PRISMAFlowState
	Reports        ui.ReportViewModel
}

// CitationGraphSVG renders a small accessible SVG preview for exported citation graphs.
func (s ArtifactDashboardState) CitationGraphSVG() template.HTML {
	if len(s.CitationGraph.Nodes) == 0 {
		return ""
	}
	positions := map[string][2]int{}
	var b strings.Builder
	b.WriteString(`<svg role="img" aria-label="Citation graph visualization" viewBox="0 0 520 160" class="citation-graph-svg">`)
	b.WriteString(`<title>Citation graph visualization</title>`)
	for i, node := range s.CitationGraph.Nodes {
		x := 80 + (i%5)*100
		y := 50 + (i/5)*70
		positions[node.ID] = [2]int{x, y}
	}
	for _, edge := range s.CitationGraph.Edges {
		source, sourceOK := positions[edge.Source]
		target, targetOK := positions[edge.Target]
		if !sourceOK || !targetOK {
			continue
		}
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="currentColor" stroke-width="1" />`, source[0], source[1], target[0], target[1])
	}
	for _, node := range s.CitationGraph.Nodes {
		pos := positions[node.ID]
		label := template.HTMLEscapeString(node.ID)
		stem := template.HTMLEscapeString(graphNodeStem(node.ID))
		fmt.Fprintf(&b, `<a href="/papers/%s">`, stem)
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="16" fill="none" stroke="currentColor" stroke-width="2" />`, pos[0], pos[1])
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle">%s</text>`, pos[0], pos[1]+34, label)
		b.WriteString(`</a>`)
	}
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

var artifactsTemplate = template.Must(template.New("artifacts").Parse(`<section aria-labelledby="artifacts-title" class="rf-card" hx-get="/artifacts/refresh" hx-trigger="refresh-artifacts from:body">
  <h2 id="artifacts-title">CLI-generated artifacts</h2>
  <section aria-labelledby="artifact-papers-title">
    <h3 id="artifact-papers-title">Papers</h3>
    {{if .Papers.Rows}}{{range .Papers.Rows}}<p>{{.Title}}</p>{{end}}{{else}}<p>No papers exported yet</p>{{end}}
  </section>
  <section aria-labelledby="artifact-analysis-title">
    <h3 id="artifact-analysis-title">Meta-analysis outputs</h3>
    {{if .AnalysisDetail.Ready}}
    <p>Run: {{.AnalysisDetail.RunID}}</p>
    <dl>
      <dt>Heterogeneity I²</dt><dd>{{.AnalysisDetail.I2}}</dd>
      <dt>τ²</dt><dd>{{.AnalysisDetail.Tau2}}</dd>
      <dt>Q</dt><dd>{{.AnalysisDetail.Q}}</dd>
    </dl>
    <ul>
      {{if .AnalysisDetail.HasForestPlot}}<li>Forest plot registered</li>{{end}}
      {{if .AnalysisDetail.HasFunnelPlot}}<li>Funnel plot registered</li>{{end}}
    </ul>
    {{if .AnalysisDetail.Warnings}}<p class="analysis-warnings">Warnings: {{range .AnalysisDetail.Warnings}}{{.}}; {{end}}</p>{{end}}
    {{else}}<p>No analysis run ready</p>{{end}}
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
    <div id="citation-graph" data-citation-graph data-src="/artifacts/graph.json" role="application" aria-label="Citation graph (drag to pan, scroll to zoom, click a node to open the paper)">
      {{.CitationGraphSVG}}
    </div>
    <script src="/assets/citation-graph.js" defer></script>
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

// NewSourcePlanningHandler renders the source-planning cockpit over protocol compiler output.
func NewSourcePlanningHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		question := strings.TrimSpace(r.URL.Query().Get("question"))
		if question == "" {
			question = "Does the intervention improve the outcome compared with control?"
		}
		plan, err := protocol.CompileSourcePlanFromQuestion(protocol.QuestionInput{
			Framework:    r.URL.Query().Get("type"),
			Question:     question,
			Population:   r.URL.Query().Get("population"),
			Intervention: r.URL.Query().Get("intervention"),
			Comparator:   r.URL.Query().Get("comparator"),
			Outcome:      r.URL.Query().Get("outcome"),
			Exposure:     r.URL.Query().Get("exposure"),
			Sample:       r.URL.Query().Get("sample"),
			Phenomenon:   r.URL.Query().Get("phenomenon"),
			Design:       r.URL.Query().Get("design"),
			Evaluation:   r.URL.Query().Get("evaluation"),
			ResearchType: r.URL.Query().Get("research-type"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = sourcePlanningTemplate.Execute(w, plan)
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
