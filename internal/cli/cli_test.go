package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteProjectCreateWritesProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created project") {
		t.Fatalf("stdout missing success message: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "rforge.project.toml")); err != nil {
		t.Fatalf("manifest not created: %v", err)
	}
}

func TestExecuteVersion(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"version"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rforge") {
		t.Fatalf("stdout missing version prefix: %q", stdout.String())
	}
}

func TestExecuteProjectInspect(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"project", "inspect", dir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Demo Review") {
		t.Fatalf("stdout missing project title: %q", stdout.String())
	}
}

func TestExecuteVersionJSONEnvelope(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--log-level", "debug", "version"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope["ok"] != true {
		t.Fatalf("ok = %#v, want true", envelope["ok"])
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("data missing or wrong type: %#v", envelope["data"])
	}
	if data["name"] != "rforge" {
		t.Fatalf("data.name = %#v, want rforge", data["name"])
	}
}

func TestExecuteProjectInspectJSONEnvelope(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", dir, "project", "inspect", dir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	if data["title"] != "Demo Review" {
		t.Fatalf("data.title = %#v, want Demo Review", data["title"])
	}
	if data["manifestPath"] != filepath.Join(dir, "rforge.project.toml") {
		t.Fatalf("data.manifestPath = %#v", data["manifestPath"])
	}
	if data["lockfilePath"] != filepath.Join(dir, "rforge.lock.json") {
		t.Fatalf("data.lockfilePath = %#v", data["lockfilePath"])
	}
	if data["provenancePath"] != filepath.Join(dir, "provenance", "events.jsonl") {
		t.Fatalf("data.provenancePath = %#v", data["provenancePath"])
	}
	if data["storagePath"] != filepath.Join(dir, "data", "rforge.sqlite") {
		t.Fatalf("data.storagePath = %#v", data["storagePath"])
	}
	if data["archiveMetadataPath"] != filepath.Join(dir, "rforge.archive.json") {
		t.Fatalf("data.archiveMetadataPath = %#v", data["archiveMetadataPath"])
	}
}

func TestExecuteUnknownCommandJSONErrorEnvelope(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "unknown"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON errors", stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope["ok"] != false {
		t.Fatalf("ok = %#v, want false", envelope["ok"])
	}
	errorBody := envelope["error"].(map[string]any)
	if errorBody["code"] != "unknown_command" {
		t.Fatalf("error.code = %#v, want unknown_command", errorBody["code"])
	}
}

func TestExecuteDoctorJSONChecksRuntimeAndProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", dir, "doctor"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	checks := data["checks"].([]any)
	if len(checks) < 3 {
		t.Fatalf("len(checks) = %d, want at least 3", len(checks))
	}
	var sawRuntime, sawManifest, sawLockfile, sawSQLite bool
	for _, raw := range checks {
		check := raw.(map[string]any)
		action, ok := check["action"].(string)
		if !ok || action == "" {
			t.Fatalf("check missing actionable guidance: %#v", check)
		}
		switch check["name"] {
		case "go_runtime":
			sawRuntime = check["ok"] == true
		case "project_manifest":
			sawManifest = check["ok"] == true
		case "project_lockfile":
			sawLockfile = check["ok"] == true
		case "sqlite":
			sawSQLite = check["ok"] == true
		}
	}
	if !sawRuntime || !sawManifest || !sawLockfile || !sawSQLite {
		t.Fatalf("missing passing checks: runtime=%v manifest=%v lockfile=%v sqlite=%v checks=%#v", sawRuntime, sawManifest, sawLockfile, sawSQLite, checks)
	}
}

func TestExecuteDoctorJSONChecksConfiguredGROBIDEndpoint(t *testing.T) {
	t.Setenv("RFORGE_GROBID_URL", "http://localhost:8070")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "doctor"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	checks := data["checks"].([]any)
	for _, raw := range checks {
		check := raw.(map[string]any)
		if check["name"] == "grobid_endpoint" {
			if check["ok"] != true {
				t.Fatalf("grobid endpoint check failed: %#v", check)
			}
			if check["message"] != "http://localhost:8070" {
				t.Fatalf("grobid endpoint message = %#v", check["message"])
			}
			return
		}
	}
	t.Fatalf("missing grobid_endpoint check: %#v", checks)
}

func TestExecuteDoctorJSONChecksConfiguredOpenSearchEndpoint(t *testing.T) {
	t.Setenv("RFORGE_OPENSEARCH_URL", "http://localhost:9200")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "doctor"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	checks := data["checks"].([]any)
	for _, raw := range checks {
		check := raw.(map[string]any)
		if check["name"] == "opensearch_endpoint" {
			if check["ok"] != true {
				t.Fatalf("opensearch endpoint check failed: %#v", check)
			}
			if check["message"] != "http://localhost:9200" {
				t.Fatalf("opensearch endpoint message = %#v", check["message"])
			}
			return
		}
	}
	t.Fatalf("missing opensearch_endpoint check: %#v", checks)
}

func TestExecuteDoctorJSONChecksConfiguredQdrantEndpoint(t *testing.T) {
	t.Setenv("RFORGE_QDRANT_URL", "http://localhost:6333")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "doctor"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	checks := data["checks"].([]any)
	for _, raw := range checks {
		check := raw.(map[string]any)
		if check["name"] == "qdrant_endpoint" {
			if check["ok"] != true {
				t.Fatalf("qdrant endpoint check failed: %#v", check)
			}
			if check["message"] != "http://localhost:6333" {
				t.Fatalf("qdrant endpoint message = %#v", check["message"])
			}
			return
		}
	}
	t.Fatalf("missing qdrant_endpoint check: %#v", checks)
}

func TestExecuteDoctorJSONChecksConfiguredRMetafor(t *testing.T) {
	dir := t.TempDir()
	rscript := filepath.Join(dir, "Rscript")
	if err := os.WriteFile(rscript, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake Rscript: %v", err)
	}
	t.Setenv("RFORGE_RSCRIPT_PATH", rscript)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "doctor"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	checks := data["checks"].([]any)
	for _, raw := range checks {
		check := raw.(map[string]any)
		if check["name"] == "r_metafor" {
			if check["ok"] != true {
				t.Fatalf("r_metafor check failed: %#v", check)
			}
			if check["message"] != rscript {
				t.Fatalf("r_metafor message = %#v", check["message"])
			}
			return
		}
	}
	t.Fatalf("missing r_metafor check: %#v", checks)
}

func TestExecuteProjectListJSON(t *testing.T) {
	root := t.TempDir()
	if code := Execute([]string{"project", "create", filepath.Join(root, "alpha"), "--title", "Alpha"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create alpha exit code = %d", code)
	}
	if code := Execute([]string{"project", "create", filepath.Join(root, "beta"), "--title", "Beta"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create beta exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "project", "list", root}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	projects := data["projects"].([]any)
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}
	first := projects[0].(map[string]any)
	if first["title"] != "Alpha" {
		t.Fatalf("first title = %#v, want Alpha", first["title"])
	}
}
