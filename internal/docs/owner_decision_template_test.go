package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoFyneDependencyUntilBuildDecision(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(data), "fyne.io/fyne") {
		t.Fatalf("go.mod contains Fyne dependency while ADR 0005 defers build decision")
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

func TestLicenseDecisionBriefCoversRemainingLicenseTODO(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "license-decision.md"))
	if err != nil {
		t.Fatalf("read license decision brief: %v", err)
	}
	text := string(data)
	for _, want := range []string{"No public license yet", "copyright holder", "Add `LICENSE`", "Update `README.md`", "Mark `TODO.md` license item complete"} {
		if !strings.Contains(text, want) {
			t.Fatalf("license decision brief missing %q", want)
		}
	}
}

func TestFyneDesktopPlanCoversRemainingFyneTODOs(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "fyne-desktop-plan.md"))
	if err != nil {
		t.Fatalf("read Fyne desktop plan: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Add the Fyne dependency", "project create/open", "search", "library", "OSS", "view models"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Fyne desktop plan missing %q", want)
		}
	}
}

func TestADRIndexIncludesFyneDeferralDecision(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "adr", "README.md"))
	if err != nil {
		t.Fatalf("read ADR index: %v", err)
	}
	text := string(data)
	for _, want := range []string{"ADR 0005", "Defer Fyne dependency", "0005-defer-fyne-dependency-until-desktop-build-scope-is-owned.md"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ADR index missing %q", want)
		}
	}
}

func TestRoadmapDocumentsDecisionGatedFyneScope(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "ROADMAP.md"))
	if err != nil {
		t.Fatalf("read ROADMAP: %v", err)
	}
	text := string(data)
	for _, want := range []string{"rforge decisions", "docs/owner-decisions.md", "docs/remaining-todo-audit.md", "ADR 0005", "dependency-free view models"} {
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
	for _, want := range []string{"Decision-gated TODOs", "make todo-audit", "Owner decision issue", "make decision-issues", "rforge decisions --check TODO.md"} {
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
	for _, want := range []string{"No license has been selected yet", "docs/owner-decisions.md", "rforge decisions", "Decision-gated scope", "ADR 0005", "make todo-audit", "make decisions-markdown"} {
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

func TestMakefileIncludesDecisionIssueScaffolds(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(data)
	for _, want := range []string{"decision-issues", "--issue-body project_license", "--issue-body fyne_desktop_build_scope"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile missing %q", want)
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
	if !strings.Contains(text, "check: test vet todo-audit") || !strings.Contains(text, "decisions --check TODO.md") {
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
	if !strings.Contains(text, "TODO decision audit") || !strings.Contains(text, "make todo-audit") {
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
	for _, want := range []string{"Decision-gated TODOs", "Owner decision issue linked", "rforge decisions --check TODO.md"} {
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
	for _, want := range []string{"make todo-audit", "Owner decision issue", "Update `TODO.md`", "make check", "rforge decisions --check TODO.md"} {
		if !strings.Contains(text, want) {
			t.Fatalf("decision checklist missing %q", want)
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
	for _, want := range []string{"Decision ID", "Blocked TODO items", "Options considered", "Implementation steps after approval"} {
		if !strings.Contains(template, want) {
			t.Fatalf("owner decision template missing %q", want)
		}
	}
	for _, want := range []string{"project_license", "fyne_desktop_build_scope", "make decisions", "decisions/project_license_issue.md", "decisions/fyne_desktop_build_scope_issue.md"} {
		if !strings.Contains(string(decisionsData), want) {
			t.Fatalf("owner decisions doc missing %q", want)
		}
	}
	drafts := map[string][]string{
		"project_license_issue.md": {
			"Decision ID",
			"project_license",
			"Blocked TODO items",
			"Add license after owner decision",
			"Options considered",
			"Implementation steps after approval",
		},
		"fyne_desktop_build_scope_issue.md": {
			"Decision ID",
			"fyne_desktop_build_scope",
			"Blocked TODO items",
			"Add Fyne dependency after build decision",
			"Add Fyne search screen",
			"Add Fyne library screen",
			"Create/open a research project from the Fyne UI",
			"View OSS repository studies in Fyne",
			"Options considered",
			"Implementation steps after approval",
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
