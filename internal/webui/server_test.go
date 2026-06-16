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
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/library")
	if status != http.StatusOK {
		t.Fatalf("GET /library status = %d", status)
	}
	if !strings.Contains(body, "Artificial Photosynthesis Review") {
		t.Fatalf("library body missing paper title: %s", body)
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
