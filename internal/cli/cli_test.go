package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

func TestExecuteHelpMentionsDecisionCompletionAudit(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--help"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"rforge decisions --check TODO.md", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, stdout.String())
		}
	}
}

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

func TestExecuteProjectCreateRecordsCLICommandProvenance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "project", "create", dir, "--title", "Demo Review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	events, err := provenance.Read(dir)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	var sawCommand bool
	for _, event := range events {
		if event.Action != "cli.command" {
			continue
		}
		if event.Target != dir {
			t.Fatalf("cli.command target = %q, want %q", event.Target, dir)
		}
		if event.Inputs["command"] != "project create" {
			t.Fatalf("cli.command command = %#v", event.Inputs["command"])
		}
		if event.Inputs["json"] != true {
			t.Fatalf("cli.command json = %#v", event.Inputs["json"])
		}
		if event.Outputs["exitCode"] != float64(0) {
			t.Fatalf("cli.command exitCode = %#v", event.Outputs["exitCode"])
		}
		sawCommand = true
	}
	if !sawCommand {
		t.Fatalf("missing cli.command provenance event: %#v", events)
	}
}

func TestExecuteDecisionsCheckTODO(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--check", filepath.Join("..", "..", "TODO.md")}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "all unchecked TODO items are decision/tracker-covered") {
		t.Fatalf("check output = %s", stdout.String())
	}
}

func TestExecuteDecisionsIssueBodyRejectsResolvedDecisions(t *testing.T) {
	for _, id := range []string{"project_license", "web_gui_stack_scope"} {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		code := Execute([]string{"decisions", "--issue-body", id}, stdout, stderr)
		if code == 0 {
			t.Fatalf("expected non-zero exit for resolved decision %q", id)
		}
		if !strings.Contains(stderr.String(), "unknown decision") || !strings.Contains(stderr.String(), id) {
			t.Fatalf("stderr missing resolved decision rejection for %q: %s", id, stderr.String())
		}
	}
}

func TestExecuteDecisionsUsageMentionsMarkdownMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--bogus"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit")
	}
	for _, want := range []string{"--markdown", "--completion-audit <todo-file> <audit-file>"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("usage missing %q:\n%s", want, stderr.String())
		}
	}
}

func TestExecuteDecisionsJSONHasNoActiveOwnerBlockers(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"decisions\":[]") {
		t.Fatalf("expected empty owner-decision registry after license resolution, got:\n%s", stdout.String())
	}
}

func TestExecuteDecisionsJSONIncludesExactUncheckedTODOLineReferences(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, lineRef := range uncheckedTODOLineRefs(t) {
		if !strings.Contains(stdout.String(), lineRef) {
			t.Fatalf("json output missing TODO line reference %q:\n%s", lineRef, stdout.String())
		}
	}
}

func TestExecuteDecisionsMarkdownIncludesTodoLineReferences(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--markdown"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, lineRef := range uncheckedTODOLineRefs(t) {
		if !strings.Contains(stdout.String(), lineRef) {
			t.Fatalf("markdown output missing TODO line reference %q:\n%s", lineRef, stdout.String())
		}
	}
}

func TestExecuteDecisionsCompletionAuditVerifiesAuditDocument(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--completion-audit", filepath.Join("..", "..", "TODO.md"), filepath.Join("..", "..", "docs", "todo-completion-audit.md")}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"completion audit verified", "all unchecked TODO items are decision/tracker-covered", "issue references verified", "unchecked TODO refs verified: 0", "completion blocked by 0 owner decision(s)", "license resolution verified", "checked TODO evidence verified", "quality gate verified"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("completion audit output missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "remains pending") || strings.Contains(stdout.String(), "absent") {
		t.Fatalf("resolved completion audit should not report pending license state:\n%s", stdout.String())
	}
}

func TestExecuteDecisionsCompletionAuditJSONIsSingleEnvelope(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions", "--completion-audit", filepath.Join("..", "..", "TODO.md"), filepath.Join("..", "..", "docs", "todo-completion-audit.md")}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if strings.Count(out, "\n") != 1 || strings.Contains(out, "all unchecked TODO items are decision/tracker-covered") || !json.Valid([]byte(out)) {
		t.Fatalf("expected a single JSON envelope, got:\n%s", out)
	}
	for _, want := range []string{"\"completion_audit_verified\":true", "\"completion_audit_issue_refs_verified\":true", "\"checked_evidence_verified\":true", "\"quality_gate_verified\":true", "\"completion_blocked\":false", "\"blocked_decisions\":0", "\"blocked_decision_ids\":[]", "\"blocked_issue_urls\":[]", "\"line_refs_verified\":true", "\"issue_refs_verified\":true", "\"unchecked_refs\":0", "\"license_resolution_verified\":true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("json completion audit output missing %q:\n%s", want, out)
		}
	}
}

func TestExecuteDecisionsCheckReportsAllUncheckedTODOReferences(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions", "--check", filepath.Join("..", "..", "TODO.md")}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"line_refs_verified\":true", "\"issue_refs_verified\":true", "\"unchecked_refs\":0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json check output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsCheckVerifiesLineReferences(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--check", filepath.Join("..", "..", "TODO.md")}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "line references verified") {
		t.Fatalf("check output missing line reference verification:\n%s", stdout.String())
	}
}

func uncheckedTODOLineRefs(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	refs := []string{}
	inBacklog := false
	for i, line := range strings.Split(string(data), "\n") {
		if isTodoBacklogHeading(line) {
			inBacklog = true
		}
		if !inBacklog && strings.HasPrefix(strings.TrimSpace(line), "- [ ] ") {
			refs = append(refs, fmt.Sprintf("TODO.md:%d", i+1))
		}
	}
	return refs
}

func TestExecuteUIJSONReportsServingConfig(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", "/tmp/example", "ui", "--addr", ":9999"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"serving_ready", "go+htmx", ":9999", "/tmp/example", "/library"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ui output missing %q: %s", want, out)
		}
	}
}

func TestExecuteUIAddrDefaultsAndEnvOverride(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "ui"}, stdout, stderr); code != 0 {
		t.Fatalf("default addr exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), ":8080") {
		t.Fatalf("default addr output = %s", stdout.String())
	}

	t.Setenv("RFORGE_UI_ADDR", ":7000")
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "ui"}, stdout, stderr); code != 0 {
		t.Fatalf("env addr exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), ":7000") {
		t.Fatalf("env addr output = %s", stdout.String())
	}
}

func TestExecuteUIRejectsUnknownFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"ui", "--bogus"}, stdout, stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %s", code, stderr.String())
	}
}

