package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func getURL(t *testing.T, url string) (body string, status int, contentType string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s body: %v", url, err)
	}
	return string(data), resp.StatusCode, resp.Header.Get("Content-Type")
}

func TestNewRouterServesShellAndStaticAssets(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d", status)
	}
	if !strings.Contains(body, "ResearchForge") {
		t.Fatalf("GET / body missing title: %s", body)
	}

	cssBody, cssStatus, cssType := getURL(t, ts.URL+"/assets/researchforge.css")
	if cssStatus != http.StatusOK {
		t.Fatalf("GET css status = %d", cssStatus)
	}
	if !strings.Contains(cssType, "css") {
		t.Fatalf("css content-type = %q", cssType)
	}
	if !strings.Contains(cssBody, ".rf-shell") {
		t.Fatalf("css body missing expected rule: %s", cssBody)
	}
}

func TestNewRouterUnknownPathIs404(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()

	_, status, _ := getURL(t, ts.URL+"/does-not-exist")
	if status != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist status = %d, want 404", status)
	}
}

func TestNewRouterServesLibraryFromProjectFolder(t *testing.T) {
	dir := t.TempDir()
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(library.PaperRecord{
		Title:       "Artificial Photosynthesis Review",
		Identifiers: library.Identifiers{DOI: "10.1000/ap"},
		Authors:     []library.Author{{Given: "Ada", Family: "Lovelace"}},
		Year:        2026,
		SourceRefs:  []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"collections": "Solar fuels", "tags": "catalysis"}}},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	records, err := store.List()
	if err != nil || len(records) != 1 {
		t.Fatalf("List after create: records=%#v err=%v", records, err)
	}

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/library")
	if status != http.StatusOK {
		t.Fatalf("GET /library status = %d", status)
	}
	for _, want := range []string{
		"Artificial Photosynthesis Review",
		"Ada Lovelace",
		"2026",
		"Solar fuels",
		"catalysis",
		`href="/library/` + records[0].RecordID + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("library body missing %q: %s", want, body)
		}
	}
}

func TestLibraryRecordRouteServesMetadataOnlyWorkspace(t *testing.T) {
	dir := t.TempDir()
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(library.PaperRecord{
		Title:       "Metadata-only Research Record",
		Identifiers: library.Identifiers{DOI: "10.1000/metadata-only"},
		Authors:     []library.Author{{Given: "Grace", Family: "Hopper"}},
		Year:        2025,
		Venue:       "Journal of Durable Libraries",
		SourceRefs:  []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"collections": "Methods", "tags": "reproducibility"}}},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	records, err := store.List()
	if err != nil || len(records) != 1 {
		t.Fatalf("List: records=%#v err=%v", records, err)
	}

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/library/"+records[0].RecordID)
	if status != http.StatusOK {
		t.Fatalf("GET /library/{id} status = %d, body=%s", status, body)
	}
	for _, want := range []string{"Metadata-only Research Record", "Grace Hopper", "10.1000/metadata-only", "Journal of Durable Libraries", "Methods", "reproducibility", "Metadata only"} {
		if !strings.Contains(body, want) {
			t.Fatalf("workspace missing %q: %s", want, body)
		}
	}
}

func TestLibraryRecordRouteJoinsParsedTextAndPDF(t *testing.T) {
	dir := t.TempDir()
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: "Artificial Photosynthesis Review", Identifiers: library.Identifiers{DOI: "10.1000/ap"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	records, err := store.List()
	if err != nil || len(records) != 1 {
		t.Fatalf("List: records=%#v err=%v", records, err)
	}
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeLocalPDF(t, dir, "10-1000-ap")

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/library/"+records[0].RecordID)
	if status != http.StatusOK {
		t.Fatalf("GET workspace status = %d, body=%s", status, body)
	}
	for _, want := range []string{"We review water-splitting catalysts.", "Introduction", "Photosynthesis converts sunlight.", "/library/" + records[0].RecordID + "/pdf", "Parsed text"} {
		if !strings.Contains(body, want) {
			t.Fatalf("workspace missing %q: %s", want, body)
		}
	}
	pdfBody, pdfStatus, pdfType := getURL(t, ts.URL+"/library/"+records[0].RecordID+"/pdf")
	if pdfStatus != http.StatusOK || !strings.Contains(pdfType, "application/pdf") || !strings.HasPrefix(pdfBody, "%PDF") {
		t.Fatalf("workspace PDF status=%d type=%q body=%q", pdfStatus, pdfType, pdfBody)
	}
}

func TestLibraryListShowsResolvedAssetStatus(t *testing.T) {
	dir := t.TempDir()
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: "Readable record", Identifiers: library.Identifiers{DOI: "10.1000/ap"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeLocalPDF(t, dir, "10-1000-ap")

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/library")
	if status != http.StatusOK {
		t.Fatalf("GET /library status = %d", status)
	}
	if !strings.Contains(body, "PDF") || !strings.Contains(body, "Parsed text") {
		t.Fatalf("library missing resolved asset status: %s", body)
	}
}

func TestRoutesIncludesCoreRoutes(t *testing.T) {
	routes := Routes()
	for _, want := range []string{"/", "/library", "/artifacts", "/projects", "/search"} {
		found := false
		for _, r := range routes {
			if r == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Routes() = %v, missing %q", routes, want)
		}
	}
}
