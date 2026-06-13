package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectHandlerRendersCreateAndOpenScreen(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()

	NewProjectHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Projects",
		"Create project",
		"Open project",
		"hx-post=\"/projects/create\"",
		"hx-post=\"/projects/open\"",
		"name=\"title\"",
		"name=\"path\"",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("project screen missing %q:\n%s", want, body)
		}
	}
}

func TestCreateProjectHandlerCreatesWorkspace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	form := url.Values{"title": {"Demo Review"}, "path": {dir}}
	req := httptest.NewRequest(http.MethodPost, "/projects/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	NewCreateProjectHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Created project", "Demo Review", dir, "rforge.project.toml"} {
		if !strings.Contains(body, want) {
			t.Fatalf("create response missing %q:\n%s", want, body)
		}
	}
	if rec.Header().Get("HX-Trigger") != "project-opened" {
		t.Fatalf("HX-Trigger = %q", rec.Header().Get("HX-Trigger"))
	}
}

func TestOpenProjectHandlerOpensExistingWorkspace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	createForm := url.Values{"title": {"Demo Review"}, "path": {dir}}
	createReq := httptest.NewRequest(http.MethodPost, "/projects/create", strings.NewReader(createForm.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	NewCreateProjectHandler().ServeHTTP(httptest.NewRecorder(), createReq)

	openForm := url.Values{"path": {dir}}
	openReq := httptest.NewRequest(http.MethodPost, "/projects/open", strings.NewReader(openForm.Encode()))
	openReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	openRec := httptest.NewRecorder()

	NewOpenProjectHandler().ServeHTTP(openRec, openReq)

	if openRec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", openRec.Code, openRec.Body.String())
	}
	body := openRec.Body.String()
	for _, want := range []string{"Opened project", "Demo Review", dir, "sqlite"} {
		if !strings.Contains(body, want) {
			t.Fatalf("open response missing %q:\n%s", want, body)
		}
	}
	if openRec.Header().Get("HX-Trigger") != "project-opened" {
		t.Fatalf("HX-Trigger = %q", openRec.Header().Get("HX-Trigger"))
	}
}
