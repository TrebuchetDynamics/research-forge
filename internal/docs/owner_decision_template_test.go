package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoFyneDependencyAfterWebGUIRescope(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(data), "fyne.io/fyne") {
		t.Fatalf("go.mod contains Fyne dependency after ADR 0006 re-scoped the primary UI to a local web GUI")
	}
}

func TestLicenseTODOIdentifiesRequiredOwnerInputs(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	var licenseLine string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "Add license after owner decision") {
			licenseLine = line
			break
		}
	}
	if licenseLine == "" {
		t.Fatalf("missing license TODO")
	}
	for _, want := range []string{"SPDX", "copyright holder", "issue #1"} {
		if !strings.Contains(licenseLine, want) {
			t.Fatalf("license TODO missing %q: %s", want, licenseLine)
		}
	}
}

func TestNoLicenseFileUntilOwnerDecision(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "LICENSE")); err == nil {
		t.Fatalf("LICENSE exists but TODO.md still says license requires owner decision")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat LICENSE: %v", err)
	}
}

func TestProjectLicenseIssueDocumentsRoutingMetadata(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "decisions", "project_license_issue.md"))
	if err != nil {
		t.Fatalf("read project license issue draft: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Recommended issue routing", "Labels", "decision", "blocked", "owner-input-needed", "Milestone", "Owner decisions"} {
		if !strings.Contains(text, want) {
			t.Fatalf("project license issue draft missing routing metadata %q", want)
		}
	}
}

func TestLicenseDecisionBriefCoversRemainingLicenseTODO(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "license-decision.md"))
	if err != nil {
		t.Fatalf("read license decision brief: %v", err)
	}
	text := string(data)
	for _, want := range []string{"No public license yet", "SPDX", "MIT", "Apache-2.0", "GPL-3.0", "AGPL-3.0", "copyright holder", "Add `LICENSE`", "Update `README.md`", "Mark `TODO.md` license item complete"} {
		if !strings.Contains(text, want) {
			t.Fatalf("license decision brief missing %q", want)
		}
	}
}

func TestWebUISkillReferencesADRAndPlan(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "skills", "research-forge-web-ui-tdd", "SKILL.md"))
	if err != nil {
		t.Fatalf("read web UI skill: %v", err)
	}
	text := string(data)
	for _, want := range []string{"ADR 0006", "docs/web-gui-plan.md", "Go + HTMX", "HTMX", "local research cockpit", "CLI-generated"} {
		if !strings.Contains(text, want) {
			t.Fatalf("web UI skill missing %q", want)
		}
	}
}

func TestWebGUIPlanCoversRemainingWebGUITODOs(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "web-gui-plan.md"))
	if err != nil {
		t.Fatalf("read local web GUI plan: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Add the Go + HTMX web workspace", "Go + HTMX", "Go HTMLX", "local research cockpit", "project create/open", "search", "library", "OSS", "view models", "papers", "meta-analysis", "PRISMA"} {
		if !strings.Contains(text, want) {
			t.Fatalf("local web GUI plan missing %q", want)
		}
	}
}

func TestADRIndexIncludesWebGUIDeferralDecision(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "adr", "README.md"))
	if err != nil {
		t.Fatalf("read ADR index: %v", err)
	}
	text := string(data)
	for _, want := range []string{"ADR 0006", "Rescope Fyne desktop delivery to a local web GUI", "0006-rescope-fyne-desktop-to-local-web-gui.md"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ADR index missing %q", want)
		}
	}
}

func TestRoadmapDocumentsDecisionGatedWebGUIScope(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "ROADMAP.md"))
	if err != nil {
		t.Fatalf("read ROADMAP: %v", err)
	}
	text := string(data)
	for _, want := range []string{"rforge decisions", "docs/owner-decisions.md", "docs/remaining-todo-audit.md", "ADR 0006", "Dependency-free view models", "make license-decision-live-audit", "make license-decision-approval-gate", "TODO.md:34", "approved:true"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ROADMAP missing %q", want)
		}
	}
}