func TestExecuteCompletionBash(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"completion", "bash"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "_rforge_completion") || !strings.Contains(stdout.String(), "project doctor service") {
		t.Fatalf("completion script = %s", stdout.String())
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

func TestExecuteServiceStartStopUsesSafeLocalState(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("RFORGE_SERVICE_STATE_DIR", stateDir)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	if code := Execute([]string{"--json", "service", "start", "grobid"}, stdout, stderr); code != 0 {
		t.Fatalf("service start exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "grobid.started")); err != nil {
		t.Fatalf("missing service state marker: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "service", "stop", "grobid"}, stdout, stderr); code != 0 {
		t.Fatalf("service stop exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "grobid.started")); !os.IsNotExist(err) {
		t.Fatalf("service marker still exists or unexpected stat err: %v", err)
	}
}

func TestExecuteServiceCheckGROBIDJSON(t *testing.T) {
	t.Setenv("RFORGE_GROBID_URL", "http://localhost:8070")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "service", "check", "grobid"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	check := data["check"].(map[string]any)
	if check["name"] != "grobid_endpoint" {
		t.Fatalf("check.name = %#v", check["name"])
	}
	if check["ok"] != true {
		t.Fatalf("check.ok = %#v, want true", check["ok"])
	}
	if check["message"] != "http://localhost:8070" {
		t.Fatalf("check.message = %#v", check["message"])
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

func TestExecuteSearchArXivJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("search_query") != "all:artificial photosynthesis" {
			t.Fatalf("search_query = %q", r.URL.Query().Get("search_query"))
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><entry><id>http://arxiv.org/abs/2401.00001v1</id><title>Artificial photosynthesis preprint</title><summary>Fixture.</summary><published>2026-01-02T00:00:00Z</published></entry></feed>`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_ARXIV_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "arxiv", "--query", "artificial photosynthesis", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	papers := data["papers"].([]any)
	if len(papers) != 1 {
		t.Fatalf("len(papers) = %d, want 1", len(papers))
	}
	paper := papers[0].(map[string]any)
	if paper["Title"] != "Artificial photosynthesis preprint" {
		t.Fatalf("paper title = %#v", paper["Title"])
	}
}

func TestExecuteSearchOpenAlexAdvancedFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		wantFilter := "from_publication_date:2020-01-01,to_publication_date:2021-12-31,type:article,is_oa:true,concepts.id:C41008148"
		if got := r.URL.Query().Get("filter"); got != wantFilter {
			t.Fatalf("filter = %q, want %q", got, wantFilter)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W123","title":"Filtered"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "--source", "openalex", "--query", "test", "--from-year", "2020", "--to-year", "2021", "--type", "article", "--open-access", "true", "--concept", "C41008148"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("search exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestExecuteSearchOpenAlexJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "artificial photosynthesis" {
			t.Fatalf("search = %q", r.URL.Query().Get("search"))
		}
		if r.URL.Query().Get("filter") != "type:review" {
			t.Fatalf("filter = %q", r.URL.Query().Get("filter"))
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W123","doi":"https://doi.org/10.1000/example","title":"Artificial photosynthesis catalyst review","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "openalex", "--query", "artificial photosynthesis", "--limit", "1", "--filter", "type:review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	papers := data["papers"].([]any)
	if len(papers) != 1 {
		t.Fatalf("len(papers) = %d, want 1", len(papers))
	}
	paper := papers[0].(map[string]any)
	if paper["Title"] != "Artificial photosynthesis catalyst review" {
		t.Fatalf("paper title = %#v", paper["Title"])
	}
}

func TestExecuteCitationsExpandCrossrefReferencesWritesGraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.1000/source" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"DOI":"10.1000/source","title":["Source paper"],"reference":[{"DOI":"10.1000/ref-a","article-title":"Reference A"},{"DOI":"10.1000/ref-b","article-title":"Reference B"}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_CROSSREF_URL", server.URL)
	project := filepath.Join(t.TempDir(), "crossref-graph-project")
	if code := Execute([]string{"project", "create", project, "--title", "Crossref Graph"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	out := filepath.Join(t.TempDir(), "crossref-graph.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "citations", "expand", "--source", "crossref", "--paper", "10.1000/source", "--direction", "references", "--out", out, "--import-library"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read graph: %v", err)
	}
	for _, want := range []string{`"source": "10.1000/source"`, `"target": "10.1000/ref-a"`, `"target": "10.1000/ref-b"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("graph missing %s:\n%s", want, data)
		}
	}
	if !strings.Contains(stdout.String(), `"edges":2`) || !strings.Contains(stdout.String(), `"imported":2`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteCitationsExpandOpenAlexWritesGraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/works/W123":
			_, _ = w.Write([]byte(`{"id":"https://openalex.org/W123","referenced_works":["https://openalex.org/WREF1"]}`))
		case "/works":
			_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/WCITE1","title":"Citing OpenAlex work"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := filepath.Join(t.TempDir(), "openalex-graph.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "citations", "expand", "--source", "openalex", "--paper", "W123", "--direction", "both", "--limit", "1", "--out", out}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read graph: %v", err)
	}
	for _, want := range []string{`"source": "W123"`, `"target": "WREF1"`, `"source": "WCITE1"`, `"target": "W123"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("graph missing %s:\n%s", want, data)
		}
	}
}

func TestExecuteCitationsExpandSemanticScholarRecursiveWithProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graph/v1/paper/seed/references":
			_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"ref-1","title":"Reference one","externalIds":{"DOI":"10.1000/ref1"}}}]}`))
		case "/graph/v1/paper/ref-1/references":
			_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"ref-2","title":"Reference two","externalIds":{"DOI":"10.1000/ref2"}}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	project := filepath.Join(t.TempDir(), "recursive-graph-project")
	if code := Execute([]string{"project", "create", project, "--title", "Recursive Citation Graph"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	out := filepath.Join(t.TempDir(), "recursive-citation-graph.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", project, "citations", "expand", "--source", "semantic-scholar", "--paper", "seed", "--direction", "references", "--limit", "1", "--depth", "2", "--out", out, "--import-library"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read graph: %v", err)
	}
	for _, want := range []string{`"source": "seed"`, `"target": "ref-1"`, `"source": "ref-1"`, `"target": "ref-2"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("recursive graph missing %s:\n%s", want, data)
		}
	}
	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Edges    int `json:"edges"`
			Imported int `json:"imported"`
			Depth    int `json:"depth"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK || envelope.Data.Edges != 2 || envelope.Data.Imported != 2 || envelope.Data.Depth != 2 {
		t.Fatalf("envelope = %#v", envelope)
	}
	events, err := provenance.Read(project)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	for _, event := range events {
		if event.Action == "citations.expand" && event.Outputs["edges"] == float64(2) && event.Outputs["imported"] == float64(2) {
			return
		}
	}
	t.Fatalf("missing citations.expand provenance event: %#v", events)
}

func TestExecuteCitationsExpandSemanticScholarHonorsMaxRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graph/v1/paper/seed/references":
			_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"ref-1","title":"Reference one"}}]}`))
		case "/graph/v1/paper/ref-1/references":
			_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"ref-2","title":"Reference two"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	out := filepath.Join(t.TempDir(), "limited-citation-graph.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "citations", "expand", "--source", "semantic-scholar", "--paper", "seed", "--direction", "references", "--limit", "1", "--depth", "2", "--max-records", "1", "--out", out}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read graph: %v", err)
	}
	if !strings.Contains(string(data), `"target": "ref-1"`) || strings.Contains(string(data), `"target": "ref-2"`) {
		t.Fatalf("max-record graph not limited correctly:\n%s", data)
	}
	var envelope struct {
		Data struct {
			Edges      int `json:"edges"`
			MaxRecords int `json:"maxRecords"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.Data.Edges != 1 || envelope.Data.MaxRecords != 1 {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestExecuteCitationsExpandSemanticScholarWritesGraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graph/v1/paper/seed/references":
			_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"ref-1","title":"Reference one","externalIds":{"DOI":"10.1000/ref"}}}]}`))
		case "/graph/v1/paper/seed/citations":
			_, _ = w.Write([]byte(`{"data":[{"citingPaper":{"paperId":"citing-1","title":"Citing one","externalIds":{"DOI":"10.1000/citing"}}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	project := filepath.Join(t.TempDir(), "graph-project")
	if code := Execute([]string{"project", "create", project, "--title", "Citation Graph Import"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	out := filepath.Join(t.TempDir(), "citation-graph.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", project, "citations", "expand", "--source", "semantic-scholar", "--paper", "seed", "--direction", "both", "--limit", "1", "--out", out, "--import-library"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read graph: %v", err)
	}
	for _, want := range []string{`"source": "citing-1"`, `"target": "seed"`, `"source": "seed"`, `"target": "ref-1"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("graph missing %s:\n%s", want, data)
		}
	}
	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Imported int `json:"imported"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK || envelope.Data.Imported != 2 {
		t.Fatalf("envelope = %#v", envelope)
	}
	var lib struct {
		Data struct {
			Papers []library.PaperRecord `json:"papers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", project, "library", "list"), &lib); err != nil {
		t.Fatalf("decode library list: %v", err)
	}
	if len(lib.Data.Papers) != 2 {
		t.Fatalf("library papers = %#v, want 2 imported graph records", lib.Data.Papers)
	}
}

func TestExecuteCitationsReportWritesMarkdownSummary(t *testing.T) {
	graphPath := filepath.Join(t.TempDir(), "graph.json")
	reportPath := filepath.Join(t.TempDir(), "graph-report.md")
	graph := `{"nodes":[{"id":"paper-a"},{"id":"paper-b"},{"id":"ref-1"}],"edges":[{"source":"paper-a","target":"ref-1"},{"source":"paper-b","target":"ref-1"}]}`
	if err := os.WriteFile(graphPath, []byte(graph), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "citations", "report", "--graph", graphPath, "--out", reportPath}, stdout, stderr); code != 0 {
		t.Fatalf("citations report exit code = %d, stderr = %s", code, stderr.String())
	}
	markdown, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	for _, want := range []string{"# Citation graph report", "- Nodes: 3", "- Edges: 2", "`ref-1`"} {
		if !strings.Contains(string(markdown), want) {
			t.Fatalf("report missing %q:\n%s", want, markdown)
		}
	}
	if !strings.Contains(stdout.String(), `"edgeCount":2`) {
		t.Fatalf("json output = %s", stdout.String())
	}
}

func TestExecuteSearchPubMedJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entrez/eutils/esearch.fcgi":
			_, _ = w.Write([]byte(`{"esearchresult":{"idlist":["123456"]}}`))
		case "/entrez/eutils/esummary.fcgi":
			_, _ = w.Write([]byte(`{"result":{"uids":["123456"],"123456":{"uid":"123456","title":"PubMed fixture","pubdate":"2026","articleids":[{"idtype":"doi","value":"10.1000/pubmed"}]}}}`))
		case "/entrez/eutils/efetch.fcgi":
			_, _ = w.Write([]byte(`<PubmedArticleSet><PubmedArticle><MedlineCitation><PMID>123456</PMID><MeshHeadingList><MeshHeading><DescriptorName>Machine Learning</DescriptorName></MeshHeading></MeshHeadingList></MedlineCitation></PubmedArticle></PubmedArticleSet>`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_PUBMED_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "pubmed", "--query", "machine learning", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	identifiers := envelope["data"].(map[string]any)["papers"].([]any)[0].(map[string]any)["Identifiers"].(map[string]any)
	if identifiers["PMID"] != "123456" {
		t.Fatalf("PMID = %#v", identifiers["PMID"])
	}
}

func TestExecuteSearchEuropePMCJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/webservices/rest/search" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "biomedical machine learning" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		_, _ = w.Write([]byte(`{"resultList":{"result":[{"id":"123456","pmid":"123456","doi":"10.1000/pmc","title":"Biomedical machine learning fixture","pubYear":"2026"}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_EUROPEPMC_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "europepmc", "--query", "biomedical machine learning", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	papers := envelope["data"].(map[string]any)["papers"].([]any)
	if len(papers) != 1 {
		t.Fatalf("len(papers) = %d, want 1", len(papers))
	}
	identifiers := papers[0].(map[string]any)["Identifiers"].(map[string]any)
	if identifiers["PMID"] != "123456" {
		t.Fatalf("PMID = %#v", identifiers["PMID"])
	}
}

func TestExecuteSearchSemanticScholarJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graph/v1/paper/search" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "crypto leakage detection" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if got := r.Header.Get("x-api-key"); got != "test-semantic-scholar-key" {
			t.Fatalf("x-api-key = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"paperId":"s2-1","title":"Leakage-aware financial machine learning","year":2026,"externalIds":{"DOI":"10.1000/s2"}}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY", "test-semantic-scholar-key")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "semantic-scholar", "--query", "crypto leakage detection", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	papers := data["papers"].([]any)
	if len(papers) != 1 {
		t.Fatalf("len(papers) = %d, want 1", len(papers))
	}
	paper := papers[0].(map[string]any)
	identifiers := paper["Identifiers"].(map[string]any)
	if identifiers["SemanticScholarID"] != "s2-1" {
		t.Fatalf("SemanticScholarID = %#v", identifiers["SemanticScholarID"])
	}
}

func TestExecuteSearchSemanticScholarRetriesRateLimit(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "quota", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"paperId":"s2-retry","title":"Retried Semantic Scholar search"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES", "1")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "semantic-scholar", "--query", "quota retry", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if requests != 2 || !strings.Contains(stdout.String(), "s2-retry") {
		t.Fatalf("requests=%d stdout=%s", requests, stdout.String())
	}
}

