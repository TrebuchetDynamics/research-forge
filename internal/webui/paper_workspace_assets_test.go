package webui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestPaperWorkspaceAssetsResolve(t *testing.T) {
	const doi, arxiv, pmid = "10-1000-ap", "1706-03762", "12345"
	for _, tc := range []struct {
		name       string
		parsed     []string
		pdfs       []string
		symlinkPDF string
		malformed  string
		wantParsed string
		wantPDF    string
		wantError  bool
	}{
		{name: "DOI text and arXiv PDF", parsed: []string{doi}, pdfs: []string{arxiv}, wantParsed: doi, wantPDF: arxiv},
		{name: "DOI PDF and arXiv text", parsed: []string{arxiv}, pdfs: []string{doi}, wantParsed: arxiv, wantPDF: doi},
		{name: "first candidates win", parsed: []string{arxiv, doi}, pdfs: []string{arxiv, doi}, wantParsed: doi, wantPDF: doi},
		{name: "competing PDFs after text", parsed: []string{doi}, pdfs: []string{pmid, arxiv}, wantParsed: doi, wantPDF: arxiv},
		{name: "competing text after PDF", parsed: []string{pmid, arxiv}, pdfs: []string{doi}, wantParsed: arxiv, wantPDF: doi},
		{name: "missing assets ignore unrelated stems", parsed: []string{"other-record"}, pdfs: []string{"other-record"}},
		{name: "parsed only", parsed: []string{arxiv}, wantParsed: arxiv},
		{name: "PDF only", pdfs: []string{arxiv}, wantPDF: arxiv},
		{name: "external symlink rejected", parsed: []string{doi}, symlinkPDF: arxiv, wantParsed: doi},
		{name: "external symlink skipped for safe PDF", parsed: []string{doi}, symlinkPDF: arxiv, pdfs: []string{pmid}, wantParsed: doi, wantPDF: pmid},
		{name: "malformed text after PDF is inspected", pdfs: []string{doi}, malformed: arxiv, wantError: true},
		{name: "later text not inspected after selection", parsed: []string{doi}, malformed: arxiv, pdfs: []string{pmid}, wantParsed: doi, wantPDF: pmid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Create(library.PaperRecord{Title: "Mixed identifier record", Identifiers: library.Identifiers{DOI: "10.1000/ap", ArXivID: "1706.03762", PMID: "12345"}}); err != nil {
				t.Fatal(err)
			}
			records, err := store.List()
			if err != nil || len(records) != 1 {
				t.Fatalf("List: records=%+v err=%v", records, err)
			}
			id := records[0].RecordID
			passage := func(stem string) string { return "Selected " + stem + " <script>not executable</script>" }
			pdfBytes := func(stem string) string { return "%PDF-1.4 local " + stem }
			for _, stem := range tc.parsed {
				writeParsedDoc(t, dir, stem, strings.ReplaceAll(sampleParsedDoc, "Photosynthesis converts sunlight.", passage(stem)))
			}
			for _, stem := range tc.pdfs {
				writeLocalPDF(t, dir, stem)
				if err := os.WriteFile(filepath.Join(dir, "documents", "open-access", stem+".pdf"), []byte(pdfBytes(stem)), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.malformed != "" {
				writeParsedDoc(t, dir, tc.malformed, "{malformed")
			}
			var symlinkPath string
			if tc.symlinkPDF != "" {
				external := t.TempDir()
				writeLocalPDF(t, external, tc.symlinkPDF)
				symlinkPath = filepath.Join(dir, "documents", "open-access", tc.symlinkPDF+".pdf")
				if err := os.MkdirAll(filepath.Dir(symlinkPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(external, "documents", "open-access", tc.symlinkPDF+".pdf"), symlinkPath); err != nil {
					t.Fatal(err)
				}
			}

			workspace, found, workspaceErr := BuildPaperWorkspace(dir, id)
			list, listErr := BuildLibraryViewModel(dir)
			router := NewRouter(Config{ProjectPath: dir})
			get := func(path string) *httptest.ResponseRecorder {
				t.Helper()
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				return response
			}
			if tc.wantError {
				if workspaceErr == nil || listErr == nil {
					t.Errorf("malformed parsed JSON: workspace error=%v, list error=%v", workspaceErr, listErr)
				}
				for _, path := range []string{"/library", "/library/" + id, "/library/" + id + "/pdf"} {
					if response := get(path); response.Code != http.StatusInternalServerError {
						t.Errorf("GET %s status=%d, want 500", path, response.Code)
					}
				}
				return
			}
			if workspaceErr != nil || listErr != nil || !found || len(list.Rows) != 1 {
				t.Fatalf("builders: found=%v workspace error=%v list=%+v list error=%v", found, workspaceErr, list, listErr)
			}
			wantParsed, wantPDF := tc.wantParsed != "", tc.wantPDF != ""
			if workspace.Parsed != wantParsed || workspace.HasPDF != wantPDF || list.Rows[0].Parsed != wantParsed || list.Rows[0].HasPDF != wantPDF {
				t.Errorf("availability: workspace parsed=%v PDF=%v, list parsed=%v PDF=%v; want parsed=%v PDF=%v", workspace.Parsed, workspace.HasPDF, list.Rows[0].Parsed, list.Rows[0].HasPDF, wantParsed, wantPDF)
			}
			if wantParsed && (len(workspace.Sections) != 1 || len(workspace.Sections[0].Passages) != 1 || workspace.Sections[0].Passages[0].Text != passage(tc.wantParsed)) {
				t.Errorf("selected parsed passages=%+v, want %q", workspace.Sections, passage(tc.wantParsed))
			}
			for _, path := range []string{"/library", "/library/" + id} {
				response := get(path)
				body := response.Body.String()
				if response.Code != http.StatusOK || strings.Contains(body, "Parsed text") != wantParsed || strings.Contains(body, ">PDF ") != wantPDF {
					t.Errorf("GET %s: status=%d body=%s", path, response.Code, body)
				}
				if path != "/library" {
					if strings.Contains(body, `<embed src="/library/`+id+`/pdf" type="application/pdf"`) != wantPDF {
						t.Errorf("workspace PDF embed disagrees with availability: %s", body)
					}
					if wantParsed && (!strings.Contains(body, html.EscapeString(passage(tc.wantParsed))) || strings.Contains(body, "<script>not executable</script>")) {
						t.Errorf("workspace missing safely escaped selected passage: %s", body)
					}
				}
			}
			response := get("/library/" + id + "/pdf")
			if wantPDF {
				if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/pdf" || response.Body.String() != pdfBytes(tc.wantPDF) {
					t.Errorf("PDF: status=%d type=%q body=%q, want %q", response.Code, response.Header().Get("Content-Type"), response.Body.String(), pdfBytes(tc.wantPDF))
				}
			} else if response.Code != http.StatusNotFound {
				t.Errorf("missing PDF: status=%d body=%q", response.Code, response.Body.String())
			}
			if symlinkPath != "" {
				if info, err := os.Lstat(symlinkPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
					t.Errorf("PDF symlink changed: info=%v err=%v", info, err)
				}
			}
		})
	}
}