func TestContributingDocumentsDecisionGatedTODOWorkflow(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "CONTRIBUTING.md"))
	if err != nil {
		t.Fatalf("read CONTRIBUTING: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Decision-gated TODOs", "make todo-audit", "Owner decision issue", "make decision-issues", "SPDX identifier", "copyright holder", "approver", "approval date", "make license-decision-live-audit", "make license-decision-approval-gate", "rforge decisions --check TODO.md", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md"} {
		if !strings.Contains(text, want) {
			t.Fatalf("CONTRIBUTING missing %q", want)
		}
	}
}

func TestReadmeLinksLicenseDecisionTracking(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(data)
	for _, want := range []string{"No license has been selected yet", "issue #1", "docs/owner-decisions.md", "rforge decisions", "license choice", "copyright holder string", "approver", "approval date", "make license-decision-live-audit", "make license-decision-approval-gate", "Decision-gated scope", "issue #2", "ADR 0006", "make todo-audit", "make todo-completion-audit", "make decisions-markdown"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README license section missing %q", want)
		}
	}
}

func TestMakefileIncludesDecisionMarkdownAudit(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	for _, want := range []string{"decisions-markdown", "decisions --markdown"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile missing %q", want)
		}
	}
}

func TestMakefileIncludesWebGUISmokeTarget(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	for _, want := range []string{"web-gui-smoke", "go test ./internal/webui"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile missing web GUI smoke evidence %q", want)
		}
	}
	if strings.Contains(text, "fyne-smoke") {
		t.Fatalf("Makefile still exposes obsolete fyne-smoke target")
	}
}

func TestMakefileIncludesDecisionIssueScaffolds(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	for _, want := range []string{"decision-issues", "--issue-body project_license"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile missing %q", want)
		}
	}
}

func TestMakefileIncludesTodoCompletionAuditTarget(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	for _, want := range []string{"todo-completion-audit", "--completion-audit TODO.md docs/todo-completion-audit.md", "docs/todo-completion-audit.md"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile missing TODO completion audit target content %q:\n%s", want, text)
		}
	}
}

func TestMakeCheckIncludesTodoDecisionAudit(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "check: test vet todo-completion-audit") || !strings.Contains(text, "decisions --check TODO.md") {
		t.Fatalf("Makefile check target does not include TODO decision audit:\n%s", text)
	}
}

func TestCIEnforcesTodoDecisionAudit(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "TODO decision audit") || !strings.Contains(text, "make todo-completion-audit") {
		t.Fatalf("CI workflow does not enforce TODO decision audit:\n%s", text)
	}
}

func TestPullRequestTemplateRequiresDecisionLinksForDecisionGatedTODOs(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".github", "PULL_REQUEST_TEMPLATE.md"))
	if err != nil {
		t.Fatalf("read PR template: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Decision-gated TODOs", "Owner decision issue linked", "SPDX identifier", "copyright holder", "approver", "approval date", "make license-decision-live-audit", "approved", "owner_response_required_fields", "license_owner_approval_absent_verified", "license_owner_response_fields_verified", "rforge decisions --check TODO.md", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md", "completion_blocked", "blocked_decisions", "blocked_decision_ids"} {
		if !strings.Contains(text, want) {
			t.Fatalf("PR template missing %q", want)
		}
	}
}

func TestDecisionResolutionChecklistDocumentsApprovalWorkflow(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "decision-resolution-checklist.md"))
	if err != nil {
		t.Fatalf("read decision checklist: %v", err)
	}
	text := string(data)
	for _, want := range []string{"make todo-audit", "reuse an existing open issue", "Owner decision issue", "SPDX", "copyright holder", "issue_labels", "milestone", "options_considered", "owner_response_required_fields", "license_owner_response_fields_verified", "license_options_verified", "license_issue_routing_verified", "make license-decision-live-audit", "gh issue view 1", "non-placeholder", "Update `TODO.md`", "make check", "rforge decisions --check TODO.md", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md", "completion_blocked", "blocked_decisions"} {
		if !strings.Contains(text, want) {
			t.Fatalf("decision checklist missing %q", want)
		}
	}
}