func TestExecuteSearchRelatedOpenAlex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/W1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"https://openalex.org/W1","related_works":["https://openalex.org/W2"]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "related", "--source", "openalex", "--paper", "W1", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"source":"openalex"`) || !strings.Contains(stdout.String(), `"SourceID":"W2"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteSearchImportOpenAlexResumeState(t *testing.T) {
	dir := t.TempDir()
	if code := Execute([]string{"project", "create", dir, "--title", "OpenAlex Resume"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	var cursors []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		switch r.URL.Query().Get("cursor") {
		case "*":
			_, _ = w.Write([]byte(`{"meta":{"next_cursor":"page-2"},"results":[{"id":"https://openalex.org/W1","title":"First","doi":"https://doi.org/10.1000/one"}]}`))
		case "page-2":
			_, _ = w.Write([]byte(`{"meta":{"next_cursor":"page-3"},"results":[{"id":"https://openalex.org/W2","title":"Second","doi":"https://doi.org/10.1000/two"}]}`))
		default:
			t.Fatalf("unexpected cursor: %q", r.URL.Query().Get("cursor"))
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	state := filepath.Join(dir, "openalex-state.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", dir, "search", "import", "--source", "openalex", "--query", "machine learning", "--pages", "1", "--limit", "1", "--resume-state", state}, stdout, stderr)
	if code != 0 {
		t.Fatalf("first import exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(state)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(data), `"nextCursor": "page-2"`) || !strings.Contains(string(data), `"query": "machine learning"`) {
		t.Fatalf("state missing cursor/query:\n%s", data)
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", dir, "search", "import", "--source", "openalex", "--query", "machine learning", "--pages", "1", "--limit", "1", "--resume-state", state}, stdout, stderr)
	if code != 0 {
		t.Fatalf("second import exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if got, want := strings.Join(cursors, ","), "*,page-2"; got != want {
		t.Fatalf("cursors = %q, want %q", got, want)
	}
}

func TestExecuteSearchImportOpenAlexPages(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "OpenAlex Import"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" || r.URL.Query().Get("search") != "machine learning" || r.URL.Query().Get("per-page") != "1" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		requests = append(requests, r.URL.Query().Get("cursor"))
		switch r.URL.Query().Get("cursor") {
		case "*":
			_, _ = w.Write([]byte(`{"meta":{"next_cursor":"page-2"},"results":[{"id":"https://openalex.org/W1","title":"First imported work","doi":"10.1000/oa1"}]}`))
		case "page-2":
			_, _ = w.Write([]byte(`{"meta":{"next_cursor":""},"results":[{"id":"https://openalex.org/W2","title":"Second imported work","doi":"10.1000/oa2"}]}`))
		default:
			t.Fatalf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", dir, "search", "import", "--source", "openalex", "--query", "machine learning", "--pages", "2", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(requests) != 2 || requests[0] != "*" || requests[1] != "page-2" {
		t.Fatalf("requests = %#v", requests)
	}
	if !strings.Contains(stdout.String(), `"imported":2`) || !strings.Contains(stdout.String(), "page-2") {
		t.Fatalf("stdout = %s", stdout.String())
	}
	listed := mustRunCLI(t, "--json", "--project", dir, "library", "list")
	if !strings.Contains(string(listed), "First imported work") || !strings.Contains(string(listed), "Second imported work") {
		t.Fatalf("library = %s", listed)
	}
}

func TestExecuteSearchOpenAlexAuthorsJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authors" || r.URL.Query().Get("search") != "Ada Lovelace" || r.URL.Query().Get("per-page") != "1" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/A123","display_name":"Ada Lovelace","works_count":42}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "--source", "openalex", "--entity", "authors", "--query", "Ada Lovelace", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"entity":"authors"`) || !strings.Contains(stdout.String(), `"sourceId":"A123"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteSearchOpenAlexInstitutionsJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/institutions" || r.URL.Query().Get("search") != "University" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/I123","display_name":"Example University","works_count":100,"ror":"https://ror.org/123","country_code":"GB"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "--source", "openalex", "--entity", "institutions", "--query", "University", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"entity":"institutions"`) || !strings.Contains(stdout.String(), `"ror":"https://ror.org/123"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteSearchCrossrefJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "artificial photosynthesis" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		_, _ = w.Write([]byte(`{"message":{"items":[{"DOI":"10.5555/crossref.example","title":["Artificial photosynthesis Crossref fixture"],"issued":{"date-parts":[[2026]]}}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_CROSSREF_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "crossref", "--query", "artificial photosynthesis", "--limit", "1"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	papers := data["papers"].([]any)
	if len(papers) != 1 {
		t.Fatalf("len(papers) = %d, want 1", len(papers))
	}
	paper := papers[0].(map[string]any)
	if paper["Title"] != "Artificial photosynthesis Crossref fixture" {
		t.Fatalf("paper title = %#v", paper["Title"])
	}
}

func TestExecuteOALookupUnpaywallJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/v2/10.5555%2Fexample" {
			t.Fatalf("path = %q", r.URL.EscapedPath())
		}
		if r.URL.Query().Get("email") != "researcher@example.org" {
			t.Fatalf("email query = %q", r.URL.Query().Get("email"))
		}
		_, _ = w.Write([]byte(`{"doi":"10.5555/example","is_oa":true,"oa_status":"green","best_oa_location":{"url":"https://example.org/article","url_for_pdf":"https://example.org/article.pdf","license":"cc-by","host_type":"repository"}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_UNPAYWALL_URL", server.URL)
	t.Setenv("RFORGE_UNPAYWALL_EMAIL", "researcher@example.org")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "oa", "lookup", "10.5555/example"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	record := data["open_access"].(map[string]any)
	if record["DOI"] != "10.5555/example" || record["OpenAccess"] != true || record["PDFURL"] != "https://example.org/article.pdf" {
		t.Fatalf("open_access = %#v", record)
	}
	if strings.Contains(stdout.String(), "researcher@example.org") || strings.Contains(stderr.String(), "researcher@example.org") {
		t.Fatalf("output leaked Unpaywall email: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestExecuteDuplicateReportJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Artificial photosynthesis catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/left"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Artificial photosynthesis catalysts: a review", Identifiers: library.Identifiers{DOI: "10.1000/right"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026})
	if err := store.Create(left); err != nil {
		t.Fatalf("Create left returned error: %v", err)
	}
	if err := store.Create(right); err != nil {
		t.Fatalf("Create right returned error: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", dir, "duplicate", "report"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	matches := data["matches"].([]any)
	if len(matches) != 1 {
		t.Fatalf("len(matches) = %d, want 1", len(matches))
	}
	match := matches[0].(map[string]any)
	if match["Reason"] != "fuzzy_title_author_year" || match["Score"].(float64) <= 0 {
		t.Fatalf("match = %#v", match)
	}
}

func TestExecuteDuplicateReportFiltersGraphImportedSource(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Graph Dedupe"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	existing, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Graph neural networks for reviews", Identifiers: library.Identifiers{DOI: "10.1000/existing"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026, SourceRefs: []library.SourceRef{{Source: "openalex"}}})
	imported, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Graph neural networks in reviews", Identifiers: library.Identifiers{SemanticScholarID: "S2-1"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026, SourceRefs: []library.SourceRef{{Source: "semantic-scholar", RawPayloadRef: "semantic-scholar:/recursive?seed=S2"}}})
	unrelated, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Unrelated graph neural networks in reviews", Identifiers: library.Identifiers{DOI: "10.1000/unrelated"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026, SourceRefs: []library.SourceRef{{Source: "crossref"}}})
	for _, paper := range []library.PaperRecord{existing, imported, unrelated} {
		if err := store.Create(paper); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "duplicate", "report", "--source", "semantic-scholar"}, stdout, stderr); code != 0 {
		t.Fatalf("duplicate report exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"RightSources":["semantic-scholar"]`) {
		t.Fatalf("filtered duplicate report = %s", stdout.String())
	}
}

func TestExecuteDuplicateMergeAndSplitRecordProvenance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "A artificial photosynthesis catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/left"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026, SourceRefs: []library.SourceRef{{Source: "openalex"}}})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "B artificial photosynthesis catalysts: a review", Identifiers: library.Identifiers{ArXivID: "2401.00001"}, Authors: []library.Author{{Family: "Lovelace"}}, Year: 2026, SourceRefs: []library.SourceRef{{Source: "arxiv"}}})
	if err := store.Create(left); err != nil {
		t.Fatalf("Create left returned error: %v", err)
	}
	if err := store.Create(right); err != nil {
		t.Fatalf("Create right returned error: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "duplicate", "merge", "0", "1"}, stdout, stderr); code != 0 {
		t.Fatalf("merge exit code = %d, stderr = %s", code, stderr.String())
	}
	mergedItems, err := store.List()
	if err != nil {
		t.Fatalf("List after merge returned error: %v", err)
	}
	if len(mergedItems) != 1 || mergedItems[0].Identifiers.ArXivID != "2401.00001" || len(mergedItems[0].SourceRefs) != 2 {
		t.Fatalf("merged items = %#v", mergedItems)
	}

	splitPath := filepath.Join(t.TempDir(), "split.json")
	if err := library.ExportJSON(splitPath, []library.PaperRecord{left, right}); err != nil {
		t.Fatalf("write split JSON: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "duplicate", "split", "0", splitPath}, stdout, stderr); code != 0 {
		t.Fatalf("split exit code = %d, stderr = %s", code, stderr.String())
	}
	splitItems, err := store.List()
	if err != nil {
		t.Fatalf("List after split returned error: %v", err)
	}
	if len(splitItems) != 2 {
		t.Fatalf("split items = %#v", splitItems)
	}
	events, err := provenance.Read(dir)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	if !sawAction(events, "duplicate.merge") || !sawAction(events, "duplicate.split") {
		t.Fatalf("missing merge/split provenance: %#v", events)
	}
}

func sawAction(events []provenance.Event, action string) bool {
	for _, event := range events {
		if event.Action == action {
			return true
		}
	}
	return false
}

func TestExecuteOSSCloneWithLocalFakeRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	remote := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGitCommand(t, remote, "init")
	runGitCommand(t, remote, "config", "user.email", "test@example.org")
	runGitCommand(t, remote, "config", "user.name", "ResearchForge Test")
	if err := os.WriteFile(filepath.Join(remote, "README.md"), []byte("# fake repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCommand(t, remote, "add", "README.md")
	runGitCommand(t, remote, "commit", "-m", "initial")
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "oss", "clone", "owner/repo", "--url", remote}, stdout, stderr); code != 0 {
		t.Fatalf("oss clone exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "opensource", "clones", "owner", "repo", "README.md")); err != nil {
		t.Fatalf("missing cloned README: %v", err)
	}
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func TestExecuteReportBuildAndAudit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	out := filepath.Join(t.TempDir(), "report.md")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "report", "build", "--out", out}, stdout, stderr); code != 0 {
		t.Fatalf("report build exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(data), "# Demo Review") || !strings.Contains(string(data), "Audit appendix") {
		t.Fatalf("report = %s", data)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "report", "audit"}, stdout, stderr); code != 0 {
		t.Fatalf("report audit exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "issues") {
		t.Fatalf("audit output = %s", stdout.String())
	}
}

func TestExecuteAnalysisPrepareRunAndExport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	item := `[{"PaperID":"paper-1","Values":{"mean_treatment":"10","mean_control":"8","sd_pooled":"2","n_treatment":"25","n_control":"25"},"Support":{"Kind":"passage","Ref":"p1"},"Status":"accepted"}]`
	if err := os.WriteFile(filepath.Join(dir, "data", "evidence.items.json"), []byte(item), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "prepare", "run-1"}, stdout, stderr); code != 0 {
		t.Fatalf("analysis prepare exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "analysis", "run", "run-1"}, stdout, stderr); code != 0 {
		t.Fatalf("analysis run exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	exportPath := filepath.Join(t.TempDir(), "analysis.json")
	if code := Execute([]string{"--json", "--project", dir, "analysis", "export", "run-1", exportPath}, stdout, stderr); code != 0 {
		t.Fatalf("analysis export exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !strings.Contains(string(data), "run-1") || !strings.Contains(string(data), "InputRows") {
		t.Fatalf("export = %s", data)
	}
}

func TestExecuteAnalysisSubgroup(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Subgroup"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "sub-run", InputRows: []analysis.InputRow{
		{PaperID: "p1", EffectSize: 1, Variance: 1},
		{PaperID: "p2", EffectSize: 3, Variance: 1},
		{PaperID: "p3", EffectSize: 10, Variance: 2},
	}}
	if err := writeJSONFile(filepath.Join(dir, "analysis", "sub-run.json"), run); err != nil {
		t.Fatalf("write run: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "subgroup", "sub-run", "--variable", "region", "--group", "p1=EU", "--group", "p2=EU", "--group", "p3=US"}, stdout, stderr); code != 0 {
		t.Fatalf("subgroup exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"variable":"region"`) || !strings.Contains(stdout.String(), `"group":"EU"`) {
		t.Fatalf("subgroup output = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis", "sub-run-subgroup.json")); err != nil {
		t.Fatalf("subgroup artifact missing: %v", err)
	}
}

func TestExecuteAnalysisMetaRegression(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Meta Regression"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "reg-run", InputRows: []analysis.InputRow{
		{PaperID: "p1", EffectSize: 1, Variance: 1},
		{PaperID: "p2", EffectSize: 2, Variance: 1},
		{PaperID: "p3", EffectSize: 3, Variance: 1},
	}}
	if err := writeJSONFile(filepath.Join(dir, "analysis", "reg-run.json"), run); err != nil {
		t.Fatalf("write run: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "meta-regression", "reg-run", "--moderator", "dose", "--value", "p1=1", "--value", "p2=2", "--value", "p3=3"}, stdout, stderr); code != 0 {
		t.Fatalf("meta-regression exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"moderator":"dose"`) || !strings.Contains(stdout.String(), `"slope":1`) {
		t.Fatalf("meta-regression output = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis", "reg-run-meta-regression.json")); err != nil {
		t.Fatalf("meta-regression artifact missing: %v", err)
	}
}

func TestExecuteAnalysisBayesianNormalApproximation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Bayesian Meta-analysis"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "bayes-run", InputRows: []analysis.InputRow{
		{PaperID: "p1", EffectSize: 1, Variance: 1},
		{PaperID: "p2", EffectSize: 3, Variance: 1},
	}}
	if err := writeJSONFile(filepath.Join(dir, "analysis", "bayes-run.json"), run); err != nil {
		t.Fatalf("write run: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "bayesian", "bayes-run", "--method", "normal-approx", "--prior-mean", "0", "--prior-variance", "1"}, stdout, stderr); code != 0 {
		t.Fatalf("bayesian exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"method":"normal-approx"`) || !strings.Contains(stdout.String(), `"runId":"bayes-run"`) {
		t.Fatalf("bayesian output = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis", "bayes-run-bayesian.json")); err != nil {
		t.Fatalf("bayesian artifact missing: %v", err)
	}
}

func TestExecuteAnalysisPublicationBiasEgger(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Bias Meta-analysis"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "bias-run", InputRows: []analysis.InputRow{
		{PaperID: "p1", EffectSize: 0.2, Variance: 0.04},
		{PaperID: "p2", EffectSize: 0.3, Variance: 0.05},
		{PaperID: "p3", EffectSize: 0.4, Variance: 0.06},
	}}
	if err := writeJSONFile(filepath.Join(dir, "analysis", "bias-run.json"), run); err != nil {
		t.Fatalf("write run: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "publication-bias", "bias-run", "--method", "egger"}, stdout, stderr); code != 0 {
		t.Fatalf("publication bias exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"method":"egger"`) || !strings.Contains(stdout.String(), "underpowered") {
		t.Fatalf("publication bias output = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis", "bias-run-publication-bias.json")); err != nil {
		t.Fatalf("publication bias artifact missing: %v", err)
	}
}

func TestExecuteAnalysisPrepareSupportsLogOddsRatio(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Binary Meta-analysis"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	items := `[{"PaperID":"paper-1","Values":{"events_treatment":"30","n_treatment":"100","events_control":"20","n_control":"100"},"Support":{"Kind":"passage","Ref":"p1"},"Status":"accepted"}]`
	if err := os.WriteFile(filepath.Join(dir, "data", "evidence.items.json"), []byte(items), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "prepare", "binary-run", "--effect", "log-odds-ratio"}, stdout, stderr); code != 0 {
		t.Fatalf("analysis prepare exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope struct {
		Data struct {
			Run analysis.AnalysisRun `json:"run"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("prepare stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if len(envelope.Data.Run.InputRows) != 1 || envelope.Data.Run.InputRows[0].PaperID != "paper-1" || envelope.Data.Run.InputRows[0].Variance <= 0 {
		t.Fatalf("prepare output = %#v", envelope)
	}
}

func TestExecuteAnalysisPrepareSupportsRiskRatio(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Risk Ratio Meta-analysis"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	items := `[{"PaperID":"paper-1","Values":{"events_treatment":"40","n_treatment":"100","events_control":"20","n_control":"100"},"Support":{"Kind":"passage","Ref":"p1"},"Status":"accepted"}]`
	if err := os.WriteFile(filepath.Join(dir, "data", "evidence.items.json"), []byte(items), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "prepare", "rr-run", "--effect", "risk-ratio"}, stdout, stderr); code != 0 {
		t.Fatalf("analysis prepare exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ID":"rr-run"`) || !strings.Contains(stdout.String(), `"PaperID":"paper-1"`) {
		t.Fatalf("prepare output = %s", stdout.String())
	}
}

func TestExecuteAnalysisSensitivityLeaveOneOut(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	items := `[
{"PaperID":"paper-1","Values":{"mean_treatment":"10","mean_control":"8","sd_pooled":"2","n_treatment":"25","n_control":"25"},"Support":{"Kind":"passage","Ref":"p1"},"Status":"accepted"},
{"PaperID":"paper-2","Values":{"mean_treatment":"12","mean_control":"8","sd_pooled":"2","n_treatment":"25","n_control":"25"},"Support":{"Kind":"passage","Ref":"p2"},"Status":"accepted"},
{"PaperID":"paper-3","Values":{"mean_treatment":"16","mean_control":"8","sd_pooled":"2","n_treatment":"25","n_control":"25"},"Support":{"Kind":"passage","Ref":"p3"},"Status":"accepted"}
]`
	if err := os.WriteFile(filepath.Join(dir, "data", "evidence.items.json"), []byte(items), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "analysis", "prepare", "run-1"}, stdout, stderr); code != 0 {
		t.Fatalf("analysis prepare exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "analysis", "sensitivity", "run-1", "--method", "leave-one-out"}, stdout, stderr); code != 0 {
		t.Fatalf("analysis sensitivity exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "leave-one-out") || !strings.Contains(stdout.String(), "paper-1") {
		t.Fatalf("sensitivity output = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis", "run-1-sensitivity.json")); err != nil {
		t.Fatalf("sensitivity artifact missing: %v", err)
	}
}

func TestExecuteEvidenceSchemaExtractAuditAndSuggest(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "extraction", "schema", "add", "catalysts", "--field", "catalyst:string"}, stdout, stderr); code != 0 {
		t.Fatalf("schema add exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "extract", "add", "--paper", "paper-1", "--schema", "catalysts", "--value", "catalyst=TiO2", "--support", "passage:p1", "--status", "accepted"}, stdout, stderr); code != 0 {
		t.Fatalf("extract add exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "evidence", "audit"}, stdout, stderr); code != 0 {
		t.Fatalf("evidence audit exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"issues":[]`) {
		t.Fatalf("audit output = %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "extract", "suggest", "--paper", "paper-2"}, stdout, stderr); code != 0 {
		t.Fatalf("extract suggest exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "noop-llm") {
		t.Fatalf("suggest output = %s", stdout.String())
	}
}

func TestExecuteScreenPrioritizeRanksLibraryFromScreeningFeedback(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"LightGBM leakage detection for crypto order books","Abstract":"microstructure forecasting","Identifiers":{"DOI":"10.1000/include"}},
  {"Title":"Plant photosynthesis catalyst review","Abstract":"materials chemistry","Identifiers":{"DOI":"10.1000/exclude"}},
  {"Title":"Crypto order book leakage detection","Abstract":"LightGBM microstructure forecasting","Identifiers":{"DOI":"10.1000/relevant"}},
  {"Title":"Artificial photosynthesis catalyst","Abstract":"materials review","Identifiers":{"DOI":"10.1000/irrelevant"}}
]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write library fixture: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, stdout, stderr); code != 0 {
		t.Fatalf("import exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := Execute([]string{"--json", "--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/include", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("include decision exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/exclude", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("exclude decision exit code = %d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "screen", "prioritize", "--stage", "title_abstract", "--limit", "2"}, stdout, stderr); code != 0 {
		t.Fatalf("screen prioritize exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope struct {
		Data struct {
			Prioritized []struct {
				ID    string  `json:"id"`
				Score float64 `json:"score"`
			} `json:"prioritized"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if len(envelope.Data.Prioritized) != 2 {
		t.Fatalf("prioritized length = %d, want 2: %#v", len(envelope.Data.Prioritized), envelope.Data.Prioritized)
	}
	if envelope.Data.Prioritized[0].ID != "10.1000/relevant" {
		t.Fatalf("top priority = %#v, want relevant paper", envelope.Data.Prioritized[0])
	}
}

func TestExecuteScreenModelPrioritizeRanksWithModelScore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Model Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"crypto leakage microstructure","Identifiers":{"DOI":"10.1000/include"}},
  {"Title":"plant catalyst photosynthesis","Identifiers":{"DOI":"10.1000/exclude"}},
  {"Title":"crypto microstructure signals","Identifiers":{"DOI":"10.1000/relevant"}},
  {"Title":"plant photosynthesis materials","Identifiers":{"DOI":"10.1000/irrelevant"}}
]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write library fixture: %v", err)
	}
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("import exit code = %d", code)
	}
	if code := Execute([]string{"--json", "--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/include", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("include decision exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/exclude", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("exclude decision exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "model-prioritize", "--stage", "title_abstract", "--limit", "1"}, stdout, stderr); code != 0 {
		t.Fatalf("screen model-prioritize exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"method":"naive-bayes"`) || !strings.Contains(stdout.String(), `"id":"10.1000/relevant"`) {
		t.Fatalf("model-prioritize output = %s", stdout.String())
	}
}

func TestExecuteScreenUncertaintyRanksBoundaryRecords(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"crypto leakage","Identifiers":{"DOI":"10.1000/include"}},
  {"Title":"plant catalyst","Identifiers":{"DOI":"10.1000/exclude"}},
  {"Title":"crypto leakage","Identifiers":{"DOI":"10.1000/positive"}},
  {"Title":"unseen validation","Identifiers":{"DOI":"10.1000/boundary"}}
]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write library fixture: %v", err)
	}
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("import exit code = %d", code)
	}
	if code := Execute([]string{"--json", "--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/include", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("include decision exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/exclude", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("exclude decision exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "uncertainty", "--stage", "title_abstract", "--limit", "1"}, stdout, stderr); code != 0 {
		t.Fatalf("screen uncertainty exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id":"10.1000/boundary"`) || !strings.Contains(stdout.String(), `"uncertainty":1`) {
		t.Fatalf("uncertainty output = %s", stdout.String())
	}
}

func TestExecuteScreenWorkflowAndPrismaCounts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "configure", "--reason", "wrong population"}, stdout, stderr); code != 0 {
		t.Fatalf("screen configure exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "exclude", "--reason", "wrong population", "--reviewer", "ada"}, stdout, stderr); code != 0 {
		t.Fatalf("screen decide exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "screen", "queue", "--stage", "title_abstract", "--decision", "exclude"}, stdout, stderr); code != 0 {
		t.Fatalf("screen queue exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "paper-1") {
		t.Fatalf("queue output = %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "prisma", "counts"}, stdout, stderr); code != 0 {
		t.Fatalf("prisma counts exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"Excluded":1`) {
		t.Fatalf("counts output = %s", stdout.String())
	}
}

func TestExecuteScreenStoppingReportsRecommendation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Stopping Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "p1", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "stopping", "--stage", "title_abstract", "--target-recall", "0.9"}, stdout, stderr); code != 0 {
		t.Fatalf("screen stopping exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"canStop":true`) || !strings.Contains(stdout.String(), `"targetRecall":0.9`) {
		t.Fatalf("stopping output = %s", stdout.String())
	}
}

func TestExecuteScreenRecallReportsEffortCurve(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Recall Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	decisions := [][]string{
		{"p1", "exclude", "off-topic", "ada"},
		{"p2", "include", "", "ada"},
		{"p3", "include", "", "grace"},
	}
	for _, decision := range decisions {
		args := []string{"--project", dir, "screen", "decide", "--paper", decision[0], "--stage", "title_abstract", "--decision", decision[1], "--reviewer", decision[3]}
		if decision[2] != "" {
			args = append(args, "--reason", decision[2])
		}
		if code := Execute(args, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
			t.Fatalf("screen decide %v exit code = %d", decision, code)
		}
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "recall", "--stage", "title_abstract"}, stdout, stderr); code != 0 {
		t.Fatalf("screen recall exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"recall":1`) || !strings.Contains(stdout.String(), `"screened":3`) {
		t.Fatalf("recall output = %s", stdout.String())
	}
}

func TestExecuteScreenProgressReportsReviewerMetrics(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"Paper one","Identifiers":{"DOI":"10.1000/one"}},
  {"Title":"Paper two","Identifiers":{"DOI":"10.1000/two"}},
  {"Title":"Paper three","Identifiers":{"DOI":"10.1000/three"}}
]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write library fixture: %v", err)
	}
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("import exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/one", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide include exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/two", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "grace"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide exclude exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "10.1000/two", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide conflict exit code = %d", code)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "progress", "--stage", "title_abstract"}, stdout, stderr); code != 0 {
		t.Fatalf("screen progress exit code = %d, stderr = %s", code, stderr.String())
	}
	var env struct {
		Data struct {
			Progress struct {
				TotalRecords    int `json:"totalRecords"`
				ScreenedRecords int `json:"screenedRecords"`
				Remaining       int `json:"remaining"`
				Conflicts       int `json:"conflicts"`
				Reviewers       []struct {
					Reviewer  string `json:"reviewer"`
					Decisions int    `json:"decisions"`
				} `json:"reviewers"`
			} `json:"progress"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("progress stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if env.Data.Progress.TotalRecords != 3 || env.Data.Progress.ScreenedRecords != 2 || env.Data.Progress.Remaining != 1 || env.Data.Progress.Conflicts != 1 {
		t.Fatalf("progress = %#v", env.Data.Progress)
	}
	if len(env.Data.Progress.Reviewers) != 2 || env.Data.Progress.Reviewers[0].Reviewer != "ada" || env.Data.Progress.Reviewers[0].Decisions != 2 {
		t.Fatalf("reviewers = %#v", env.Data.Progress.Reviewers)
	}
}

func TestExecuteScreenConflictsReportsConflictingDecisions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	// Two reviewers disagree on the same paper at the same stage.
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide include exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "linus"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide exclude exit code = %d", code)
	}
	// A non-conflicting paper must not appear.
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "paper-2", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide paper-2 exit code = %d", code)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "conflicts", "--stage", "title_abstract"}, stdout, stderr); code != 0 {
		t.Fatalf("screen conflicts exit code = %d, stderr = %s", code, stderr.String())
	}
	var env struct {
		Data struct {
			Conflicts []string `json:"conflicts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("conflicts stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if len(env.Data.Conflicts) != 1 || env.Data.Conflicts[0] != "paper-1" {
		t.Fatalf("conflicts = %v, want [paper-1]", env.Data.Conflicts)
	}
}

func TestExecuteScreenAdjudicateResolvesConflict(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide include exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "linus"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen decide exclude exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "screen", "adjudicate", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "include", "--reviewer", "carol"}, stdout, stderr); code != 0 {
		t.Fatalf("screen adjudicate exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "screen", "conflicts", "--stage", "title_abstract"}, stdout, stderr); code != 0 {
		t.Fatalf("screen conflicts exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "paper-1") {
		t.Fatalf("conflict not resolved: %s", stdout.String())
	}
	var events []screening.DecisionEvent
	if err := readJSONFile(screenEventsPath(dir), &events); err != nil {
		t.Fatalf("load events: %v", err)
	}
	if !events[len(events)-1].Adjudicated {
		t.Fatalf("last event not adjudicated: %#v", events[len(events)-1])
	}
}

func TestExecuteScreenConflictsRequiresStage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--project", dir, "screen", "conflicts"}, new(bytes.Buffer), stderr); code == 0 {
		t.Fatalf("expected non-zero exit for missing --stage")
	}
}

func TestExecuteIndexRebuildAndRetrieve(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	parsed := `{"PaperID":"paper-1","Sections":[{"ID":"paper-1-sec-1","Title":"Intro","Passages":[{"ID":"paper-1-sec-1-p-1","PaperID":"paper-1","SectionID":"paper-1-sec-1","Text":"Solar fuel catalysts split water."}]}]}`
	if err := os.WriteFile(filepath.Join(parsedDir, "paper-1.json"), []byte(parsed), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "index", "rebuild"}, stdout, stderr); code != 0 {
		t.Fatalf("index rebuild exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "retrieve", "--query", "solar catalysts"}, stdout, stderr); code != 0 {
		t.Fatalf("retrieve exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "paper-1-sec-1-p-1") || !strings.Contains(stdout.String(), "Solar fuel catalysts") {
		t.Fatalf("retrieve output = %s", stdout.String())
	}
}

func TestExecuteParseReviewRefsWritesAmbiguousQueue(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Reference Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsed := filepath.Join(t.TempDir(), "parsed.json")
	out := filepath.Join(t.TempDir(), "review.json")
	doc := `{"PaperID":"paper-1","References":[{"Title":"Confident","DOI":"10.1000/ok","Confidence":0.95},{"Title":"Ambiguous","Raw":"raw ref","Confidence":0.4}]}`
	if err := os.WriteFile(parsed, []byte(doc), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "review-refs", "--parsed", parsed, "--out", out, "--threshold", "0.75"}, stdout, stderr); code != 0 {
		t.Fatalf("review refs exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read review output: %v", err)
	}
	if !strings.Contains(string(data), `"reason": "low_confidence"`) || !strings.Contains(string(data), `"raw": "raw ref"`) {
		t.Fatalf("review report = %s", data)
	}
}

func TestExecuteParseReferencesWithAnyStyleCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Anystyle"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	refs := filepath.Join(t.TempDir(), "refs.txt")
	out := filepath.Join(t.TempDir(), "refs.json")
	if err := os.WriteFile(refs, []byte("Reference text"), 0o644); err != nil {
		t.Fatalf("write refs: %v", err)
	}
	script := filepath.Join(t.TempDir(), "anystyle-fixture")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s' '[{\"title\":\"Parsed reference\",\"doi\":\"10.1000/ref\"}]'\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	t.Setenv("RFORGE_ANYSTYLE_CMD", script)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "references", "--paper", "paper-1", "--parser", "anystyle", "--file", refs, "--out", out}, stdout, stderr); code != 0 {
		t.Fatalf("parse references exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read refs output: %v", err)
	}
	if !strings.Contains(string(data), `"ParserName": "anystyle"`) || !strings.Contains(string(data), `"DOI": "10.1000/ref"`) {
		t.Fatalf("parsed references = %s", data)
	}
}

func TestExecuteParseNormalizeRefsWritesSourceMatches(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Reference Normalize"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsed := filepath.Join(t.TempDir(), "parsed.json")
	out := filepath.Join(t.TempDir(), "refs.json")
	parsedDoc := `{"PaperID":"paper-1","References":[{"Title":"Reference work","DOI":"10.5555/ref"}]}`
	if err := os.WriteFile(parsed, []byte(parsedDoc), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" || r.URL.Query().Get("query") != "10.5555/ref" {
			t.Fatalf("unexpected crossref request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"message":{"items":[{"DOI":"10.5555/ref","title":["Normalized Reference"],"reference-count":1}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_CROSSREF_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "normalize-refs", "--parsed", parsed, "--source", "crossref", "--out", out}, stdout, stderr); code != 0 {
		t.Fatalf("normalize refs exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read normalization: %v", err)
	}
	if !strings.Contains(string(data), `"matchedDoi": "10.5555/ref"`) || !strings.Contains(string(data), "Normalized Reference") {
		t.Fatalf("normalization report = %s", data)
	}
}

func TestExecutePDFFetchArXivAsset(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "arXiv Fetch"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pdf/2401.00001" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("%PDF arxiv fixture"))
	}))
	defer server.Close()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "pdf", "fetch-arxiv", "--paper", "2401.00001", "--kind", "pdf", "--url", server.URL + "/pdf/2401.00001"}, stdout, stderr); code != 0 {
		t.Fatalf("fetch arxiv exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"AcquisitionSource":"arxiv-pdf"`) || !strings.Contains(stdout.String(), `"MIMEType":"application/pdf"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteParsePaperMageWritesParsedDocumentAndManifest(t *testing.T) {
	dir := t.TempDir()
	if code := Execute([]string{"project", "create", dir, "--title", "PaperMage Demo"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	input := filepath.Join(dir, "papermage.json")
	if err := os.WriteFile(input, []byte(`{"metadata":{"title":"Layered paper"},"layers":{"sections":[{"text":"Intro"}],"paragraphs":[{"section":"Intro","text":"Paragraph text."}],"bibliography":[{"title":"Ref","doi":"10.1000/ref"}]}}`), 0o644); err != nil {
		t.Fatalf("write PaperMage input: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", dir, "parse", "--paper", "paper-1", "--parser", "papermage", "--papermage", input}, stdout, stderr)
	if code != 0 {
		t.Fatalf("parse papermage exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	parsed, err := os.ReadFile(filepath.Join(dir, "parsed", "paper-1.json"))
	if err != nil {
		t.Fatalf("read parsed: %v", err)
	}
	if !strings.Contains(string(parsed), `"ParserName": "papermage"`) || !strings.Contains(string(parsed), "Layered paper") {
		t.Fatalf("parsed document = %s", parsed)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "parsed", "paper-1.manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(manifest), `"parserName": "papermage"`) || !strings.Contains(string(manifest), `"passages": 1`) {
		t.Fatalf("manifest = %s", manifest)
	}
}

func TestExecuteParseS2ORCWritesParsedDocument(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "S2ORC Parse"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	s2orcPath := filepath.Join(t.TempDir(), "paper.json")
	fixture := `{"title":"S2ORC Fixture","abstract":"Abstract text.","body_text":[{"section":"Intro","text":"Body text."}],"bib_entries":{"B0":{"title":"Reference","doi":"10.1000/ref"}}}`
	if err := os.WriteFile(s2orcPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write s2orc: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "--paper", "paper-1", "--parser", "s2orc", "--s2orc", s2orcPath}, stdout, stderr); code != 0 {
		t.Fatalf("parse s2orc exit code = %d, stderr = %s", code, stderr.String())
	}
	parsedPath := filepath.Join(dir, "parsed", "paper-1.json")
	data, err := os.ReadFile(parsedPath)
	if err != nil {
		t.Fatalf("read parsed: %v", err)
	}
	if !strings.Contains(string(data), `"ParserName": "s2orc-doc2json"`) || !strings.Contains(string(data), "S2ORC Fixture") || !strings.Contains(string(data), "Body text") {
		t.Fatalf("parsed doc = %s", data)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "parsed", "paper-1.manifest.json"))
	if err != nil {
		t.Fatalf("read parser manifest: %v", err)
	}
	if !strings.Contains(string(manifest), `"parserName": "s2orc-doc2json"`) || !strings.Contains(string(manifest), `"passages": 1`) || !strings.Contains(string(manifest), `"references": 1`) {
		t.Fatalf("parser manifest = %s", manifest)
	}
}

func TestExecuteParseTeXWritesParsedDocument(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "TeX Parse"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	texPath := filepath.Join(t.TempDir(), "paper.tex")
	if err := os.WriteFile(texPath, []byte(`\title{TeX Fixture}\begin{abstract}Abstract text.\end{abstract}\section{Intro}Body text.`), 0o644); err != nil {
		t.Fatalf("write tex: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "--paper", "paper-1", "--parser", "tex", "--tex", texPath}, stdout, stderr); code != 0 {
		t.Fatalf("parse tex exit code = %d, stderr = %s", code, stderr.String())
	}
	parsedPath := filepath.Join(dir, "parsed", "paper-1.json")
	data, err := os.ReadFile(parsedPath)
	if err != nil {
		t.Fatalf("read parsed: %v", err)
	}
	if !strings.Contains(string(data), `"ParserName": "tex"`) || !strings.Contains(string(data), "TeX Fixture") || !strings.Contains(string(data), "Body text") {
		t.Fatalf("parsed doc = %s", data)
	}
}

func TestExecuteParseNormalizeRefsSupportsOpenAlex(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "OpenAlex Ref Normalize"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsed := filepath.Join(t.TempDir(), "parsed.json")
	out := filepath.Join(t.TempDir(), "refs.json")
	parsedDoc := `{"PaperID":"paper-1","References":[{"Title":"Reference work","DOI":"10.5555/ref"}]}`
	if err := os.WriteFile(parsed, []byte(parsedDoc), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" || r.URL.Query().Get("search") != "10.5555/ref" {
			t.Fatalf("unexpected openalex request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W1","doi":"https://doi.org/10.5555/ref","title":"Normalized OpenAlex Reference"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "normalize-refs", "--parsed", parsed, "--source", "openalex", "--out", out}, stdout, stderr); code != 0 {
		t.Fatalf("normalize refs exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read normalization: %v", err)
	}
	if !strings.Contains(string(data), `"source": "openalex"`) || !strings.Contains(string(data), "Normalized OpenAlex Reference") {
		t.Fatalf("normalization report = %s", data)
	}
}

func TestExecuteParseCompareWritesComparisonReport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Parser Compare"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	left := filepath.Join(t.TempDir(), "left.json")
	right := filepath.Join(t.TempDir(), "right.json")
	out := filepath.Join(t.TempDir(), "compare.json")
	leftDoc := `{"PaperID":"paper-1","ParserName":"grobid","Title":"Shared","Sections":[{"ID":"s1","Passages":[{"ID":"p1"}]}],"References":[{"Title":"Ref"}]}`
	rightDoc := `{"PaperID":"paper-1","ParserName":"s2orc","Title":"Different","Sections":[{"ID":"s1","Passages":[{"ID":"p1"},{"ID":"p2"}]}],"References":[{"Title":"Ref"},{"Title":"Ref 2"}],"Warnings":["low confidence"]}`
	if err := os.WriteFile(left, []byte(leftDoc), 0o644); err != nil {
		t.Fatalf("write left: %v", err)
	}
	if err := os.WriteFile(right, []byte(rightDoc), 0o644); err != nil {
		t.Fatalf("write right: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "parse", "compare", "--left", left, "--right", right, "--out", out}, stdout, stderr); code != 0 {
		t.Fatalf("parse compare exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read comparison: %v", err)
	}
	if !strings.Contains(string(data), "review-required") || !strings.Contains(string(data), "referenceDelta") {
		t.Fatalf("comparison report = %s", data)
	}
}

func TestExecuteIndexAndRetrieveWithHybridBackend(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Hybrid Demo"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	parsed := `{"PaperID":"paper-1","Sections":[{"ID":"s1","Passages":[{"ID":"p1","PaperID":"paper-1","SectionID":"s1","Text":"Solar fuel catalysts split water."}]}]}`
	if err := os.WriteFile(filepath.Join(parsedDir, "paper-1.json"), []byte(parsed), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/researchforge_passages":
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/collections/researchforge_passages/points":
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		case "/collections/researchforge_passages/points/search":
			_, _ = w.Write([]byte(`{"result":[{"payload":{"PaperID":"paper-2","SectionID":"s2","PassageID":"p2","Text":"Vector-only solar catalyst passage."}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "index", "rebuild", "--backend", "hybrid"}, stdout, stderr); code != 0 {
		t.Fatalf("hybrid index exit code = %d, stderr = %s", code, stderr.String())
	}
	lockData, err := os.ReadFile(filepath.Join(dir, "data", "retrieval.lock.json"))
	if err != nil {
		t.Fatalf("read retrieval lock: %v", err)
	}
	if !strings.Contains(string(lockData), `"backend": "hybrid"`) || !strings.Contains(string(lockData), `"embeddingBackend": "deterministic-hash"`) {
		t.Fatalf("retrieval lock = %s", lockData)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "retrieve", "--query", "solar catalysts", "--backend", "hybrid"}, stdout, stderr); code != 0 {
		t.Fatalf("hybrid retrieve exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Solar fuel catalysts") || !strings.Contains(stdout.String(), "Vector-only solar catalyst") || !strings.Contains(stdout.String(), `"backend":"hybrid"`) {
		t.Fatalf("retrieve output = %s", stdout.String())
	}
}

func TestExecuteIndexAndRetrieveWithQdrantBackend(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Qdrant Demo"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	parsed := `{"PaperID":"paper-1","Sections":[{"ID":"s1","Passages":[{"ID":"p1","PaperID":"paper-1","SectionID":"s1","Text":"Solar fuel catalysts split water."}]}]}`
	if err := os.WriteFile(filepath.Join(parsedDir, "paper-1.json"), []byte(parsed), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/researchforge_passages":
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/collections/researchforge_passages/points":
			var request struct {
				Points []struct {
					Vector []float64 `json:"vector"`
				} `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode qdrant upsert: %v", err)
			}
			if len(request.Points) != 1 || len(request.Points[0].Vector) != 12 {
				t.Fatalf("qdrant vectors = %#v", request.Points)
			}
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		case "/collections/researchforge_passages/points/search":
			var request struct {
				Vector []float64 `json:"vector"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode qdrant search: %v", err)
			}
			if len(request.Vector) != 12 {
				t.Fatalf("qdrant query vector length = %d", len(request.Vector))
			}
			_, _ = w.Write([]byte(`{"result":[{"payload":{"PaperID":"paper-1","SectionID":"s1","PassageID":"p1","Text":"Solar fuel catalysts split water."}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	t.Setenv("RFORGE_EMBEDDING_DIMENSIONS", "12")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "index", "rebuild", "--backend", "qdrant"}, stdout, stderr); code != 0 {
		t.Fatalf("qdrant index exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"backend":"qdrant"`) {
		t.Fatalf("index output = %s", stdout.String())
	}
	lockData, err := os.ReadFile(filepath.Join(dir, "data", "retrieval.lock.json"))
	if err != nil {
		t.Fatalf("read retrieval lock: %v", err)
	}
	if !strings.Contains(string(lockData), `"embeddingVersion": "dimensions=12"`) {
		t.Fatalf("retrieval lock = %s", lockData)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "retrieve", "--query", "solar catalysts", "--backend", "qdrant"}, stdout, stderr); code != 0 {
		t.Fatalf("qdrant retrieve exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Solar fuel catalysts") || !strings.Contains(stdout.String(), `"backend":"qdrant"`) {
		t.Fatalf("retrieve output = %s", stdout.String())
	}
}

func TestExecuteQdrantBackendUsesHTTPEmbeddingProvider(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "HTTP Embeddings"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	parsed := `{"PaperID":"paper-1","Sections":[{"ID":"s1","Passages":[{"ID":"p1","PaperID":"paper-1","SectionID":"s1","Text":"Solar fuel catalysts split water."}]}]}`
	if err := os.WriteFile(filepath.Join(parsedDir, "paper-1.json"), []byte(parsed), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	var embeddingCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/researchforge_passages":
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/embed":
			embeddingCalls++
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode embedding request: %v", err)
			}
			if request["model"] != "fixture-embed" {
				t.Fatalf("embedding request = %#v", request)
			}
			_, _ = w.Write([]byte(`{"embedding":[0.25,0.75]}`))
		case "/collections/researchforge_passages/points":
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		case "/collections/researchforge_passages/points/search":
			_, _ = w.Write([]byte(`{"result":[{"payload":{"PaperID":"paper-1","SectionID":"s1","PassageID":"p1","Text":"Solar fuel catalysts split water."}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	t.Setenv("RFORGE_EMBEDDING_URL", server.URL+"/embed")
	t.Setenv("RFORGE_EMBEDDING_MODEL", "fixture-embed")
	t.Setenv("RFORGE_EMBEDDING_CONSENT", "1")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "index", "rebuild", "--backend", "qdrant"}, stdout, stderr); code != 0 {
		t.Fatalf("qdrant index exit code = %d, stderr = %s", code, stderr.String())
	}
	if embeddingCalls != 1 {
		t.Fatalf("embedding calls after rebuild = %d", embeddingCalls)
	}
	lockData, err := os.ReadFile(filepath.Join(dir, "data", "retrieval.lock.json"))
	if err != nil {
		t.Fatalf("read retrieval lock: %v", err)
	}
	if !strings.Contains(string(lockData), `"embeddingBackend": "http-embedding:fixture-embed"`) || !strings.Contains(string(lockData), `"embeddingVersion": "http-model=fixture-embed"`) {
		t.Fatalf("retrieval lock = %s", lockData)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "retrieve", "--query", "solar catalysts", "--backend", "qdrant"}, stdout, stderr); code != 0 {
		t.Fatalf("qdrant retrieve exit code = %d, stderr = %s", code, stderr.String())
	}
	if embeddingCalls != 2 {
		t.Fatalf("embedding calls after retrieve = %d", embeddingCalls)
	}
}

func TestExecuteIndexAndRetrieveWithOpenSearchBackend(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "OpenSearch Demo"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	parsed := `{"PaperID":"paper-1","Sections":[{"ID":"s1","Passages":[{"ID":"p1","PaperID":"paper-1","SectionID":"s1","Text":"Solar fuel catalysts split water."}]}]}`
	if err := os.WriteFile(filepath.Join(parsedDir, "paper-1.json"), []byte(parsed), 0o644); err != nil {
		t.Fatalf("write parsed: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/researchforge-passages":
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case "/researchforge-passages/_bulk", "/researchforge-passages/_refresh":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/researchforge-passages/_search":
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"PaperID":"paper-1","SectionID":"s1","PassageID":"p1","Text":"Solar fuel catalysts split water."}}]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENSEARCH_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "index", "rebuild", "--backend", "opensearch"}, stdout, stderr); code != 0 {
		t.Fatalf("opensearch index exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"backend":"opensearch"`) {
		t.Fatalf("index output = %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "retrieve", "--query", "solar catalysts", "--backend", "opensearch"}, stdout, stderr); code != 0 {
		t.Fatalf("opensearch retrieve exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Solar fuel catalysts") || !strings.Contains(stdout.String(), `"backend":"opensearch"`) {
		t.Fatalf("retrieve output = %s", stdout.String())
	}
}

func TestExecuteParseGROBIDRecordsParsedDocumentAndProvenance(t *testing.T) {
	tei := `<TEI><teiHeader><fileDesc><titleStmt><title>Artificial photosynthesis parsed CLI</title></titleStmt><profileDesc><abstract><p>Parsed abstract.</p></abstract></profileDesc></fileDesc></teiHeader><text><body><div><head>Intro</head><p>Passage text.</p></div></body></text></TEI>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(tei))
	}))
	defer server.Close()
	t.Setenv("RFORGE_GROBID_URL", server.URL)
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	pdf := filepath.Join(t.TempDir(), "paper.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", dir, "parse", "--paper", "paper-1", "--parser", "grobid", "--pdf", pdf}, stdout, stderr)
	if code != 0 {
		t.Fatalf("parse exit code = %d, stderr = %s", code, stderr.String())
	}
	parsedPath := filepath.Join(dir, "parsed", "paper-1.json")
	data, err := os.ReadFile(parsedPath)
	if err != nil {
		t.Fatalf("read parsed doc: %v", err)
	}
	if !strings.Contains(string(data), "Artificial photosynthesis parsed CLI") || !strings.Contains(string(data), "Passage text.") {
		t.Fatalf("parsed doc = %s", data)
	}
	events, err := provenance.Read(dir)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	if !sawAction(events, "parser.run") {
		t.Fatalf("missing parser.run event: %#v", events)
	}
}

func TestExecutePDFFetchByDOIWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("%PDF-1.4 cli fetched fixture"))
	}))
	defer server.Close()
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", dir, "pdf", "fetch", "--doi", "10.1000/example", "--pdf-url", server.URL + "/paper.pdf", "--license", "cc-by", "--oa-status", "gold"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("pdf fetch exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "documents", "open-access", "10-1000-example.pdf")); err != nil {
		t.Fatalf("missing fetched PDF: %v", err)
	}
}

func TestExecuteOSSRefreshStoresMetadata(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "oss", "add", "owner/repo"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("oss add exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "oss", "refresh", "owner/repo", "--interval", "daily", "--stale", "--archived"}, stdout, stderr); code != 0 {
		t.Fatalf("oss refresh exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "oss", "list"}, stdout, stderr); code != 0 {
		t.Fatalf("oss list exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"RefreshInterval":"daily"`) || !strings.Contains(stdout.String(), `"Archived":true`) || !strings.Contains(stdout.String(), `"Stale":true`) {
		t.Fatalf("list output = %s", stdout.String())
	}
}

func TestExecuteOSSScanAndReport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", dir, "oss", "add", "owner/repo", "--area", "literature tooling"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("oss add exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "oss", "scan", "owner/repo", "--topic", "deduplication"}, stdout, stderr); code != 0 {
		t.Fatalf("oss scan exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "oss", "report", "--area", "literature tooling"}, stdout, stderr); code != 0 {
		t.Fatalf("oss report exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "owner/repo") || !strings.Contains(stdout.String(), "OSS report") {
		t.Fatalf("report output = %s", stdout.String())
	}
}

func TestExecuteOSSNoteCreatesTemplate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "oss", "note", "owner/repo", "--area", "architecture"}, stdout, stderr); code != 0 {
		t.Fatalf("oss note exit code = %d, stderr = %s", code, stderr.String())
	}
	path := filepath.Join(dir, "opensource", "notes", "owner", "repo", "architecture.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "Do not copy external source code") {
		t.Fatalf("note = %s", data)
	}
}

func TestExecuteOSSInventoryDriftJSON(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("# Wrong\n\nArea: parser\n"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"alpha","name":"Alpha","area":"scholarly-graph-source","disposition":"adapter-only","licensePolicy":"adapter","note":"alpha.md","risk":"drift","nextSlice":"fix"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "oss", "inventory-drift", manifest}, stdout, stderr)
	if code != 0 {
		t.Fatalf("inventory-drift exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "note heading") || !strings.Contains(stdout.String(), "note area") {
		t.Fatalf("stdout missing drift issues:\n%s", stdout.String())
	}
}

func TestExecuteOSSInventoryPolicyJSON(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"old","name":"Old","area":"parser","disposition":"adapter-only","licensePolicy":"adapter","note":"old.md","risk":"stale","nextSlice":"refresh","licenseSPDX":"MIT","pushedAt":"2024-01-01T00:00:00Z"},{"id":"bad","name":"Bad","area":"parser","disposition":"integrate","licensePolicy":"review","note":"bad.md","risk":"license","nextSlice":"avoid","licenseSPDX":"AGPL-3.0-only","pushedAt":"2026-01-01T00:00:00Z"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "oss", "inventory-policy", manifest, "--stale-after", "18mo", "--now", "2026-06-14T00:00:00Z"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("inventory-policy exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "old: stale") || !strings.Contains(stdout.String(), "bad: copyleft license") {
		t.Fatalf("stdout missing policy issues:\n%s", stdout.String())
	}
}

func TestExecuteOSSInventoryRefreshGitHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/zotero/zotero" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"stargazers_count":321,"forks_count":54,"archived":true,"pushed_at":"2026-06-01T00:00:00Z","license":{"spdx_id":"AGPL-3.0-only"}}`))
	}))
	defer server.Close()
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"zotero","name":"Zotero","repository":"zotero/zotero","area":"reference-management","disposition":"pattern-reference","licensePolicy":"study","note":"zotero.md","risk":"license","nextSlice":"metadata"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "oss", "inventory-refresh", manifest, "--source", "github", "--base-url", server.URL}, stdout, stderr)
	if code != 0 {
		t.Fatalf("inventory-refresh exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"refreshed":1`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	for _, want := range []string{`"stars": 321`, `"forks": 54`, `"licenseSPDX": "AGPL-3.0-only"`, `"archived": true`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("manifest missing %s:\n%s", want, data)
		}
	}
}

func TestExecuteOSSInventoryReportFiltersArea(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"openalex","name":"OpenAlex","area":"scholarly-graph-source","disposition":"adapter-only","licensePolicy":"api","note":"openalex.md","risk":"cursor risk","nextSlice":"paginated import"},{"id":"zotero","name":"Zotero","area":"reference-management","disposition":"pattern-reference","licensePolicy":"study","note":"zotero.md","risk":"attachment risk","nextSlice":"collections"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"oss", "inventory-report", manifest, "--area", "scholarly-graph-source"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("inventory-report exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "OpenAlex") || !strings.Contains(stdout.String(), "paginated import") {
		t.Fatalf("report missing OpenAlex row:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Zotero") {
		t.Fatalf("area-filtered report included Zotero:\n%s", stdout.String())
	}
}

func TestExecuteOSSInventoryCheckJSON(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(filepath.Join(root, "zotero.md"), []byte("# Zotero\n"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"zotero","name":"Zotero","area":"reference-management","disposition":"pattern-reference","licensePolicy":"study-only","note":"zotero.md","risk":"review required","nextSlice":"CSL JSON"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "oss", "inventory-check", manifest}, stdout, stderr)
	if code != 0 {
		t.Fatalf("inventory-check exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var envelope struct {
		Data struct {
			EntryCount int      `json:"entryCount"`
			Issues     []string `json:"issues"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.Data.EntryCount != 1 || len(envelope.Data.Issues) != 0 {
		t.Fatalf("inventory check = %#v", envelope.Data)
	}
}

func TestExecuteOSSAddListAndLicenseCheck(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "oss", "add", "owner/repo", "--area", "literature tooling"}, stdout, stderr); code != 0 {
		t.Fatalf("oss add exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "oss", "list"}, stdout, stderr); code != 0 {
		t.Fatalf("oss list exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	items := envelope["data"].(map[string]any)["repositories"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["Name"] != "owner/repo" {
		t.Fatalf("items = %#v", items)
	}
	cloneDir := filepath.Join(dir, "opensource", "clones", "owner", "repo")
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatalf("create clone dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cloneDir, "LICENSE"), []byte("MIT License"), 0o644); err != nil {
		t.Fatalf("write license: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "oss", "license-check", "owner/repo"}, stdout, stderr); code != 0 {
		t.Fatalf("oss license-check exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"Kind":"MIT"`) {
		t.Fatalf("license output = %s", stdout.String())
	}
}

func TestExecuteImportExportJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(importPath, []byte(`[{"Title":"Artificial photosynthesis JSON import","Identifiers":{"DOI":"10.1000/json-import"}}]`), 0o644); err != nil {
		t.Fatalf("write import: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, stdout, stderr); code != 0 {
		t.Fatalf("import exit code = %d, stderr = %s", code, stderr.String())
	}
	var importEnvelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &importEnvelope); err != nil {
		t.Fatalf("import stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if importEnvelope["data"].(map[string]any)["imported"] != float64(1) {
		t.Fatalf("import envelope = %#v", importEnvelope)
	}

	exportPath := filepath.Join(t.TempDir(), "export.json")
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "export", "json", exportPath}, stdout, stderr); code != 0 {
		t.Fatalf("export exit code = %d, stderr = %s", code, stderr.String())
	}
	exported, _, err := library.ImportJSON(exportPath)
	if err != nil {
		t.Fatalf("read exported JSON: %v", err)
	}
	if len(exported) != 1 || exported[0].Title != "Artificial photosynthesis JSON import" {
		t.Fatalf("exported = %#v", exported)
	}
}

func TestExecuteLibraryImportCrossrefReferences(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Crossref References"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.5555/source" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"DOI":"10.5555/source","reference":[{"DOI":"10.1000/ref-one","article-title":"Reference One","key":"ref1"},{"article-title":"Title only reference","key":"ref2"}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_CROSSREF_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "library", "import-crossref-refs", "10.5555/source"}, stdout, stderr); code != 0 {
		t.Fatalf("import refs exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"extracted":2`) || !strings.Contains(stdout.String(), `"imported":1`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	listed := mustRunCLI(t, "--json", "--project", dir, "library", "list")
	if !strings.Contains(string(listed), "Reference One") || strings.Contains(string(listed), "Title only reference") {
		t.Fatalf("library = %s", listed)
	}
}

func TestExecuteLibraryRefreshCrossrefAllDOIRecords(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Crossref Batch Refresh"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"Old one","Identifiers":{"DOI":"10.5555/one"}},
  {"Title":"Old two","Identifiers":{"DOI":"10.5555/two"}},
  {"Title":"No DOI","Identifiers":{"OpenAlexID":"W123"}}
]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("import exit code = %d", code)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch r.URL.Path {
		case "/works/10.5555/one":
			_, _ = w.Write([]byte(`{"message":{"DOI":"10.5555/one","title":["Refreshed one"],"publisher":"Crossref One"}}`))
		case "/works/10.5555/two":
			_, _ = w.Write([]byte(`{"message":{"DOI":"10.5555/two","title":["Refreshed two"],"publisher":"Crossref Two"}}`))
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("RFORGE_CROSSREF_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "library", "refresh-crossref"}, stdout, stderr); code != 0 {
		t.Fatalf("batch refresh exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %#v, want two DOI refreshes", requests)
	}
	if !strings.Contains(stdout.String(), `"refreshed":2`) || !strings.Contains(stdout.String(), `"skippedNoDOI":1`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	listed := string(mustRunCLI(t, "--json", "--project", dir, "library", "list"))
	for _, want := range []string{"Refreshed one", "Crossref One", "Refreshed two", "Crossref Two", "No DOI"} {
		if !strings.Contains(listed, want) {
			t.Fatalf("library missing %q:\n%s", want, listed)
		}
	}
}

func TestExecuteLibraryRefreshDOIFromCrossref(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Crossref Refresh"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "library.json")
	fixture := `[{"Title":"Old title","Identifiers":{"DOI":"10.5555/refresh"}}]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("import exit code = %d", code)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.5555/refresh" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"DOI":"10.5555/refresh","title":["Refreshed title"],"publisher":"Crossref Publisher","reference-count":3,"license":[{"URL":"https://license.example"}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_CROSSREF_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "library", "refresh-doi", "10.5555/refresh"}, stdout, stderr); code != 0 {
		t.Fatalf("refresh exit code = %d, stderr = %s", code, stderr.String())
	}
	var lib struct {
		Data struct {
			Papers []library.PaperRecord `json:"papers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", dir, "library", "list"), &lib); err != nil {
		t.Fatalf("decode library: %v", err)
	}
	if len(lib.Data.Papers) != 1 || lib.Data.Papers[0].Title != "Refreshed title" || lib.Data.Papers[0].Publisher != "Crossref Publisher" || lib.Data.Papers[0].License != "https://license.example" {
		t.Fatalf("papers = %#v", lib.Data.Papers)
	}
	if lib.Data.Papers[0].SourceRefs[0].Metadata["reference_count"] != "3" {
		t.Fatalf("source refs = %#v", lib.Data.Papers[0].SourceRefs)
	}
}

func TestExecuteCSLJSONImportExportRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Zotero Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "zotero.csl.json")
	fixture := `[{"id":"smith2026crypto","type":"article-journal","title":"Leak-free LightGBM for crypto price data","DOI":"10.1000/csl","issued":{"date-parts":[[2026]]},"author":[{"given":"Jane","family":"Smith"}],"container-title":"Journal of Financial ML"}]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "import", "csl-json", importPath}, stdout, stderr); code != 0 {
		t.Fatalf("csl-json import exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	exportPath := filepath.Join(t.TempDir(), "export.csl.json")
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "export", "csl-json", exportPath}, stdout, stderr); code != 0 {
		t.Fatalf("csl-json export exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	exported, skipped, err := library.ImportCSLJSON(exportPath)
	if err != nil {
		t.Fatalf("read exported CSL JSON: %v", err)
	}
	if skipped != 0 || len(exported) != 1 || exported[0].SourceRefs[0].Metadata["csl_id"] != "smith2026crypto" {
		t.Fatalf("exported=%#v skipped=%d", exported, skipped)
	}
}

func TestExecuteZoteroRDFImportExportRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Zotero RDF Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "zotero.rdf")
	fixture := `<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:prism="http://prismstandard.org/namespaces/1.2/basic/" xmlns:bib="http://purl.org/net/biblio#" xmlns:better-bibtex="https://retorque.re/zotero-better-bibtex/export#"><bib:Article rdf:about="#item-1"><dc:title>Zotero RDF CLI fixture</dc:title><prism:doi>10.1000/rdf-cli</prism:doi><better-bibtex:citekey>cli2026rdf</better-bibtex:citekey></bib:Article></rdf:RDF>`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "import", "zotero-rdf", importPath}, stdout, stderr); code != 0 {
		t.Fatalf("zotero-rdf import exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	exportPath := filepath.Join(t.TempDir(), "export.rdf")
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", dir, "export", "zotero-rdf", exportPath}, stdout, stderr); code != 0 {
		t.Fatalf("zotero-rdf export exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	exported, skipped, err := library.ImportZoteroRDF(exportPath)
	if err != nil {
		t.Fatalf("read exported Zotero RDF: %v", err)
	}
	if skipped != 0 || len(exported) != 1 || exported[0].Title != "Zotero RDF CLI fixture" || exported[0].SourceRefs[0].Metadata["citation_key"] != "cli2026rdf" {
		t.Fatalf("exported=%#v skipped=%d", exported, skipped)
	}
}

func TestExecuteImportSkipsDuplicatesAndReports(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	importPath := filepath.Join(t.TempDir(), "import.json")
	fixture := `[
  {"Title":"First","Identifiers":{"DOI":"10.1000/dup"}},
  {"Title":"Duplicate of first","Identifiers":{"DOI":"10.1000/dup"}},
  {"Title":"No identifier"},
  {"Title":"Distinct","Identifiers":{"DOI":"10.1000/distinct"}}
]`
	if err := os.WriteFile(importPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write import: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "import", "json", importPath}, stdout, stderr); code != 0 {
		t.Fatalf("import aborted on duplicate/no-identifier records: exit=%d stderr=%s", code, stderr.String())
	}
	var env struct {
		Data struct {
			Imported            int      `json:"imported"`
			SkippedDuplicate    []string `json:"skipped_duplicate"`
			SkippedNoIdentifier int      `json:"skipped_no_identifier"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("import stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if env.Data.Imported != 2 {
		t.Fatalf("imported = %d, want 2", env.Data.Imported)
	}
	if len(env.Data.SkippedDuplicate) != 1 || env.Data.SkippedDuplicate[0] != "10.1000/dup" {
		t.Fatalf("skipped_duplicate = %v, want [10.1000/dup]", env.Data.SkippedDuplicate)
	}
	if env.Data.SkippedNoIdentifier != 1 {
		t.Fatalf("skipped_no_identifier = %d, want 1", env.Data.SkippedNoIdentifier)
	}
}

func TestExecuteLibraryListJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	record, err := library.NewPaperRecord(library.PaperRecordInput{Title: "Artificial photosynthesis catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/example"}})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "--project", dir, "library", "list"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	papers := data["papers"].([]any)
	if len(papers) != 1 {
		t.Fatalf("len(papers) = %d, want 1", len(papers))
	}
	paper := papers[0].(map[string]any)
	if paper["Title"] != "Artificial photosynthesis catalyst review" {
		t.Fatalf("paper title = %#v", paper["Title"])
	}
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
