package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevelopmentPlanDocumentsLicenseReleaseGate(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "DEVELOPMENT_PLAN.md"))
	if err != nil {
		t.Fatalf("read development plan: %v", err)
	}
	text := string(data)
	for _, want := range []string{"make license-decision-live-audit", "make license-decision-approval-gate", "TODO.md:34", "approved:true", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
		if !strings.Contains(text, want) {
			t.Fatalf("development plan missing license release gate %q", want)
		}
	}
}

func TestReleaseMilestonePlansIncludeLicenseApprovalGate(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, doc := range []string{"pre-alpha-release.md", "alpha-release.md", "beta-release.md"} {
		data, err := os.ReadFile(filepath.Join(root, "docs", doc))
		if err != nil {
			t.Fatalf("read %s: %v", doc, err)
		}
		text := string(data)
		for _, want := range []string{"make license-decision-live-audit", "make license-decision-approval-gate", "approved:true", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing license approval gate %q", doc, want)
			}
		}
	}
}

func TestReleaseNotesTemplateIncludesLicenseApprovalReceipt(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "release-notes-template.md"))
	if err != nil {
		t.Fatalf("read release notes template: %v", err)
	}
	text := string(data)
	for _, want := range []string{"make license-decision-live-audit", "make license-decision-approval-gate", "approved:true", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
		if !strings.Contains(text, want) {
			t.Fatalf("release notes template missing license approval receipt %q", want)
		}
	}
}

func TestReleaseDocsDocumentLicenseApprovalGate(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "release.md"))
	if err != nil {
		t.Fatalf("read release docs: %v", err)
	}
	text := string(data)
	for _, want := range []string{"make license-decision-live-audit", "make license-decision-approval-gate", "TODO.md:34", "approved:true", "LICENSE", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
		if !strings.Contains(text, want) {
			t.Fatalf("release docs missing license approval gate %q", want)
		}
	}
}

func TestDeveloperSetupDocumentsLicenseDecisionGates(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "developer-setup.md"))
	if err != nil {
		t.Fatalf("read developer setup: %v", err)
	}
	text := string(data)
	for _, want := range []string{"make check", "make license-decision-live-audit", "make license-decision-approval-gate", "approved:false", "approved:true"} {
		if !strings.Contains(text, want) {
			t.Fatalf("developer setup missing license decision gate %q", want)
		}
	}
}

func TestMakefileHasLiveLicenseDecisionAuditTarget(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	for _, want := range []string{"license-decision-live-audit", "license-decision-approval-gate", "gh issue view 1", "grep -q", "license decision approval missing", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date", "approved"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile missing live license audit target evidence %q", want)
		}
	}
}

func TestOneDotZeroReleaseGateDocumentsLicenseBlocker(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "1.0-release.md"))
	if err != nil {
		t.Fatalf("read 1.0 release gate: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Go + HTMX", "project_license", "TODO.md:34", "make license-decision-live-audit", "make license-decision-approval-gate", "approved:true", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
		if !strings.Contains(text, want) {
			t.Fatalf("1.0 release gate missing %q", want)
		}
	}
}

func TestCLIReferenceDocumentsDecisionAuditModes(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "cli.md"))
	if err != nil {
		t.Fatalf("read CLI reference: %v", err)
	}
	text := string(data)
	for _, want := range []string{"rforge decisions --check TODO.md", "tracking issue references", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md", "completion_blocked", "blocked_decisions", "blocked_decision_ids", "issue_title", "todo_refs", "issue_labels", "milestone", "options_considered", "owner_response_required_fields", "license_owner_approval_absent_verified", "license_owner_response_fields_verified", "license_options_verified", "license_issue_routing_verified", "license_issue_title_verified", "remaining_todo_audit_verified", "make license-decision-live-audit", "make license-decision-approval-gate", "approved:true", "rforge decisions --markdown", "routing/options", "rforge decisions --issue-body <decision-id>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("CLI reference missing %q", want)
		}
	}
}

func TestUncheckedTodosPointToDecisionCommands(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		if !strings.Contains(line, "rforge decisions") && !strings.Contains(line, "local web GUI build decision") && !strings.Contains(line, "local Go + HTMX implementation") && !strings.Contains(line, "Pending Go + HTMX web GUI implementation") {
			t.Fatalf("unchecked TODO does not point to decision command or web GUI decision: %s", line)
		}
	}
}

func TestLicenseTodoNamesRequiredOwnerApprovalFields(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") || !strings.Contains(line, "Add license after owner decision") {
			continue
		}
		for _, want := range []string{"SPDX identifier", "exact copyright holder", "approver", "approval date", "make license-decision-approval-gate", "approved:true"} {
			if !strings.Contains(line, want) {
				t.Fatalf("license TODO missing required owner approval field %q: %s", want, line)
			}
		}
		return
	}
	t.Fatalf("unchecked license TODO not found")
}