func TestOwnerDecisionIssueTemplateRequiresOwnerInputs(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".github", "ISSUE_TEMPLATE", "owner_decision.yml"))
	if err != nil {
		t.Fatalf("read owner decision template: %v", err)
	}
	text := string(data)
	start := strings.Index(text, "id: owner_inputs")
	if start < 0 {
		t.Fatalf("owner decision template missing owner_inputs field")
	}
	section := text[start:]
	if next := strings.Index(section[1:], "\n  - type:"); next >= 0 {
		section = section[:next+1]
	}
	for _, want := range []string{"Owner inputs needed", "SPDX identifier", "copyright holder", "required: true"} {
		if !strings.Contains(section, want) {
			t.Fatalf("owner_inputs section missing %q:\n%s", want, section)
		}
	}
}

func TestOwnerDecisionIssueTemplateRequiresOwnerResponseFields(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, ".github", "ISSUE_TEMPLATE", "owner_decision.yml"))
	if err != nil {
		t.Fatalf("read owner decision template: %v", err)
	}
	text := string(data)
	start := strings.Index(text, "id: owner_response_required_fields")
	if start < 0 {
		t.Fatalf("owner decision template missing owner_response_required_fields field")
	}
	section := text[start:]
	if next := strings.Index(section[1:], "\n  - type:"); next >= 0 {
		section = section[:next+1]
	}
	for _, want := range []string{"Required owner response fields", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date", "required: true"} {
		if !strings.Contains(section, want) {
			t.Fatalf("owner_response_required_fields section missing %q:\n%s", want, section)
		}
	}
}

func TestOwnerDecisionIssueTemplateSupportsRemainingDecisionIDs(t *testing.T) {
	root := filepath.Join("..", "..")
	templateData, err := os.ReadFile(filepath.Join(root, ".github", "ISSUE_TEMPLATE", "owner_decision.yml"))
	if err != nil {
		t.Fatalf("read owner decision template: %v", err)
	}
	decisionsData, err := os.ReadFile(filepath.Join(root, "docs", "owner-decisions.md"))
	if err != nil {
		t.Fatalf("read owner decisions doc: %v", err)
	}
	template := string(templateData)
	for _, want := range []string{"Decision ID", "Blocked TODO items", "Options considered", "Owner inputs needed", "SPDX identifier", "copyright holder", "Implementation steps after approval"} {
		if !strings.Contains(template, want) {
			t.Fatalf("owner decision template missing %q", want)
		}
	}
	for _, want := range []string{"project_license", "owner_decision_required", "Owner decision: project_license (SPDX, copyright holder, approver, date required)", "issue_labels", "milestone", "options_considered", "owner_response_required_fields", "license_owner_response_fields_verified", "license_options_verified", "license_issue_routing_verified", "web_gui_stack_scope", "complete", "Go + HTMX", "make decisions", "make todo-completion-audit", "make license-decision-approval-gate", "approved:true", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md", "decisions/project_license_issue.md", "decisions/web_gui_stack_scope_issue.md"} {
		if !strings.Contains(string(decisionsData), want) {
			t.Fatalf("owner decisions doc missing %q", want)
		}
	}
	drafts := map[string][]string{
		"project_license_issue.md": {
			"Decision ID",
			"project_license",
			"owner_decision_required",
			"https://github.com/TrebuchetDynamics/research-forge/issues/1",
			"Owner decision: project_license (SPDX, copyright holder, approver, date required)",
			"Blocked TODO items",
			"TODO.md:34",
			"Add license after owner decision",
			"Options considered",
			"Owner inputs needed",
			"copyright holder string",
			"Owner response template",
			"Approved by",
			"Approval date",
			"Implementation steps after approval",
			"make license-decision-approval-gate",
			"make check",
			"rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md",
		},
		"web_gui_stack_scope_issue.md": {
			"Decision ID",
			"web_gui_stack_scope",
			"complete",
			"https://github.com/TrebuchetDynamics/research-forge/issues/2",
			"Completed TODO items",
			"View CLI-generated papers, meta-analysis outputs, PRISMA/citation diagrams, and report artifacts in the web GUI",
			"Go + HTMX",
			"internal/webui",
			"make check",
			"rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md",
		},
	}
	for draft, required := range drafts {
		data, err := os.ReadFile(filepath.Join(root, "docs", "decisions", draft))
		if err != nil {
			t.Fatalf("read decision draft %s: %v", draft, err)
		}
		for _, want := range required {
			if !strings.Contains(string(data), want) {
				t.Fatalf("decision draft %s missing %q", draft, want)
			}
		}
	}
}
