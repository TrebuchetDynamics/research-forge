package webui

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// paperIDPattern matches the safe stem produced by the CLI parse/fetch commands
// (lowercase alphanumerics joined by hyphens). It is the allow-list used to
// reject path traversal in /papers/{id} routes.
var paperIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// paperPDFDirs are the project-local directories the CLI fetches PDFs into.
var paperPDFDirs = []string{
	filepath.Join("documents", "open-access"),
	filepath.Join("documents", "arxiv"),
	filepath.Join("documents", "local"),
}

// PaperSummary is one readable paper in the cockpit paper list.
type PaperSummary struct {
	ID           string
	Title        string
	Authors      string
	PassageCount int
	HasPDF       bool
}

// PaperListViewModel lists parsed papers available to read in the dashboard.
type PaperListViewModel struct {
	Papers []PaperSummary
	Empty  bool
}

// PaperView is the per-paper reading view: parsed structure plus an optional
// project-local PDF rendered natively by the browser.
type PaperView struct {
	ID       string
	Title    string
	Authors  string
	Abstract string
	Sections []parsing.Section
	HasPDF   bool
	Warnings []string
}

// sanitizePaperID validates a routed paper id against the safe-stem allow-list,
// rejecting any value that could escape the project parsed/documents folders.
func sanitizePaperID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if !paperIDPattern.MatchString(id) {
		return "", false
	}
	return id, true
}

// graphNodeStem normalizes a citation-graph node id (a paper identifier such as
// a DOI) into the same safe stem the CLI uses to name parsed documents, so a
// graph node can link to its /papers/{id} reading page.
func graphNodeStem(id string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(id)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return "item"
	}
	return strings.Join(parts, "-")
}

func authorsLine(doc parsing.ParsedDocument) string {
	names := make([]string, 0, len(doc.Authors))
	for _, a := range doc.Authors {
		name := strings.TrimSpace(strings.TrimSpace(a.Given) + " " + strings.TrimSpace(a.Family))
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func loadParsedDoc(projectPath, id string) (parsing.ParsedDocument, bool, error) {
	var doc parsing.ParsedDocument
	data, err := os.ReadFile(filepath.Join(projectPath, "parsed", id+".json"))
	if os.IsNotExist(err) {
		return doc, false, nil
	}
	if err != nil {
		return doc, false, err
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, false, err
	}
	return doc, true, nil
}

// paperPDFPath returns the project-local PDF for a paper id, or "" when none is
// present. It matches a fetched PDF whose filename stem equals the paper id,
// which the CLI guarantees because parse and fetch share the same safe-stem
// normalization.
func paperPDFPath(projectPath, id string) string {
	for _, sub := range paperPDFDirs {
		candidate := filepath.Join(projectPath, sub, id+".pdf")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// BuildPaperList reads the project's parsed/ directory into a readable paper
// list. A project without parsed documents yields an empty, non-error model.
func BuildPaperList(projectPath string) (PaperListViewModel, error) {
	if strings.TrimSpace(projectPath) == "" {
		return PaperListViewModel{Empty: true}, nil
	}
	entries, err := os.ReadDir(filepath.Join(projectPath, "parsed"))
	if os.IsNotExist(err) {
		return PaperListViewModel{Empty: true}, nil
	}
	if err != nil {
		return PaperListViewModel{}, err
	}
	papers := make([]PaperSummary, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".manifest.json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		doc, ok, err := loadParsedDoc(projectPath, id)
		if err != nil {
			return PaperListViewModel{}, err
		}
		if !ok {
			continue
		}
		passages := 0
		for _, section := range doc.Sections {
			passages += len(section.Passages)
		}
		title := strings.TrimSpace(doc.Title)
		if title == "" {
			title = id
		}
		papers = append(papers, PaperSummary{
			ID:           id,
			Title:        title,
			Authors:      authorsLine(doc),
			PassageCount: passages,
			HasPDF:       paperPDFPath(projectPath, id) != "",
		})
	}
	sort.Slice(papers, func(i, j int) bool { return papers[i].Title < papers[j].Title })
	return PaperListViewModel{Papers: papers, Empty: len(papers) == 0}, nil
}

// BuildPaperView loads a single parsed paper for the reading view. ok is false
// when no parsed document exists for the id.
func BuildPaperView(projectPath, id string) (PaperView, bool, error) {
	doc, ok, err := loadParsedDoc(projectPath, id)
	if err != nil || !ok {
		return PaperView{}, ok, err
	}
	title := strings.TrimSpace(doc.Title)
	if title == "" {
		title = id
	}
	return PaperView{
		ID:       id,
		Title:    title,
		Authors:  authorsLine(doc),
		Abstract: doc.Abstract,
		Sections: doc.Sections,
		HasPDF:   paperPDFPath(projectPath, id) != "",
		Warnings: doc.Warnings,
	}, true, nil
}

var paperListTemplate = template.Must(template.New("papers").Parse(`<section aria-labelledby="papers-title" class="rf-card" hx-get="/papers" hx-trigger="refresh-papers from:body">
  <h2 id="papers-title">Papers</h2>
  {{if .Empty}}
  <div role="status" class="empty-state">
    <p>No parsed papers yet</p>
    <p>Run <code>rforge parse</code> to make papers readable in the browser.</p>
  </div>
  {{else}}
  <div role="table" aria-label="Readable papers">
    <div role="row"><strong role="columnheader">Title</strong></div>
    {{range .Papers}}
    <div role="row">
      <span role="cell"><a href="/papers/{{.ID}}">{{.Title}}</a></span>
      <span role="cell">{{.Authors}}</span>
      <span role="cell">{{.PassageCount}} passages</span>
      {{if .HasPDF}}<span role="cell">PDF</span>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
</section>`))

var paperViewTemplate = template.Must(template.New("paper").Parse(`<section aria-labelledby="paper-title" class="rf-card">
  <p><a href="/papers">&larr; All papers</a></p>
  <h2 id="paper-title">{{.Title}}</h2>
  {{if .Authors}}<p class="paper-authors">{{.Authors}}</p>{{end}}
  <div class="rf-paper-view">
    <div class="rf-paper-pdf">
      {{if .HasPDF}}
      <embed src="/papers/{{.ID}}/pdf" type="application/pdf" aria-label="Paper PDF">
      {{else}}
      <p role="status">No local PDF available. Showing parsed full text only.</p>
      {{end}}
    </div>
    <div class="rf-paper-text">
      {{if .Abstract}}<section aria-label="Abstract"><h3>Abstract</h3><p>{{.Abstract}}</p></section>{{end}}
      {{range .Sections}}
      <section aria-label="{{.Title}}">
        <h3>{{if .Title}}{{.Title}}{{else}}Section{{end}}</h3>
        {{range .Passages}}<p data-passage-id="{{.ID}}">{{.Text}}</p>{{end}}
      </section>
      {{end}}
    </div>
  </div>
</section>`))

func newPaperListHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vm, err := BuildPaperList(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = paperListTemplate.Execute(w, vm)
	})
}

func newPaperDetailHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := sanitizePaperID(r.PathValue("id"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		view, found, err := BuildPaperView(projectPath(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = paperViewTemplate.Execute(w, view)
	})
}

func newPaperPDFHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := sanitizePaperID(r.PathValue("id"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		path := paperPDFPath(projectPath(), id)
		if path == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, path)
	})
}
