package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
)

func TestProjectNativePostFallback(t *testing.T) {
	alpha := seedLibraryFolder(t, "Alpha")
	router := NewRouter(Config{ProjectPath: alpha})
	get := func(path string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	projects := get("/projects")
	active := get("/projects/active")
	for _, tc := range []struct{ operation, body string }{
		{"create", projects},
		{"open", projects},
		{"switch", active},
	} {
		native := `method="post" action="/projects/` + tc.operation + `"`
		if !strings.Contains(tc.body, native) {
			t.Errorf("%s form lacks native %s", tc.operation, native)
		}
	}
	if !strings.Contains(projects, `hx-post="/projects/create" hx-target="#project-status"`) ||
		!strings.Contains(projects, `hx-post="/projects/open" hx-target="#project-status"`) {
		t.Error("create/open forms must keep the htmx progressive enhancement")
	}
	if !strings.Contains(active, `hx-post="/projects/switch" hx-target="#active-project-path" hx-swap="outerHTML"`) {
		t.Error("switch form must keep the htmx progressive enhancement")
	}
}

func TestProjectFormsNativeSubmission(t *testing.T) {
	alpha := seedLibraryFolder(t, "Alpha")
	router := NewRouter(Config{ProjectPath: alpha})
	post := func(path string, form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	active := func() string {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/projects/active", nil))
		return rec.Body.String()
	}

	created := filepath.Join(t.TempDir(), "created")
	rec := post("/projects/create", url.Values{"title": {"Native"}, "path": {created}})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Created project") {
		t.Fatalf("native create: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(active(), alpha) {
		t.Fatal("native create must not change the active folder")
	}

	opened, err := project.Create(filepath.Join(t.TempDir(), "opened"), project.CreateOptions{Title: "Opened"})
	if err != nil {
		t.Fatal(err)
	}
	rec = post("/projects/open", url.Values{"path": {opened.Path}})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Opened project") {
		t.Fatalf("native open: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(active(), alpha) {
		t.Fatal("native open must not change the active folder")
	}

	beta := seedLibraryFolder(t, "Beta")
	rec = post("/projects/switch", url.Values{"path": {beta}})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `<p id="active-project-path">`+beta+`</p>`) {
		t.Fatalf("native switch: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "refresh-library") {
		t.Fatalf("native switch HX-Trigger = %q", rec.Header().Get("HX-Trigger"))
	}

	for _, tc := range []struct{ operation, diagnostic string }{
		{"create", "project title is required"},
		{"open", "rforge.project.toml"},
		{"switch", "research folder does not exist"},
	} {
		rec := post("/projects/"+tc.operation, url.Values{"path": {filepath.Join(t.TempDir(), "absent")}, "title": {" "}})
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.diagnostic) || rec.Header().Get("HX-Trigger") != "" {
			t.Fatalf("rejected %s: %d %s %v", tc.operation, rec.Code, rec.Body.String(), rec.Header())
		}
	}
}
