package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func seedLibraryFolder(t *testing.T, title string) string {
	t.Helper()
	dir := t.TempDir()
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: title, Identifiers: library.Identifiers{DOI: "10.1000/" + title}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return dir
}

func TestProjectSwitcherChangesActiveFolder(t *testing.T) {
	alpha := seedLibraryFolder(t, "Alpha")
	beta := seedLibraryFolder(t, "Beta")

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: alpha}))
	defer ts.Close()

	body, _, _ := getURL(t, ts.URL+"/library")
	if !strings.Contains(body, "Alpha") || strings.Contains(body, "Beta") {
		t.Fatalf("initial /library should show Alpha only: %s", body)
	}

	resp, err := http.PostForm(ts.URL+"/projects/switch", url.Values{"path": {beta}})
	if err != nil {
		t.Fatalf("switch: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	body2, _, _ := getURL(t, ts.URL+"/library")
	if !strings.Contains(body2, "Beta") || strings.Contains(body2, "Alpha") {
		t.Fatalf("after switch /library should show Beta only: %s", body2)
	}
}

func TestProjectSwitcherRejectsMissingFolder(t *testing.T) {
	alpha := seedLibraryFolder(t, "Alpha")
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: alpha}))
	defer ts.Close()

	resp, err := http.PostForm(ts.URL+"/projects/switch", url.Values{"path": {"/no/such/research/folder"}})
	if err != nil {
		t.Fatalf("switch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("switch to missing folder status = %d, want 400", resp.StatusCode)
	}

	// Active project unchanged.
	body, _, _ := getURL(t, ts.URL+"/library")
	if !strings.Contains(body, "Alpha") {
		t.Fatalf("active project should be unchanged: %s", body)
	}
}

func TestProjectsActiveFragmentShowsCurrentFolder(t *testing.T) {
	alpha := seedLibraryFolder(t, "Alpha")
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: alpha}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/projects/active")
	if status != http.StatusOK {
		t.Fatalf("GET /projects/active status = %d", status)
	}
	if !strings.Contains(body, alpha) || !strings.Contains(body, "/projects/switch") {
		t.Fatalf("active fragment missing path/switch form: %s", body)
	}
}