func TestUncheckedTodosReferenceTrackingIssues(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		if strings.Contains(line, "license") && !strings.Contains(line, "#1") {
			t.Fatalf("license TODO does not reference tracking issue #1: %s", line)
		}
		if strings.Contains(line, "web GUI") && !strings.Contains(line, "#2") {
			t.Fatalf("web GUI TODO does not reference tracking issue #2: %s", line)
		}
	}
}

func TestRemainingTodoAuditDocumentsExecutableAuditCommand(t *testing.T) {
	root := filepath.Join("..", "..")
	auditData, err := os.ReadFile(filepath.Join(root, "docs", "remaining-todo-audit.md"))
	if err != nil {
		t.Fatalf("read remaining audit: %v", err)
	}
	audit := string(auditData)
	for _, want := range []string{"make todo-audit", "make todo-completion-audit", "make license-decision-live-audit", "approved", "completion_blocked", "blocked_decisions", "blocked_decision_ids", "license_decision_required_response_fields_verified", "license_owner_approval_absent_verified", "license_owner_response_fields_verified", "owner_decision_template_response_fields_verified", "license_options_verified", "license_issue_routing_verified", "license_issue_title_verified", "remaining_todo_audit_verified", "verify decision line references", "verify tracking issue references", "Prompt-to-artifact checklist", "decision-resolution-checklist.md", "go test ./...", "go vet ./...", "git diff --check"} {
		if !strings.Contains(audit, want) {
			t.Fatalf("remaining TODO audit missing %q", want)
		}
	}
}

func TestTodoReferencesCompletionAudit(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	if !strings.Contains(string(data), "docs/todo-completion-audit.md") {
		t.Fatalf("TODO.md does not reference docs/todo-completion-audit.md")
	}
}

func TestTodoCompletionAuditCoversUncheckedTodos(t *testing.T) {
	root := filepath.Join("..", "..")
	todoData, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	auditData, err := os.ReadFile(filepath.Join(root, "docs", "todo-completion-audit.md"))
	if err != nil {
		t.Fatalf("read TODO completion audit: %v", err)
	}
	audit := string(auditData)
	for _, line := range strings.Split(string(todoData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		if !strings.Contains(audit, item) {
			t.Fatalf("TODO completion audit does not cover unchecked TODO %q", item)
		}
	}
}

func TestTodoCompletionAuditMapsObjectiveToEvidence(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "todo-completion-audit.md"))
	if err != nil {
		t.Fatalf("read TODO completion audit: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"TODO.md",
		"Success criteria",
		"Prompt-to-artifact checklist",
		"issue #1",
		"issue #2",
		"make check",
		"make license-decision-live-audit",
		"make license-decision-approval-gate",
		"approved:false",
		"rforge decisions --check TODO.md",
		"rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md",
		"completion_blocked",
		"blocked_decisions",
		"blocked_decision_ids",
		"license_decision_required_response_fields_verified",
		"license_owner_response_fields_verified",
		"owner_decision_template_response_fields_verified",
		"license_options_verified",
		"license_issue_routing_verified",
		"web-gui-smoke",
		"internal/webui",
		"web/assets/researchforge.css",
		"Go + HTMX",
		"Go HTMLX",
		"SKILLS.md",
		"research-forge-web-ui-tdd",
		"skills/research-forge-web-ui-tdd/SKILL.md",
		"Add license after owner decision",
		"Add Go + HTMX web GUI workspace/dependencies",
		"View CLI-generated papers, meta-analysis outputs, PRISMA/citation diagrams, and report artifacts in the web GUI",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("TODO completion audit missing %q", want)
		}
	}
}

func TestReadmeLicenseSectionNamesRequiredOwnerResponseFields(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(data)
	for _, want := range []string{"No license has been selected yet", "license choice", "copyright holder string", "license SPDX identifier", "approver", "approval date"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README license section missing %q", want)
		}
	}
}

func TestLicenseDecisionBriefNamesRequiredOwnerResponseFields(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "license-decision.md"))
	if err != nil {
		t.Fatalf("read license decision brief: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Required owner response fields", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date", "make license-decision-live-audit", "make license-decision-approval-gate", "approved:true", "issue #1", "TODO.md:34"} {
		if !strings.Contains(text, want) {
			t.Fatalf("license decision brief missing %q", want)
		}
	}
}

func TestRemainingTodoAuditCoversUncheckedTodos(t *testing.T) {
	root := filepath.Join("..", "..")
	todoData, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	auditData, err := os.ReadFile(filepath.Join(root, "docs", "remaining-todo-audit.md"))
	if err != nil {
		t.Fatalf("read remaining audit: %v", err)
	}
	audit := string(auditData)
	for _, line := range strings.Split(string(todoData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		if !strings.Contains(audit, item) {
			t.Fatalf("remaining TODO %q not covered by docs/remaining-todo-audit.md", item)
		}
	}
}
