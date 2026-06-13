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

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
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

func TestExecuteDecisionsIssueBodyForLicenseDecisionIncludesBlockedItem(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--issue-body", "project_license"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"project_license", "owner_decision_required", "https://github.com/TrebuchetDynamics/research-forge/issues/1", "Recommended issue routing", "Labels", "decision", "owner-input-needed", "Milestone", "Owner decisions", "Add license after owner decision", "Options considered", "MIT", "Apache-2.0", "No public license yet", "Required owner response fields", "Owner response template", "License SPDX identifier", "Approved by", "Approval date", "Pending owner selection", "Implementation steps after approval", "make check", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("license issue body missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsIssueBodyRejectsCompletedWebGUITracker(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--issue-body", "web_gui_stack_scope"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit for completed tracker")
	}
	if !strings.Contains(stderr.String(), "unknown decision") || !strings.Contains(stderr.String(), "web_gui_stack_scope") {
		t.Fatalf("stderr missing completed tracker rejection: %s", stderr.String())
	}
}

func TestExecuteDecisionsIssueBodyForOwnerDecision(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--issue-body", "project_license"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"Decision ID", "project_license", "Current issue title", "Owner decision: project_license (SPDX, copyright holder, approver, date required)", "Blocked TODO items", "TODO.md:34", "Add license after owner decision", "Options considered", "Owner inputs needed", "License choice with SPDX identifier", "NOASSERTION", "Exact copyright holder string", "Implementation steps after approval", "make license-decision-approval-gate"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("issue body missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsIssueBodyMatchesTrackedDraft(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--issue-body", "project_license"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "docs", "decisions", "project_license_issue.md"))
	if err != nil {
		t.Fatalf("read project license issue draft: %v", err)
	}
	if stdout.String() != string(want) {
		t.Fatalf("generated issue body drifted from tracked draft\n--- generated ---\n%s\n--- want ---\n%s", stdout.String(), string(want))
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

func TestExecuteDecisionsJSONExposesIssueTitleAndTodoRefs(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"issue_title\"", "Owner decision: project_license (SPDX, copyright holder, approver, date required)", "\"todo_refs\"", "TODO.md:34"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json output missing %q:\n%s", want, stdout.String())
		}
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

func TestExecuteDecisionsJSONIncludesLicenseOwnerInputs(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"owner_inputs\"", "license choice", "copyright holder string", "patent posture", "adoption model"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json output missing owner input %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsJSONIncludesLicenseOwnerResponseFields(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"owner_response_required_fields\"", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json output missing owner response field %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsJSONMarksLicenseAsOwnerActionRequired(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"blocker_kind\":\"owner_decision\"", "\"owner_action_required\":true"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json output missing owner-action marker %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsJSONIncludesIssueRoutingForLicenseBlocker(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"issue_labels\":[\"decision\",\"blocked\",\"owner-input-needed\"]", "\"milestone\":\"Owner decisions\""} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json output missing issue routing %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsJSONIncludesLicenseImplementationSteps(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"\"implementation_steps\"", "Add LICENSE", "Update README.md license section", "Update TODO.md license checkbox", "make license-decision-approval-gate", "make check", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json output missing implementation step %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsPlainOutputShowsOwnerAction(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"project_license", "owner_decision_required", "owner required", "license choice", "copyright holder string"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("decisions output missing %q:\n%s", want, stdout.String())
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

func TestExecuteDecisionsMarkdownPrintsAuditTable(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"decisions", "--markdown"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"| Decision ID | Status | Owner Action | TODO Lines | Blocked TODOs | Evidence |", "project_license", "owner required", "license choice", "docs/remaining-todo-audit.md", "owner-input-needed", "Owner decisions", "MIT", "No public license yet", "response fields", "License SPDX identifier", "Approval date", "make license-decision-approval-gate", "approved:true"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("markdown output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteDecisionsHasNoStaleTODOCoverage(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, covered := range []string{
		"Add license after owner decision",
	} {
		found := false
		for _, remaining := range uncheckedTODOItems(t) {
			if remaining == covered {
				found = true
			}
		}
		if !found {
			t.Fatalf("decisions output covers stale or checked TODO %q", covered)
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
	for _, want := range []string{"completion audit verified", "all unchecked TODO items are decision/tracker-covered", "issue references verified", "unchecked TODO refs verified: 1", "completion remains blocked by 1 decision(s)", "blocked decision ids: project_license", "blocked issue urls: https://github.com/TrebuchetDynamics/research-forge/issues/1", "license TODO owner inputs verified", "license file absent while owner decision pending", "license decision remains pending", "license decision draft owner inputs verified", "license decision required response fields verified", "license owner approval absent verified", "README license section remains pending", "license owner inputs verified", "license owner response fields verified", "license options verified", "license implementation steps verified", "license issue routing verified", "license issue title verified", "remaining TODO audit verified", "license approval gate target verified", "license decision brief verified", "owner decisions license section verified", "owner decision template verified", "owner decision template response fields verified", "PR license gate verified", "contributing license workflow verified"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("completion audit output missing %q:\n%s", want, stdout.String())
		}
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
	for _, want := range []string{"\"completion_audit_verified\":true", "\"completion_audit_issue_refs_verified\":true", "\"checked_evidence_verified\":true", "\"quality_gate_verified\":true", "\"completion_blocked\":true", "\"blocked_decisions\":1", "\"blocked_decision_ids\":[\"project_license\"]", "\"blocked_issue_urls\":[\"https://github.com/TrebuchetDynamics/research-forge/issues/1\"]", "\"line_refs_verified\":true", "\"issue_refs_verified\":true", "\"unchecked_refs\":1", "\"license_todo_owner_inputs_verified\":true", "\"license_file_absent_when_blocked\":true", "\"license_decision_pending_verified\":true", "\"license_decision_draft_owner_inputs_verified\":true", "\"license_decision_required_response_fields_verified\":true", "\"license_owner_approval_absent_verified\":true", "\"readme_license_pending_verified\":true", "\"license_owner_inputs_verified\":true", "\"license_owner_response_fields_verified\":true", "\"license_options_verified\":true", "\"license_implementation_steps_verified\":true", "\"license_issue_routing_verified\":true", "\"license_issue_title_verified\":true", "\"remaining_todo_audit_verified\":true", "\"license_approval_gate_target_verified\":true", "\"license_decision_brief_verified\":true", "\"owner_decisions_license_section_verified\":true", "\"owner_decision_template_verified\":true", "\"owner_decision_template_response_fields_verified\":true", "\"pr_license_gate_verified\":true", "\"contributing_license_workflow_verified\":true"} {
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
	for _, want := range []string{"\"line_refs_verified\":true", "\"issue_refs_verified\":true", "\"unchecked_refs\":1"} {
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

func TestExecuteDecisionsCoversEveryRemainingUncheckedTODO(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, remaining := range uncheckedTODOItems(t) {
		if !strings.Contains(stdout.String(), remaining) {
			t.Fatalf("decisions output does not cover remaining TODO %q:\n%s", remaining, stdout.String())
		}
	}
}

func uncheckedTODOLineRefs(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	refs := []string{}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- [ ] ") {
			refs = append(refs, fmt.Sprintf("TODO.md:%d", i+1))
		}
	}
	return refs
}

func uncheckedTODOItems(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	items := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		items = append(items, item)
	}
	if len(items) == 0 {
		t.Fatalf("expected decision-gated unchecked TODO items")
	}
	return items
}

func TestExecuteDecisionsJSONListsOwnerBlockedTODOs(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "decisions"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"project_license", "docs/owner-decisions.md", "docs/remaining-todo-audit.md", "https://github.com/TrebuchetDynamics/research-forge/issues/1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("decisions output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestExecuteUIJSONReportsReadyWebGUI(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "ui"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "go_htmx_web_gui_ready") || !strings.Contains(stdout.String(), "ADR 0006") {
		t.Fatalf("ui output = %s", stdout.String())
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

func TestExecuteSearchOpenAlexJSONWithMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "artificial photosynthesis" {
			t.Fatalf("search = %q", r.URL.Query().Get("search"))
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W123","doi":"https://doi.org/10.1000/example","title":"Artificial photosynthesis catalyst review","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "search", "--source", "openalex", "--query", "artificial photosynthesis", "--limit", "1"}, stdout, stderr)
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
	exported, err := library.ImportJSON(exportPath)
	if err != nil {
		t.Fatalf("read exported JSON: %v", err)
	}
	if len(exported) != 1 || exported[0].Title != "Artificial photosynthesis JSON import" {
		t.Fatalf("exported = %#v", exported)
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
