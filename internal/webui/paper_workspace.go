package webui

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

var recordIDPattern = regexp.MustCompile(`^rec_[0-9a-f]{32}$`)

// PaperWorkspaceView joins one library record with the local research context
// and readable assets available for that record.
type PaperWorkspaceView struct {
	RecordID    string
	Title       string
	Authors     string
	Year        int
	Venue       string
	Identifiers []string
	Collections string
	Tags        string
	Notes       string
	Abstract    string
	Sections    []parsing.Section
	HasPDF      bool
	Parsed      bool
	assetStem   string
}

// BuildPaperWorkspace resolves one stable library record into its read-only
// workspace. Metadata-only records are valid workspaces.
func BuildPaperWorkspace(projectPath, recordID string) (PaperWorkspaceView, bool, error) {
	if strings.TrimSpace(projectPath) == "" || !recordIDPattern.MatchString(recordID) {
		return PaperWorkspaceView{}, false, nil
	}
	store, err := library.OpenStore(filepath.Join(projectPath, "data", "library.json"))
	if err != nil {
		return PaperWorkspaceView{}, false, err
	}
	record, found, err := store.GetByRecordID(recordID)
	if err != nil || !found {
		return PaperWorkspaceView{}, found, err
	}
	view := PaperWorkspaceView{
		RecordID:    record.RecordID,
		Title:       record.Title,
		Authors:     libraryAuthorsLine(record.Authors),
		Year:        record.Year,
		Venue:       record.Venue,
		Identifiers: workspaceIdentifiers(record.Identifiers),
		Collections: libraryMetadataValue(record, "collections", "groups"),
		Tags:        libraryMetadataValue(record, "tags", "keywords"),
		Notes:       libraryMetadataValue(record, "note"),
	}
	for _, stem := range workspaceAssetStems(record) {
		parsed, ok, loadErr := BuildPaperView(projectPath, stem)
		if loadErr != nil {
			return PaperWorkspaceView{}, false, loadErr
		}
		if ok {
			view.Parsed = true
			view.Abstract = parsed.Abstract
			view.Sections = parsed.Sections
			view.HasPDF = parsed.HasPDF
			view.assetStem = stem
			break
		}
		if paperPDFAvailable(projectPath, stem) {
			view.HasPDF = true
			view.assetStem = stem
			break
		}
	}
	return view, true, nil
}

func recordIDForAssetStem(projectPath, stem string) (string, bool, error) {
	path := filepath.Join(projectPath, "data", "library.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, nil
	}
	store, err := library.OpenStore(path)
	if err != nil {
		return "", false, err
	}
	records, err := store.List()
	if err != nil {
		return "", false, err
	}
	for _, record := range records {
		for _, candidate := range workspaceAssetStems(record) {
			if candidate == stem {
				return record.RecordID, true, nil
			}
		}
	}
	return "", false, nil
}

func workspaceAssetStatus(projectPath string, record library.PaperRecord) (bool, bool, error) {
	for _, stem := range workspaceAssetStems(record) {
		parsed, ok, err := BuildPaperView(projectPath, stem)
		if err != nil {
			return false, false, err
		}
		if ok {
			return parsed.HasPDF, true, nil
		}
		if paperPDFAvailable(projectPath, stem) {
			return true, false, nil
		}
	}
	return false, false, nil
}

func workspaceAssetStems(record library.PaperRecord) []string {
	ids := record.Identifiers
	values := []string{ids.DOI, ids.ArXivID, ids.PMID, ids.PMCID, ids.OpenAlexID, ids.SemanticScholarID, ids.ZoteroItemKey, ids.ADSBibcode}
	seen := map[string]bool{}
	stems := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		stem := graphNodeStem(value)
		if !seen[stem] {
			seen[stem] = true
			stems = append(stems, stem)
		}
	}
	return stems
}

func workspaceIdentifiers(ids library.Identifiers) []string {
	values := []struct {
		label string
		value string
	}{
		{"DOI", ids.DOI},
		{"arXiv", ids.ArXivID},
		{"PMID", ids.PMID},
		{"PMCID", ids.PMCID},
		{"OpenAlex", ids.OpenAlexID},
		{"Semantic Scholar", ids.SemanticScholarID},
		{"Zotero", ids.ZoteroItemKey},
		{"ADS", ids.ADSBibcode},
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		if value := strings.TrimSpace(item.value); value != "" {
			out = append(out, fmt.Sprintf("%s: %s", item.label, value))
		}
	}
	return out
}

var paperWorkspaceTemplate = template.Must(template.New("paper-workspace").Parse(`<section aria-labelledby="paper-workspace-title" class="rf-card">
  <p><a href="/library">&larr; Library</a></p>
  <h2 id="paper-workspace-title">{{.Title}}</h2>
  {{if .Authors}}<p class="paper-authors">{{.Authors}}</p>{{end}}
  <dl>
    {{if .Year}}<dt>Year</dt><dd>{{.Year}}</dd>{{end}}
    {{if .Venue}}<dt>Venue</dt><dd>{{.Venue}}</dd>{{end}}
    {{if .Identifiers}}<dt>Identifiers</dt><dd><ul>{{range .Identifiers}}<li>{{.}}</li>{{end}}</ul></dd>{{end}}
    {{if .Collections}}<dt>Collections</dt><dd>{{.Collections}}</dd>{{end}}
    {{if .Tags}}<dt>Tags</dt><dd>{{.Tags}}</dd>{{end}}
    {{if .Notes}}<dt>Notes</dt><dd>{{.Notes}}</dd>{{end}}
    <dt>Assets</dt><dd>{{if .HasPDF}}PDF {{end}}{{if .Parsed}}Parsed text{{else}}Metadata only{{end}}</dd>
  </dl>
  {{if .HasPDF}}<div class="rf-paper-pdf"><embed src="/library/{{.RecordID}}/pdf" type="application/pdf" aria-label="Paper PDF"></div>{{end}}
  {{if .Parsed}}<div class="rf-paper-text">
    {{if .Abstract}}<section aria-label="Abstract"><h3>Abstract</h3><p>{{.Abstract}}</p></section>{{end}}
    {{range .Sections}}<section aria-label="{{.Title}}"><h3>{{if .Title}}{{.Title}}{{else}}Section{{end}}</h3>{{range .Passages}}<p data-passage-id="{{.ID}}">{{.Text}}</p>{{end}}</section>{{end}}
  </div>{{end}}
</section>`))

func newPaperWorkspacePDFHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		view, found, err := BuildPaperWorkspace(projectPath(), r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found || !view.HasPDF || view.assetStem == "" {
			http.NotFound(w, r)
			return
		}
		file, ok := openPaperPDF(projectPath(), view.assetStem)
		if !ok {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			http.Error(w, "failed to read PDF", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	})
}

func newPaperWorkspaceHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		view, found, err := BuildPaperWorkspace(projectPath(), r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = paperWorkspaceTemplate.Execute(w, view)
	})
}
