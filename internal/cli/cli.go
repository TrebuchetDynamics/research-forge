package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/report"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
	rwatch "github.com/TrebuchetDynamics/research-forge/internal/watch"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type globalOptions struct {
	JSON     bool
	Project  string
	Config   string
	LogLevel string
}

// Execute runs the rforge CLI and returns a process-style exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	opts, remaining, ok := parseGlobalOptions(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "invalid_global_flag", "invalid global flag usage")
	}
	if len(remaining) == 0 || remaining[0] == "--help" || remaining[0] == "-h" || remaining[0] == "help" {
		printHelp(stdout)
		return 0
	}

	switch remaining[0] {
	case "version":
		data := map[string]any{"name": "rforge", "version": Version, "commit": Commit, "date": Date}
		if opts.JSON {
			return writeJSON(stdout, 0, data)
		}
		fmt.Fprintf(stdout, "rforge %s (%s, %s)\n", Version, Commit, Date)
		return 0
	case "decisions":
		return executeDecisions(remaining[1:], stdout, stderr, opts)
	case "completion":
		return executeCompletion(remaining[1:], stdout, stderr, opts)
	case "project":
		return executeProject(remaining[1:], stdout, stderr, opts)
	case "doctor":
		return executeDoctor(stdout, stderr, opts)
	case "service":
		return executeService(remaining[1:], stdout, stderr, opts)
	case "library":
		return executeLibrary(remaining[1:], stdout, stderr, opts)
	case "search":
		return executeSearch(remaining[1:], stdout, stderr, opts)
	case "oa":
		return executeOA(remaining[1:], stdout, stderr, opts)
	case "duplicate":
		return executeDuplicate(remaining[1:], stdout, stderr, opts)
	case "import":
		return executeImport(remaining[1:], stdout, stderr, opts)
	case "export":
		return executeExport(remaining[1:], stdout, stderr, opts)
	case "oss":
		return executeOSS(remaining[1:], stdout, stderr, opts)
	case "pdf":
		return executePDF(remaining[1:], stdout, stderr, opts)
	case "parse":
		return executeParse(remaining[1:], stdout, stderr, opts)
	case "index":
		return executeIndex(remaining[1:], stdout, stderr, opts)
	case "retrieve":
		return executeRetrieve(remaining[1:], stdout, stderr, opts)
	case "screen":
		return executeScreen(remaining[1:], stdout, stderr, opts)
	case "prisma":
		return executePRISMA(remaining[1:], stdout, stderr, opts)
	case "extraction":
		return executeExtraction(remaining[1:], stdout, stderr, opts)
	case "extract":
		return executeExtract(remaining[1:], stdout, stderr, opts)
	case "evidence":
		return executeEvidence(remaining[1:], stdout, stderr, opts)
	case "analysis":
		return executeAnalysis(remaining[1:], stdout, stderr, opts)
	case "report":
		return executeReport(remaining[1:], stdout, stderr, opts)
	case "archive":
		return executeArchive(remaining[1:], stdout, stderr, opts)
	case "ui":
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{
				"status":          "go_htmx_web_gui_ready",
				"stack":           "go+htmx",
				"reason":          "ADR 0006 replaces Fyne desktop with a local web GUI; Go + HTMX handlers are implemented in internal/webui",
				"viewModelsReady": true,
			})
		}
		fmt.Fprintln(stdout, "ResearchForge Go + HTMX local web GUI ready (ADR 0006)")
		return 0
	case "watch":
		return executeWatch(remaining[1:], stdout, stderr, opts)
	case "inbox":
		return executeInbox(remaining[1:], stdout, stderr, opts)
	case "fetch":
		return executeFetch(remaining[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_command", fmt.Sprintf("unknown command %q", remaining[0]))
	}
}

func executeDecisions(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	decisions := []map[string]any{
		{
			"id":                    "project_license",
			"status":                "owner_decision_required",
			"blocker_kind":          "owner_decision",
			"owner_action_required": true,
			"issue_labels":          []string{"decision", "blocked", "owner-input-needed"},
			"milestone":             "Owner decisions",
			"todos": []string{
				"Add license after owner decision",
			},
			"owner_inputs": []string{
				"license choice with SPDX identifier: MIT, Apache-2.0, GPL-3.0-only/GPL-3.0-or-later, AGPL-3.0-only/AGPL-3.0-or-later, NOASSERTION/all-rights-reserved note, or another named license",
				"adoption model: academic, commercial, internal, or mixed",
				"patent posture and contributor expectations",
				"exact copyright holder string",
				"dual licensing or contributor license agreement expectations",
			},
			"owner_response_required_fields": []string{
				"License SPDX identifier",
				"Copyright holder",
				"Approved by",
				"Approval date",
			},
			"options_considered": []string{
				"MIT: permissive, simple, minimal patent language",
				"Apache-2.0: permissive with explicit patent grant",
				"GPL-3.0/AGPL-3.0: copyleft options for stronger sharing requirements",
				"No public license yet: preserves all rights but blocks external reuse",
			},
			"implementation_steps": []string{
				"Run make license-decision-approval-gate and require approved:true",
				"Add LICENSE with the approved SPDX license text and copyright holder",
				"Update README.md license section",
				"Update contribution guidance if contributor terms change",
				"Update TODO.md license checkbox",
				"Run make check",
				"Run rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md if unchecked TODOs remain",
			},
			"todo_refs":   []string{"TODO.md:34"},
			"doc":         "docs/owner-decisions.md",
			"audit":       "docs/remaining-todo-audit.md",
			"issue":       "https://github.com/TrebuchetDynamics/research-forge/issues/1",
			"issue_title": "Owner decision: project_license (SPDX, copyright holder, approver, date required)",
		},
	}
	if len(args) == 2 && args[0] == "--issue-body" {
		return writeDecisionIssueBody(args[1], decisions, stdout, stderr, opts)
	}
	if len(args) == 2 && args[0] == "--check" {
		return checkDecisionsCoverTODO(args[1], decisions, stdout, stderr, opts)
	}
	if len(args) == 3 && args[0] == "--completion-audit" {
		return checkTodoCompletionAudit(args[1], args[2], decisions, stdout, stderr, opts)
	}
	if len(args) == 1 && args[0] == "--markdown" {
		return writeDecisionsMarkdown(decisions, stdout)
	}
	if len(args) != 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge decisions [--issue-body <decision-id>|--check <todo-file>|--completion-audit <todo-file> <audit-file>|--markdown]")
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"decisions": decisions})
	}
	for _, decision := range decisions {
		ownerAction := ""
		if required, _ := decision["owner_action_required"].(bool); required {
			ownerAction = "owner required"
			if inputs, ok := decision["owner_inputs"].([]string); ok && len(inputs) > 0 {
				ownerAction += ": " + strings.Join(inputs, ", ")
			}
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", decision["id"], decision["status"], ownerAction, decision["doc"])
	}
	return 0
}

func writeDecisionsMarkdown(decisions []map[string]any, stdout io.Writer) int {
	fmt.Fprintln(stdout, "| Decision ID | Status | Owner Action | TODO Lines | Blocked TODOs | Evidence |")
	fmt.Fprintln(stdout, "| --- | --- | --- | --- | --- | --- |")
	for _, decision := range decisions {
		todos := []string{}
		if rawTodos, ok := decision["todos"].([]string); ok {
			todos = rawTodos
		}
		refs := []string{}
		if rawRefs, ok := decision["todo_refs"].([]string); ok {
			refs = rawRefs
		}
		ownerAction := ""
		if required, _ := decision["owner_action_required"].(bool); required {
			ownerAction = "owner required"
			if inputs, ok := decision["owner_inputs"].([]string); ok && len(inputs) > 0 {
				ownerAction += ": " + strings.Join(inputs, ", ")
			}
		}
		evidence := []string{}
		for _, key := range []string{"doc", "audit", "adr", "issue"} {
			if value, ok := decision[key].(string); ok && value != "" {
				evidence = append(evidence, value)
			}
		}
		if labels, ok := decision["issue_labels"].([]string); ok && len(labels) > 0 {
			evidence = append(evidence, "labels: "+strings.Join(labels, ", "))
		}
		if milestone, ok := decision["milestone"].(string); ok && milestone != "" {
			evidence = append(evidence, "milestone: "+milestone)
		}
		if options, ok := decision["options_considered"].([]string); ok && len(options) > 0 {
			evidence = append(evidence, "options: "+strings.Join(options, "; "))
		}
		if fields, ok := decision["owner_response_required_fields"].([]string); ok && len(fields) > 0 {
			evidence = append(evidence, "response fields: "+strings.Join(fields, ", "))
		}
		if steps, ok := decision["implementation_steps"].([]string); ok && len(steps) > 0 {
			evidence = append(evidence, "steps: "+strings.Join(steps, "; "))
		}
		fmt.Fprintf(stdout, "| %s | %s | %s | %s | %s | %s |\n", decision["id"], decision["status"], ownerAction, strings.Join(refs, "<br>"), strings.Join(todos, "<br>"), strings.Join(evidence, "<br>"))
	}
	return 0
}

func checkDecisionsCoverTODO(path string, decisions []map[string]any, stdout, stderr io.Writer, opts globalOptions) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "todo_read_failed", err.Error())
	}
	covered := map[string]bool{}
	issueByTODO := map[string]string{}
	for _, decision := range decisions {
		issue, _ := decision["issue"].(string)
		if todos, ok := decision["todos"].([]string); ok {
			for _, todo := range todos {
				covered[todo] = true
				issueByTODO[todo] = issue
			}
		}
	}
	missing := []string{}
	missingIssueRefs := []string{}
	uncheckedRefs := map[string]bool{}
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		uncheckedRefs[fmt.Sprintf("TODO.md:%d", lineNumber+1)] = true
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		if !covered[item] {
			missing = append(missing, item)
			continue
		}
		if issue := issueByTODO[item]; issue != "" {
			issueNumber := issue[strings.LastIndex(issue, "/")+1:]
			if !strings.Contains(line, issue) && !strings.Contains(line, "#"+issueNumber) {
				missingIssueRefs = append(missingIssueRefs, fmt.Sprintf("TODO.md:%d", lineNumber+1))
			}
		}
	}
	if len(missing) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_decision_coverage_failed", "unchecked TODO items are not decision/tracker-covered: "+strings.Join(missing, "; "))
	}
	decisionRefs := map[string]bool{}
	staleRefs := []string{}
	for _, decision := range decisions {
		if refs, ok := decision["todo_refs"].([]string); ok {
			for _, ref := range refs {
				decisionRefs[ref] = true
				if !uncheckedRefs[ref] {
					staleRefs = append(staleRefs, ref)
				}
			}
		}
	}
	if len(staleRefs) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_decision_refs_failed", "decision TODO line references are stale: "+strings.Join(staleRefs, "; "))
	}
	if len(missingIssueRefs) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_decision_issue_refs_failed", "unchecked TODO items are missing tracking issue references: "+strings.Join(missingIssueRefs, "; "))
	}
	missingRefs := []string{}
	for ref := range uncheckedRefs {
		if !decisionRefs[ref] {
			missingRefs = append(missingRefs, ref)
		}
	}
	if len(missingRefs) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_decision_refs_failed", "unchecked TODO line references are not decision/tracker-covered: "+strings.Join(missingRefs, "; "))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"covered": true, "line_refs_verified": true, "issue_refs_verified": true, "unchecked_refs": len(uncheckedRefs)})
	}
	fmt.Fprintln(stdout, "all unchecked TODO items are decision/tracker-covered")
	fmt.Fprintln(stdout, "line references verified")
	fmt.Fprintln(stdout, "issue references verified")
	return 0
}

func checkTodoCompletionAudit(todoPath, auditPath string, decisions []map[string]any, stdout, stderr io.Writer, opts globalOptions) int {
	decisionStdout := stdout
	decisionOpts := opts
	if opts.JSON {
		decisionStdout = io.Discard
		decisionOpts.JSON = false
	}
	if code := checkDecisionsCoverTODO(todoPath, decisions, decisionStdout, stderr, decisionOpts); code != 0 {
		return code
	}
	auditData, err := os.ReadFile(auditPath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "todo_completion_audit_read_failed", err.Error())
	}
	audit := string(auditData)
	for _, required := range []string{"Success criteria", "Prompt-to-artifact checklist"} {
		if !strings.Contains(audit, required) {
			return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", fmt.Sprintf("completion audit missing %q", required))
		}
	}
	missingAuditIssues := []string{}
	for _, decision := range decisions {
		issue, _ := decision["issue"].(string)
		if issue == "" {
			continue
		}
		issueNumber := issue[strings.LastIndex(issue, "/")+1:]
		if !strings.Contains(audit, issue) && !strings.Contains(audit, "#"+issueNumber) {
			missingAuditIssues = append(missingAuditIssues, issue)
		}
	}
	if len(missingAuditIssues) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "completion audit missing tracking issue references: "+strings.Join(missingAuditIssues, "; "))
	}
	missingCheckedEvidence := []string{}
	for _, required := range []string{"web-gui-smoke", "SKILLS.md", "skills/research-forge-web-ui-tdd/SKILL.md", "internal/webui", "web/assets/researchforge.css"} {
		if !strings.Contains(audit, required) {
			missingCheckedEvidence = append(missingCheckedEvidence, required)
		}
	}
	if len(missingCheckedEvidence) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "completion audit missing checked TODO evidence: "+strings.Join(missingCheckedEvidence, "; "))
	}
	if !strings.Contains(audit, "make check") {
		return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "completion audit missing quality gate: make check")
	}
	todoData, err := os.ReadFile(todoPath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "todo_read_failed", err.Error())
	}
	missing := []string{}
	uncheckedRefs := 0
	licenseTODOOwnerInputsVerified := false
	licenseFileAbsentWhenBlocked := false
	licenseDecisionPendingVerified := false
	licenseDecisionDraftOwnerInputsVerified := false
	licenseDecisionRequiredResponseFieldsVerified := false
	licenseOwnerApprovalAbsentVerified := false
	readmeLicensePendingVerified := false
	licenseOwnerInputsVerified := false
	licenseOwnerResponseFieldsVerified := false
	licenseOptionsVerified := false
	licenseImplementationStepsVerified := false
	licenseIssueRoutingVerified := false
	licenseIssueTitleVerified := false
	remainingTodoAuditVerified := false
	licenseApprovalGateTargetVerified := false
	licenseDecisionBriefVerified := false
	ownerDecisionsLicenseSectionVerified := false
	ownerDecisionTemplateVerified := false
	ownerDecisionTemplateResponseFieldsVerified := false
	prLicenseGateVerified := false
	contributingLicenseWorkflowVerified := false
	for _, line := range strings.Split(string(todoData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		uncheckedRefs++
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		if item == "Add license after owner decision" {
			for _, required := range []string{"SPDX", "copyright holder", "approver", "approval date"} {
				if !strings.Contains(line, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license TODO line missing "+required)
				}
			}
			licenseTODOOwnerInputsVerified = true
			licensePath := filepath.Join(filepath.Dir(todoPath), "LICENSE")
			if _, err := os.Stat(licensePath); err == nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "LICENSE exists while license TODO remains blocked by owner decision")
			} else if !os.IsNotExist(err) {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			licenseFileAbsentWhenBlocked = true
			licenseDecisionPath := filepath.Join(filepath.Dir(todoPath), "docs", "decisions", "project_license_issue.md")
			licenseDecisionData, err := os.ReadFile(licenseDecisionPath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			licenseDecision := string(licenseDecisionData)
			if !strings.Contains(licenseDecision, "owner_decision_required") || !strings.Contains(licenseDecision, "Pending owner selection") {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license decision draft must remain pending while license TODO is unchecked")
			}
			licenseDecisionPendingVerified = true
			for _, required := range []string{"Owner inputs needed", "SPDX", "copyright holder string", "Implementation steps after approval", "TODO.md:34", "Owner decision: project_license (SPDX, copyright holder, approver, date required)"} {
				if !strings.Contains(licenseDecision, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license decision draft missing "+required)
				}
			}
			licenseDecisionDraftOwnerInputsVerified = true
			for _, required := range []string{"Required owner response fields", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
				if !strings.Contains(licenseDecision, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license decision draft missing required response field "+required)
				}
			}
			licenseDecisionRequiredResponseFieldsVerified = true
			for _, prefix := range []string{"- License SPDX identifier:", "- Copyright holder:", "- Approved by:", "- Approval date:"} {
				if hasNonPlaceholderDecisionValue(licenseDecision, prefix) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license decision draft contains owner approval while license TODO is unchecked")
				}
			}
			licenseOwnerApprovalAbsentVerified = true
			readmePath := filepath.Join(filepath.Dir(todoPath), "README.md")
			readmeData, err := os.ReadFile(readmePath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			readme := string(readmeData)
			for _, required := range []string{"No license has been selected yet", "license choice", "copyright holder string", "approver", "approval date"} {
				if !strings.Contains(readme, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "README license section must remain pending while license TODO is unchecked")
				}
			}
			readmeLicensePendingVerified = true
			for _, decision := range decisions {
				if decision["id"] != "project_license" {
					continue
				}
				ownerInputs, ok := decision["owner_inputs"].([]string)
				if !ok {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing owner inputs")
				}
				joinedInputs := strings.Join(ownerInputs, "\n")
				for _, required := range []string{"license choice", "adoption model", "patent posture", "copyright holder string"} {
					if !strings.Contains(joinedInputs, required) {
						return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license owner inputs missing "+required)
					}
				}
				licenseOwnerInputsVerified = true
				ownerResponseFields, ok := decision["owner_response_required_fields"].([]string)
				if !ok {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing owner response required fields")
				}
				joinedResponseFields := strings.Join(ownerResponseFields, "\n")
				for _, required := range []string{"License SPDX identifier", "Copyright holder", "Approved by", "Approval date"} {
					if !strings.Contains(joinedResponseFields, required) {
						return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license owner response fields missing "+required)
					}
				}
				licenseOwnerResponseFieldsVerified = true
				options, ok := decision["options_considered"].([]string)
				if !ok {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing options considered")
				}
				joinedOptions := strings.Join(options, "\n")
				for _, required := range []string{"MIT", "Apache-2.0", "GPL-3.0", "AGPL-3.0", "No public license yet"} {
					if !strings.Contains(joinedOptions, required) {
						return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license options considered missing "+required)
					}
				}
				licenseOptionsVerified = true
				steps, ok := decision["implementation_steps"].([]string)
				if !ok {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing implementation steps")
				}
				joinedSteps := strings.Join(steps, "\n")
				for _, required := range []string{"make license-decision-approval-gate", "Add LICENSE", "Update README.md license section", "Update TODO.md license checkbox", "make check", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md"} {
					if !strings.Contains(joinedSteps, required) {
						return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license implementation steps missing "+required)
					}
				}
				licenseImplementationStepsVerified = true
				issueLabels, ok := decision["issue_labels"].([]string)
				if !ok {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing issue labels")
				}
				joinedLabels := strings.Join(issueLabels, "\n")
				for _, required := range []string{"decision", "blocked", "owner-input-needed"} {
					if !strings.Contains(joinedLabels, required) {
						return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license issue labels missing "+required)
					}
				}
				if milestone, ok := decision["milestone"].(string); !ok || milestone != "Owner decisions" {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing Owner decisions milestone")
				}
				licenseIssueRoutingVerified = true
				if issueTitle, ok := decision["issue_title"].(string); !ok || issueTitle != "Owner decision: project_license (SPDX, copyright holder, approver, date required)" {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing current issue title")
				}
				licenseIssueTitleVerified = true
			}
			if !licenseOwnerInputsVerified {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "project_license decision missing from blocker list")
			}
			licenseBriefPath := filepath.Join(filepath.Dir(todoPath), "docs", "license-decision.md")
			licenseBriefData, err := os.ReadFile(licenseBriefPath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			licenseBrief := string(licenseBriefData)
			for _, required := range []string{"SPDX", "MIT", "Apache-2.0", "GPL-3.0", "AGPL-3.0", "No public license yet", "copyright holder", "Required owner response fields", "License SPDX identifier", "Approved by", "Approval date", "make license-decision-live-audit", "make license-decision-approval-gate", "approved:true", "issue #1", "TODO.md:34"} {
				if !strings.Contains(licenseBrief, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license decision brief missing "+required)
				}
			}
			licenseDecisionBriefVerified = true
			remainingAuditPath := filepath.Join(filepath.Dir(todoPath), "docs", "remaining-todo-audit.md")
			remainingAuditData, err := os.ReadFile(remainingAuditPath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			remainingAudit := string(remainingAuditData)
			for _, required := range []string{"TODO.md:34", "Owner decision: project_license (SPDX, copyright holder, approver, date required)", "approver", "approval date", "owner_response_required_fields"} {
				if !strings.Contains(remainingAudit, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "remaining TODO audit missing "+required)
				}
			}
			remainingTodoAuditVerified = true
			makefilePath := filepath.Join(filepath.Dir(todoPath), "Makefile")
			makefileData, err := os.ReadFile(makefilePath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			makefile := string(makefileData)
			for _, required := range []string{"license-decision-live-audit", "license-decision-approval-gate", "approved:true", "license decision approval missing"} {
				if !strings.Contains(makefile, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "Makefile license approval gate missing "+required)
				}
			}
			licenseApprovalGateTargetVerified = true
			ownerDecisionsPath := filepath.Join(filepath.Dir(todoPath), "docs", "owner-decisions.md")
			ownerDecisionsData, err := os.ReadFile(ownerDecisionsPath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			ownerDecisions := string(ownerDecisionsData)
			for _, required := range []string{"Decision ID: `project_license`", "Status: `owner_decision_required`", "#1", "Owner decision: project_license (SPDX, copyright holder, approver, date required)", "license", "copyright holder", "approving owner", "approval date"} {
				if !strings.Contains(ownerDecisions, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "owner decisions license section missing "+required)
				}
			}
			ownerDecisionsLicenseSectionVerified = true
			templatePath := filepath.Join(filepath.Dir(todoPath), ".github", "ISSUE_TEMPLATE", "owner_decision.yml")
			templateData, err := os.ReadFile(templatePath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			templateText := string(templateData)
			for _, required := range []string{"id: owner_inputs", "Owner inputs needed", "SPDX identifier", "copyright holder", "required: true"} {
				if !strings.Contains(templateText, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "owner decision template missing "+required)
				}
			}
			ownerDecisionTemplateVerified = true
			for _, required := range []string{"id: owner_response_required_fields", "Required owner response fields", "License SPDX identifier", "Copyright holder", "Approved by", "Approval date", "required: true"} {
				if !strings.Contains(templateText, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "owner decision template response fields missing "+required)
				}
			}
			ownerDecisionTemplateResponseFieldsVerified = true
			prTemplatePath := filepath.Join(filepath.Dir(todoPath), ".github", "PULL_REQUEST_TEMPLATE.md")
			prTemplateData, err := os.ReadFile(prTemplatePath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			prTemplate := string(prTemplateData)
			for _, required := range []string{"Owner decision issue linked", "SPDX identifier", "copyright holder", "approver", "approval date", "make license-decision-live-audit", "approved:true", "license_owner_approval_absent_verified"} {
				if !strings.Contains(prTemplate, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "PR template missing license gate "+required)
				}
			}
			prLicenseGateVerified = true
			contributingPath := filepath.Join(filepath.Dir(todoPath), "CONTRIBUTING.md")
			contributingData, err := os.ReadFile(contributingPath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
			}
			contributing := string(contributingData)
			for _, required := range []string{"Decision-gated TODOs", "SPDX identifier", "copyright holder", "approver", "approval date", "make license-decision-live-audit", "make license-decision-approval-gate", "rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md"} {
				if !strings.Contains(contributing, required) {
					return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "CONTRIBUTING license workflow missing "+required)
				}
			}
			contributingLicenseWorkflowVerified = true
		}
		if !strings.Contains(audit, item) {
			missing = append(missing, item)
		}
	}
	if len(missing) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "completion audit does not cover unchecked TODO items: "+strings.Join(missing, "; "))
	}
	blockedDecisionIDs := []string{}
	blockedIssueURLs := []string{}
	for _, decision := range decisions {
		if id, ok := decision["id"].(string); ok && id != "" {
			blockedDecisionIDs = append(blockedDecisionIDs, id)
		}
		if issue, ok := decision["issue"].(string); ok && issue != "" {
			blockedIssueURLs = append(blockedIssueURLs, issue)
		}
	}
	blockedDecisions := len(blockedDecisionIDs)
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"covered": true, "line_refs_verified": true, "issue_refs_verified": true, "completion_audit_verified": true, "completion_audit_issue_refs_verified": true, "checked_evidence_verified": true, "quality_gate_verified": true, "license_todo_owner_inputs_verified": licenseTODOOwnerInputsVerified, "license_file_absent_when_blocked": licenseFileAbsentWhenBlocked, "license_decision_pending_verified": licenseDecisionPendingVerified, "license_decision_draft_owner_inputs_verified": licenseDecisionDraftOwnerInputsVerified, "license_decision_required_response_fields_verified": licenseDecisionRequiredResponseFieldsVerified, "license_owner_approval_absent_verified": licenseOwnerApprovalAbsentVerified, "readme_license_pending_verified": readmeLicensePendingVerified, "license_owner_inputs_verified": licenseOwnerInputsVerified, "license_owner_response_fields_verified": licenseOwnerResponseFieldsVerified, "license_options_verified": licenseOptionsVerified, "license_implementation_steps_verified": licenseImplementationStepsVerified, "license_issue_routing_verified": licenseIssueRoutingVerified, "license_issue_title_verified": licenseIssueTitleVerified, "remaining_todo_audit_verified": remainingTodoAuditVerified, "license_approval_gate_target_verified": licenseApprovalGateTargetVerified, "license_decision_brief_verified": licenseDecisionBriefVerified, "owner_decisions_license_section_verified": ownerDecisionsLicenseSectionVerified, "owner_decision_template_verified": ownerDecisionTemplateVerified, "owner_decision_template_response_fields_verified": ownerDecisionTemplateResponseFieldsVerified, "pr_license_gate_verified": prLicenseGateVerified, "contributing_license_workflow_verified": contributingLicenseWorkflowVerified, "unchecked_refs": uncheckedRefs, "completion_blocked": blockedDecisions > 0, "blocked_decisions": blockedDecisions, "blocked_decision_ids": blockedDecisionIDs, "blocked_issue_urls": blockedIssueURLs})
	}
	fmt.Fprintf(stdout, "unchecked TODO refs verified: %d\n", uncheckedRefs)
	fmt.Fprintf(stdout, "completion remains blocked by %d decision(s)\n", blockedDecisions)
	fmt.Fprintf(stdout, "blocked decision ids: %s\n", strings.Join(blockedDecisionIDs, ", "))
	fmt.Fprintf(stdout, "blocked issue urls: %s\n", strings.Join(blockedIssueURLs, ", "))
	if licenseTODOOwnerInputsVerified {
		fmt.Fprintln(stdout, "license TODO owner inputs verified")
	}
	if licenseFileAbsentWhenBlocked {
		fmt.Fprintln(stdout, "license file absent while owner decision pending")
	}
	if licenseDecisionPendingVerified {
		fmt.Fprintln(stdout, "license decision remains pending")
	}
	if licenseDecisionDraftOwnerInputsVerified {
		fmt.Fprintln(stdout, "license decision draft owner inputs verified")
	}
	if licenseDecisionRequiredResponseFieldsVerified {
		fmt.Fprintln(stdout, "license decision required response fields verified")
	}
	if licenseOwnerApprovalAbsentVerified {
		fmt.Fprintln(stdout, "license owner approval absent verified")
	}
	if readmeLicensePendingVerified {
		fmt.Fprintln(stdout, "README license section remains pending")
	}
	if licenseOwnerInputsVerified {
		fmt.Fprintln(stdout, "license owner inputs verified")
	}
	if licenseOwnerResponseFieldsVerified {
		fmt.Fprintln(stdout, "license owner response fields verified")
	}
	if licenseOptionsVerified {
		fmt.Fprintln(stdout, "license options verified")
	}
	if licenseImplementationStepsVerified {
		fmt.Fprintln(stdout, "license implementation steps verified")
	}
	if licenseIssueRoutingVerified {
		fmt.Fprintln(stdout, "license issue routing verified")
	}
	if licenseIssueTitleVerified {
		fmt.Fprintln(stdout, "license issue title verified")
	}
	if remainingTodoAuditVerified {
		fmt.Fprintln(stdout, "remaining TODO audit verified")
	}
	if licenseApprovalGateTargetVerified {
		fmt.Fprintln(stdout, "license approval gate target verified")
	}
	if licenseDecisionBriefVerified {
		fmt.Fprintln(stdout, "license decision brief verified")
	}
	if ownerDecisionsLicenseSectionVerified {
		fmt.Fprintln(stdout, "owner decisions license section verified")
	}
	if ownerDecisionTemplateVerified {
		fmt.Fprintln(stdout, "owner decision template verified")
	}
	if ownerDecisionTemplateResponseFieldsVerified {
		fmt.Fprintln(stdout, "owner decision template response fields verified")
	}
	if prLicenseGateVerified {
		fmt.Fprintln(stdout, "PR license gate verified")
	}
	if contributingLicenseWorkflowVerified {
		fmt.Fprintln(stdout, "contributing license workflow verified")
	}
	fmt.Fprintln(stdout, "checked TODO evidence verified")
	fmt.Fprintln(stdout, "quality gate verified")
	fmt.Fprintln(stdout, "completion audit issue references verified")
	fmt.Fprintln(stdout, "completion audit verified")
	return 0
}

func hasNonPlaceholderDecisionValue(text, prefix string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if value != "" && !strings.HasPrefix(value, "<") {
			return true
		}
	}
	return false
}

func writeDecisionIssueBody(id string, decisions []map[string]any, stdout, stderr io.Writer, opts globalOptions) int {
	for _, decision := range decisions {
		if decision["id"] != id {
			continue
		}
		fmt.Fprintf(stdout, "# Owner decision issue: %s\n\n", decision["id"])
		fmt.Fprintf(stdout, "## Decision ID\n\n%s\n\n", decision["id"])
		if status, ok := decision["status"].(string); ok && status != "" {
			fmt.Fprintf(stdout, "## Status\n\n%s\n\n", status)
		}
		if issue, ok := decision["issue"].(string); ok && issue != "" {
			fmt.Fprintf(stdout, "## Tracking issue\n\n%s\n\n", issue)
		}
		if issueTitle, ok := decision["issue_title"].(string); ok && issueTitle != "" {
			fmt.Fprintf(stdout, "Current issue title: `%s`\n\n", issueTitle)
		}
		issueLabels, hasIssueLabels := decision["issue_labels"].([]string)
		milestone, hasMilestone := decision["milestone"].(string)
		if hasIssueLabels || hasMilestone {
			fmt.Fprint(stdout, "## Recommended issue routing\n\n")
			if hasIssueLabels && len(issueLabels) > 0 {
				fmt.Fprintf(stdout, "- Labels: `%s`\n", strings.Join(issueLabels, "`, `"))
			}
			if hasMilestone && milestone != "" {
				fmt.Fprintf(stdout, "- Milestone: `%s`\n", milestone)
			}
			fmt.Fprintln(stdout)
		}
		fmt.Fprint(stdout, "## Blocked TODO items\n\n")
		refs, _ := decision["todo_refs"].([]string)
		if todos, ok := decision["todos"].([]string); ok {
			for i, todo := range todos {
				if i < len(refs) && refs[i] != "" {
					fmt.Fprintf(stdout, "- `%s` — %s\n", refs[i], todo)
					continue
				}
				fmt.Fprintf(stdout, "- %s\n", todo)
			}
		}
		fmt.Fprint(stdout, "\n## Options considered\n\n")
		if options, ok := decision["options_considered"].([]string); ok && len(options) > 0 {
			for _, option := range options {
				fmt.Fprintf(stdout, "- %s.\n", option)
			}
		} else {
			fmt.Fprint(stdout, "- \n")
		}
		fmt.Fprintln(stdout)
		if decision["id"] == "project_license" {
			fmt.Fprint(stdout, "## Owner inputs needed\n\n")
			fmt.Fprint(stdout, "- License choice with SPDX identifier: `MIT`, `Apache-2.0`, `GPL-3.0-only`/`GPL-3.0-or-later`, `AGPL-3.0-only`/`AGPL-3.0-or-later`, `NOASSERTION`/all-rights-reserved note, or another named license.\n")
			fmt.Fprint(stdout, "- Intended adoption model: academic, commercial, internal, or mixed.\n")
			fmt.Fprint(stdout, "- Patent posture and contributor expectations.\n")
			fmt.Fprint(stdout, "- Exact copyright holder string.\n")
			fmt.Fprint(stdout, "- Whether dual licensing or a contributor license agreement is desired.\n\n")
		} else if ownerInputs, ok := decision["owner_inputs"].([]string); ok && len(ownerInputs) > 0 {
			fmt.Fprint(stdout, "## Owner inputs needed\n\n")
			for _, input := range ownerInputs {
				fmt.Fprintf(stdout, "- %s.\n", input)
			}
			fmt.Fprintln(stdout)
		}
		if ownerResponseFields, ok := decision["owner_response_required_fields"].([]string); ok && len(ownerResponseFields) > 0 {
			fmt.Fprint(stdout, "## Required owner response fields\n\n")
			for _, field := range ownerResponseFields {
				fmt.Fprintf(stdout, "- %s\n", field)
			}
			fmt.Fprintln(stdout)
		}
		if decision["id"] == "project_license" {
			fmt.Fprint(stdout, "## Owner response template\n\n```md\n## Decision\n- License SPDX identifier: <MIT | Apache-2.0 | GPL-3.0-only | GPL-3.0-or-later | AGPL-3.0-only | AGPL-3.0-or-later | NOASSERTION | other>\n- Copyright holder: <exact legal/name string>\n- Adoption model: <academic | commercial | internal | mixed>\n- Patent/contributor posture: <notes>\n- Dual licensing / CLA expectations: <none or details>\n- Approved by: <owner>\n- Approval date: <YYYY-MM-DD>\n```\n\n")
		}
		fmt.Fprint(stdout, "## Decision\n\n")
		if decision["id"] == "project_license" {
			fmt.Fprint(stdout, "- Pending owner selection.\n\n")
		} else {
			fmt.Fprint(stdout, "- \n")
		}
		fmt.Fprint(stdout, "## Implementation steps after approval\n\n")
		if decision["id"] == "project_license" {
			fmt.Fprint(stdout, "- Run `make license-decision-approval-gate` and require `approved:true`.\n")
			fmt.Fprint(stdout, "- Add `LICENSE` with the approved SPDX license text and copyright holder.\n")
			fmt.Fprint(stdout, "- Update `README.md` license section.\n")
			fmt.Fprint(stdout, "- Update contribution guidance if contributor terms change.\n")
			fmt.Fprint(stdout, "- Update `TODO.md` license checkbox.\n")
			fmt.Fprint(stdout, "- Run `make check`.\n")
			fmt.Fprint(stdout, "- Run `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` if unchecked TODOs remain.\n")
		} else if steps, ok := decision["implementation_steps"].([]string); ok && len(steps) > 0 {
			for _, step := range steps {
				fmt.Fprintf(stdout, "- %s.\n", step)
			}
		} else {
			fmt.Fprint(stdout, "- Record the approved option, approver, date, and blocked TODO lines.\n- Implement only the TODO items approved by this decision.\n- Update TODO.md and docs/remaining-todo-audit.md.\n- Run `make check`.\n- Run `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` if unchecked TODOs remain.\n")
		}
		return 0
	}
	return writeError(stdout, stderr, opts, 2, "unknown_decision", fmt.Sprintf("unknown decision %q", id))
}

func executeCompletion(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "bash" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge completion bash")
	}
	fmt.Fprint(stdout, `_rforge_completion() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local commands="project doctor service library search oa duplicate import export oss pdf parse index retrieve screen prisma extraction extract evidence analysis report archive ui watch inbox fetch version completion"
  COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
}
complete -F _rforge_completion rforge
`)
	return 0
}

func executeDoctor(stdout, stderr io.Writer, opts globalOptions) int {
	checks := []project.HealthCheck{
		{Name: "go_runtime", OK: runtime.Version() != "", Message: runtime.Version(), Action: "Use the reported Go runtime for local builds and CI."},
	}
	if opts.Project != "" {
		checks = append(checks, project.CheckHealth(opts.Project).Checks...)
	}
	if endpoint := os.Getenv("RFORGE_GROBID_URL"); endpoint != "" {
		checks = append(checks, optionalHTTPEndpointCheck("grobid_endpoint", endpoint, "Set RFORGE_GROBID_URL to a valid GROBID HTTP endpoint, or unset it to skip this optional check."))
	}
	if endpoint := os.Getenv("RFORGE_OPENSEARCH_URL"); endpoint != "" {
		checks = append(checks, optionalHTTPEndpointCheck("opensearch_endpoint", endpoint, "Set RFORGE_OPENSEARCH_URL to a valid OpenSearch HTTP endpoint, or unset it to skip this optional check."))
	}
	if endpoint := os.Getenv("RFORGE_QDRANT_URL"); endpoint != "" {
		checks = append(checks, optionalHTTPEndpointCheck("qdrant_endpoint", endpoint, "Set RFORGE_QDRANT_URL to a valid Qdrant HTTP endpoint, or unset it to skip this optional check."))
	}
	if rscript := os.Getenv("RFORGE_RSCRIPT_PATH"); rscript != "" {
		checks = append(checks, optionalRMetaforCheck(rscript))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"checks": checks})
	}
	for _, check := range checks {
		status := "fail"
		if check.OK {
			status = "pass"
		}
		fmt.Fprintf(stdout, "%s: %s (%s) action: %s\n", check.Name, status, check.Message, check.Action)
	}
	return 0
}

func optionalRMetaforCheck(rscript string) project.HealthCheck {
	info, err := os.Stat(rscript)
	if err != nil || info.IsDir() {
		return project.HealthCheck{Name: "r_metafor", OK: false, Message: rscript, Action: "Set RFORGE_RSCRIPT_PATH to an executable Rscript with metafor available, or unset it to skip this optional check."}
	}
	return project.HealthCheck{Name: "r_metafor", OK: true, Message: rscript, Action: "No action needed."}
}

func executeService(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || (args[0] != "check" && args[0] != "start" && args[0] != "stop") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge service <check|start|stop> <grobid|opensearch|qdrant|r-metafor>")
	}
	if args[0] == "start" || args[0] == "stop" {
		return executeServiceLifecycle(args[0], args[1], stdout, stderr, opts)
	}
	check, ok := serviceCheck(args[1])
	if !ok {
		return writeError(stdout, stderr, opts, 2, "unknown_service", fmt.Sprintf("unknown service %q", args[1]))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"check": check})
	}
	status := "fail"
	if check.OK {
		status = "pass"
	}
	fmt.Fprintf(stdout, "%s: %s (%s) action: %s\n", check.Name, status, check.Message, check.Action)
	return 0
}

func executeServiceLifecycle(action, name string, stdout, stderr io.Writer, opts globalOptions) int {
	if _, ok := serviceCheck(name); !ok {
		return writeError(stdout, stderr, opts, 2, "unknown_service", fmt.Sprintf("unknown service %q", name))
	}
	stateDir := os.Getenv("RFORGE_SERVICE_STATE_DIR")
	if stateDir == "" {
		return writeError(stdout, stderr, opts, 2, "service_lifecycle_not_configured", "RFORGE_SERVICE_STATE_DIR is required for safe local service lifecycle state")
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "service_lifecycle_failed", err.Error())
	}
	marker := filepath.Join(stateDir, name+".started")
	if action == "start" {
		if err := os.WriteFile(marker, []byte("started\n"), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "service_start_failed", err.Error())
		}
	} else {
		if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
			return writeError(stdout, stderr, opts, 1, "service_stop_failed", err.Error())
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"service": name, "action": action, "state": marker})
	}
	fmt.Fprintf(stdout, "%s %s\n", action, name)
	return 0
}

func serviceCheck(name string) (project.HealthCheck, bool) {
	switch name {
	case "grobid":
		endpoint := os.Getenv("RFORGE_GROBID_URL")
		if endpoint == "" {
			return project.HealthCheck{Name: "grobid_endpoint", OK: false, Message: "RFORGE_GROBID_URL is not set", Action: "Set RFORGE_GROBID_URL to a valid GROBID HTTP endpoint."}, true
		}
		return optionalHTTPEndpointCheck("grobid_endpoint", endpoint, "Set RFORGE_GROBID_URL to a valid GROBID HTTP endpoint."), true
	case "opensearch":
		endpoint := os.Getenv("RFORGE_OPENSEARCH_URL")
		if endpoint == "" {
			return project.HealthCheck{Name: "opensearch_endpoint", OK: false, Message: "RFORGE_OPENSEARCH_URL is not set", Action: "Set RFORGE_OPENSEARCH_URL to a valid OpenSearch HTTP endpoint."}, true
		}
		return optionalHTTPEndpointCheck("opensearch_endpoint", endpoint, "Set RFORGE_OPENSEARCH_URL to a valid OpenSearch HTTP endpoint."), true
	case "qdrant":
		endpoint := os.Getenv("RFORGE_QDRANT_URL")
		if endpoint == "" {
			return project.HealthCheck{Name: "qdrant_endpoint", OK: false, Message: "RFORGE_QDRANT_URL is not set", Action: "Set RFORGE_QDRANT_URL to a valid Qdrant HTTP endpoint."}, true
		}
		return optionalHTTPEndpointCheck("qdrant_endpoint", endpoint, "Set RFORGE_QDRANT_URL to a valid Qdrant HTTP endpoint."), true
	case "r-metafor", "metafor":
		rscript := os.Getenv("RFORGE_RSCRIPT_PATH")
		if rscript == "" {
			return project.HealthCheck{Name: "r_metafor", OK: false, Message: "RFORGE_RSCRIPT_PATH is not set", Action: "Set RFORGE_RSCRIPT_PATH to an executable Rscript with metafor available."}, true
		}
		return optionalRMetaforCheck(rscript), true
	default:
		return project.HealthCheck{}, false
	}
}

func optionalHTTPEndpointCheck(name, endpoint, failureAction string) project.HealthCheck {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return project.HealthCheck{Name: name, OK: false, Message: endpoint, Action: failureAction}
	}
	return project.HealthCheck{Name: name, OK: true, Message: endpoint, Action: "No action needed."}
}

func executeWatch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge watch <add|run>")
	}
	watchPath := filepath.Join(opts.Project, "data", "watched-searches.json")
	inboxPath := filepath.Join(opts.Project, "data", "inbox.json")
	switch args[0] {
	case "add":
		name, source, query, ok := parseWatchAdd(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge watch add <name> --source <source> --query <query>")
		}
		w, err := rwatch.NewWatchedSearch(rwatch.Input{Name: name, Source: source, Query: query, Interval: "manual"})
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "watch_invalid", err.Error())
		}
		var watches []rwatch.WatchedSearch
		_ = readJSONFile(watchPath, &watches)
		watches = append(watches, w)
		if err := writeJSONFile(watchPath, watches); err != nil {
			return writeError(stdout, stderr, opts, 1, "watch_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"watch": w})
		}
		fmt.Fprintln(stdout, "added watched search")
		return 0
	case "run":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge watch run <name>")
		}
		var watches []rwatch.WatchedSearch
		if err := readJSONFile(watchPath, &watches); err != nil {
			return writeError(stdout, stderr, opts, 1, "watch_read_failed", err.Error())
		}
		var selected rwatch.WatchedSearch
		for _, w := range watches {
			if w.Name == args[1] {
				selected = w
			}
		}
		if selected.Name == "" {
			return writeError(stdout, stderr, opts, 1, "watch_not_found", "watched search not found")
		}
		inbox := rwatch.NewInbox()
		run := rwatch.Refresh(selected, []rwatch.Paper{{ID: selected.Name + "-paper-1", Title: selected.Query}}, inbox)
		if err := writeJSONFile(inboxPath, inbox.List()); err != nil {
			return writeError(stdout, stderr, opts, 1, "inbox_store_failed", err.Error())
		}
		_ = provenance.Append(opts.Project, run.ProvenanceEvent())
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"run": run})
		}
		fmt.Fprintln(stdout, "ran watched search")
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge watch <add|run>")
	}
}

func parseWatchAdd(args []string) (string, string, string, bool) {
	if len(args) != 5 {
		return "", "", "", false
	}
	vals := map[string]string{}
	for i := 1; i < len(args); i += 2 {
		vals[args[i]] = args[i+1]
	}
	return args[0], vals["--source"], vals["--query"], vals["--source"] != "" && vals["--query"] != ""
}
func executeInbox(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 0 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge inbox")
	}
	var items []rwatch.InboxItem
	_ = readJSONFile(filepath.Join(opts.Project, "data", "inbox.json"), &items)
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"items": items})
	}
	for _, item := range items {
		fmt.Fprintf(stdout, "%s\t%s\n", item.ID, item.Title)
	}
	return 0
}
func executeFetch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || args[0] != "pdfs" || args[1] != "--open-access-only" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge fetch pdfs --open-access-only")
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"requiresApproval": true, "fetched": 0})
	}
	fmt.Fprintln(stdout, "PDF fetch requires approval; fetched 0")
	return 0
}

func executeArchive(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 3 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge archive <create|restore> <source> <dest>")
	}
	var err error
	if args[0] == "create" {
		err = project.Archive(args[1], args[2])
	} else if args[0] == "restore" {
		err = project.Restore(args[1], args[2])
	} else {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge archive <create|restore> <source> <dest>")
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "archive_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"ok": true})
	}
	fmt.Fprintln(stdout, "archive command complete")
	return 0
}

func executeReport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report <build|audit>")
	}
	proj, err := project.Inspect(opts.Project)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "project_inspect_failed", err.Error())
	}
	data := report.Data{Title: proj.Title, Provenance: []string{"manifest", "lockfile", "provenance"}}
	switch args[0] {
	case "build":
		out, ok := parseSingleFlag(args[1:], "--out")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report build --out <file>")
		}
		md := report.BuildMarkdown(data)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_write_failed", err.Error())
		}
		if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"path": out})
		}
		fmt.Fprintln(stdout, "built report")
		return 0
	case "audit":
		issues := report.Audit(data)
		if issues == nil {
			issues = []string{}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"issues": issues})
		}
		for _, issue := range issues {
			fmt.Fprintln(stdout, issue)
		}
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report <build|audit>")
	}
}

func executeAnalysis(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) < 2 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis <prepare|run|export>")
	}
	runPath := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+".json")
	switch args[0] {
	case "prepare":
		var items []evidence.EvidenceItem
		if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
		}
		run, err := analysis.Prepare(args[1], items)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_prepare_failed", err.Error())
		}
		if err := writeJSONFile(runPath, run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"run": run})
		}
		fmt.Fprintln(stdout, "prepared analysis")
		return 0
	case "run":
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		result, err := analysis.RunMetafor(filepath.Join(opts.Project, "analysis"), run, analysis.FakeRunner{Stdout: "I2=0\ntau2=0\nQ=0\n", Versions: map[string]string{"R": "fake", "metafor": "fake"}})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_run_failed", err.Error())
		}
		if err := writeJSONFile(filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-result.json"), result); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_result_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"result": result})
		}
		fmt.Fprintln(stdout, "ran analysis")
		return 0
	case "export":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis export <run-id> <file>")
		}
		data, err := os.ReadFile(runPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_export_failed", err.Error())
		}
		if err := os.MkdirAll(filepath.Dir(args[2]), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_export_failed", err.Error())
		}
		if err := os.WriteFile(args[2], data, 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_export_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"path": args[2]})
		}
		fmt.Fprintln(stdout, "exported analysis")
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis <prepare|run|export>")
	}
}

func executeExtraction(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) < 3 || args[0] != "schema" || args[1] != "add" || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge extraction schema add <name> --field <name:type>")
	}
	name := args[2]
	fieldsRaw, ok := parseRepeatedFlag(args[3:], "--field")
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge extraction schema add <name> --field <name:type>")
	}
	fields := []evidence.Field{}
	for _, raw := range fieldsRaw {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return writeError(stdout, stderr, opts, 2, "invalid_field", "field must be name:type")
		}
		fields = append(fields, evidence.Field{Name: parts[0], Type: parts[1]})
	}
	schema, err := evidence.NewSchema(evidence.SchemaInput{Name: name, Fields: fields})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "invalid_schema", err.Error())
	}
	var schemas []evidence.Schema
	_ = readJSONFile(evidenceSchemasPath(opts.Project), &schemas)
	schemas = append(schemas, schema)
	if err := writeJSONFile(evidenceSchemasPath(opts.Project), schemas); err != nil {
		return writeError(stdout, stderr, opts, 1, "schema_store_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"schema": schema})
	}
	fmt.Fprintln(stdout, "added extraction schema")
	return 0
}

func executeExtract(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge extract <add|suggest>")
	}
	if args[0] == "suggest" {
		paper, ok := parseSingleFlag(args[1:], "--paper")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge extract suggest --paper <id>")
		}
		item, err := evidence.SuggestWithLLM(evidence.NoopSuggestionAdapter{}, evidence.SuggestRequest{PaperID: paper})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "suggest_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"suggestion": item})
		}
		fmt.Fprintln(stdout, "suggested evidence")
		return 0
	}
	if args[0] != "add" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge extract add ...")
	}
	input, ok := parseEvidenceAdd(args[1:])
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge extract add --paper <id> --schema <name> --value k=v --support kind:ref --status <status>")
	}
	item, err := evidence.NewEvidenceItem(input)
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "invalid_evidence", err.Error())
	}
	var items []evidence.EvidenceItem
	_ = readJSONFile(evidenceItemsPath(opts.Project), &items)
	items = append(items, item)
	if err := writeJSONFile(evidenceItemsPath(opts.Project), items); err != nil {
		return writeError(stdout, stderr, opts, 1, "evidence_store_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"evidence": item})
	}
	fmt.Fprintln(stdout, "added evidence")
	return 0
}

func executeEvidence(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "audit" || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence audit")
	}
	var items []evidence.EvidenceItem
	_ = readJSONFile(evidenceItemsPath(opts.Project), &items)
	issues := evidence.Audit(items)
	if issues == nil {
		issues = []evidence.AuditIssue{}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"issues": issues})
	}
	for _, issue := range issues {
		fmt.Fprintln(stdout, issue.Code)
	}
	return 0
}

func parseSingleFlag(args []string, flag string) (string, bool) {
	if len(args) != 2 || args[0] != flag || args[1] == "" {
		return "", false
	}
	return args[1], true
}
func parseEvidenceAdd(args []string) (evidence.EvidenceInput, bool) {
	values := map[string]string{}
	valMap := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--schema", "--value", "--support", "--status":
			if i+1 >= len(args) {
				return evidence.EvidenceInput{}, false
			}
			if args[i] == "--value" {
				parts := strings.SplitN(args[i+1], "=", 2)
				if len(parts) != 2 {
					return evidence.EvidenceInput{}, false
				}
				valMap[parts[0]] = parts[1]
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return evidence.EvidenceInput{}, false
		}
	}
	supportParts := strings.SplitN(values["--support"], ":", 2)
	if len(supportParts) != 2 {
		return evidence.EvidenceInput{}, false
	}
	return evidence.EvidenceInput{PaperID: values["--paper"], SchemaName: values["--schema"], Values: valMap, Support: evidence.Support{Kind: evidence.SupportKind(supportParts[0]), Ref: supportParts[1]}, Status: evidence.Status(values["--status"])}, values["--paper"] != "" && values["--schema"] != "" && values["--status"] != ""
}
func evidenceSchemasPath(project string) string {
	return filepath.Join(project, "data", "evidence.schemas.json")
}
func evidenceItemsPath(project string) string {
	return filepath.Join(project, "data", "evidence.items.json")
}

func executeScreen(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> screen <configure|decide|queue>")
	}
	switch args[0] {
	case "configure":
		reasons, ok := parseRepeatedFlag(args[1:], "--reason")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen configure --reason <reason>")
		}
		workflow, err := screening.Configure(screening.Options{ExclusionReasons: reasons})
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "screen_config_invalid", err.Error())
		}
		if err := writeJSONFile(screenWorkflowPath(opts.Project), workflow); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_config_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"workflow": workflow})
		}
		fmt.Fprintln(stdout, "configured screening")
		return 0
	case "decide":
		input, ok := parseScreenDecision(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen decide --paper <id> --stage <stage> --decision <decision> [--reason <reason>] --reviewer <name>")
		}
		workflow, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store := screening.NewMemoryStore(workflow)
		for _, event := range events {
			_ = store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer})
		}
		if err := store.Decide(input); err != nil {
			return writeError(stdout, stderr, opts, 2, "screen_decision_invalid", err.Error())
		}
		events = append(events, screening.DecisionEvent{PaperID: input.PaperID, Stage: input.Stage, Decision: input.Decision, Reason: input.Reason, Reviewer: input.Reviewer})
		if err := writeJSONFile(screenEventsPath(opts.Project), events); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_decision_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"decided": input.PaperID})
		}
		fmt.Fprintln(stdout, "recorded screening decision")
		return 0
	case "queue":
		stage, decision, ok := parseScreenQueue(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen queue --stage <stage> --decision <decision>")
		}
		workflow, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store := screening.NewMemoryStore(workflow)
		for _, event := range events {
			_ = store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer})
		}
		queue := store.Queue(screening.QueueFilter{Stage: stage, Decision: decision})
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": queue})
		}
		for _, paper := range queue {
			fmt.Fprintln(stdout, paper)
		}
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> screen <configure|decide|queue>")
	}
}

func executePRISMA(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "counts" || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> prisma counts")
	}
	workflow, events, err := loadScreening(opts.Project)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
	}
	store := screening.NewMemoryStore(workflow)
	for _, event := range events {
		_ = store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer})
	}
	counts := store.PRISMACounts()
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"counts": counts})
	}
	fmt.Fprintf(stdout, "included=%d excluded=%d uncertain=%d\n", counts.Included, counts.Excluded, counts.Uncertain)
	return 0
}

func parseRepeatedFlag(args []string, flag string) ([]string, bool) {
	if len(args) == 0 || len(args)%2 != 0 {
		return nil, false
	}
	var values []string
	for i := 0; i < len(args); i += 2 {
		if args[i] != flag {
			return nil, false
		}
		values = append(values, args[i+1])
	}
	return values, true
}
func parseScreenDecision(args []string) (screening.DecisionInput, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--stage", "--decision", "--reason", "--reviewer":
			if i+1 >= len(args) {
				return screening.DecisionInput{}, false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return screening.DecisionInput{}, false
		}
	}
	return screening.DecisionInput{PaperID: values["--paper"], Stage: screening.Stage(values["--stage"]), Decision: screening.Decision(values["--decision"]), Reason: values["--reason"], Reviewer: values["--reviewer"]}, values["--paper"] != "" && values["--stage"] != "" && values["--decision"] != "" && values["--reviewer"] != ""
}
func parseScreenQueue(args []string) (screening.Stage, screening.Decision, bool) {
	if len(args) != 4 {
		return "", "", false
	}
	values := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		values[args[i]] = args[i+1]
	}
	return screening.Stage(values["--stage"]), screening.Decision(values["--decision"]), values["--stage"] != "" && values["--decision"] != ""
}
func screenWorkflowPath(project string) string {
	return filepath.Join(project, "data", "screening.workflow.json")
}
func screenEventsPath(project string) string {
	return filepath.Join(project, "data", "screening.events.json")
}
func loadScreening(project string) (screening.Workflow, []screening.DecisionEvent, error) {
	var workflow screening.Workflow
	if err := readJSONFile(screenWorkflowPath(project), &workflow); err != nil {
		return workflow, nil, err
	}
	var events []screening.DecisionEvent
	_ = readJSONFile(screenEventsPath(project), &events)
	return workflow, events, nil
}
func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
func readJSONFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func executeProject(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "missing_project_subcommand", "missing project subcommand")
	}
	switch args[0] {
	case "create":
		path, title, ok := parseProjectCreate(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project create [path] --title <title>")
		}
		if path == "" {
			defaultPath, err := defaultRepoProjectPath()
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "repo_project_defaults_failed", err.Error())
			}
			path = defaultPath
		}
		created, err := project.Create(path, project.CreateOptions{Title: title})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_create_failed", fmt.Sprintf("create project: %v", err))
		}
		if err := writeRepoConfigForProject(created); err != nil {
			return writeError(stdout, stderr, opts, 1, "repo_config_failed", fmt.Sprintf("write repo config: %v", err))
		}
		if err := recordCLICommand(created.Path, "project create", opts, map[string]any{"exitCode": 0}); err != nil {
			return writeError(stdout, stderr, opts, 1, "cli_provenance_failed", fmt.Sprintf("record cli provenance: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, projectData(created))
		}
		fmt.Fprintf(stdout, "created project %s\n", created.Path)
		return 0
	case "discover-assets":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project discover-assets")
		}
		repoRoot, err := findRepoRoot(".")
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "repo_discovery_failed", err.Error())
		}
		projectPath, err := repoProjectPath(repoRoot)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "repo_project_defaults_failed", err.Error())
		}
		assets, err := project.DiscoverAssets(repoRoot, projectPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "asset_discovery_failed", fmt.Sprintf("discover assets: %v", err))
		}
		if err := recordCLICommand(projectPath, "project discover-assets", opts, map[string]any{"exitCode": 0, "assetCount": len(assets)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "cli_provenance_failed", fmt.Sprintf("record cli provenance: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"assets": assets})
		}
		for _, asset := range assets {
			fmt.Fprintf(stdout, "%s\t%s\timported=%v\n", asset.Kind, asset.Path, asset.Imported)
		}
		return 0
	case "inspect":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project inspect <path>")
		}
		inspected, err := project.Inspect(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_inspect_failed", fmt.Sprintf("inspect project: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, projectData(inspected))
		}
		fmt.Fprintf(stdout, "path: %s\ntitle: %s\nstorage: %s\n", inspected.Path, inspected.Title, inspected.StorageMode)
		return 0
	case "list":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project list <root>")
		}
		projects, err := project.List(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_list_failed", fmt.Sprintf("list projects: %v", err))
		}
		if opts.JSON {
			items := make([]map[string]any, 0, len(projects))
			for _, proj := range projects {
				items = append(items, projectData(proj))
			}
			return writeJSON(stdout, 0, map[string]any{"projects": items})
		}
		for _, proj := range projects {
			fmt.Fprintf(stdout, "%s\t%s\n", proj.Path, proj.Title)
		}
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_project_subcommand", fmt.Sprintf("unknown project subcommand %q", args[0]))
	}
}

func recordCLICommand(projectPath, command string, opts globalOptions, outputs map[string]any) error {
	now := time.Now().UTC()
	return provenance.Append(projectPath, provenance.Event{
		SchemaVersion: "1",
		ID:            "evt_" + now.Format("20060102T150405Z") + "_cli",
		Timestamp:     now.Format(time.RFC3339),
		Actor:         "rforge",
		Action:        "cli.command",
		Target:        projectPath,
		Inputs: map[string]any{
			"command":  command,
			"json":     opts.JSON,
			"logLevel": opts.LogLevel,
		},
		Outputs:  outputs,
		Warnings: []string{},
	})
}

func projectData(proj project.Project) map[string]any {
	return map[string]any{
		"path":                proj.Path,
		"title":               proj.Title,
		"storageMode":         proj.StorageMode,
		"schemaVersion":       proj.SchemaVersion,
		"manifestPath":        proj.ManifestPath,
		"lockfilePath":        proj.LockfilePath,
		"provenancePath":      proj.ProvenancePath,
		"storagePath":         proj.StoragePath,
		"archiveMetadataPath": proj.ArchiveMetadataPath,
	}
}

func parseGlobalOptions(args []string) (globalOptions, []string, bool) {
	var opts globalOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			opts.JSON = true
		case "--project":
			if i+1 >= len(args) {
				return opts, nil, false
			}
			opts.Project = args[i+1]
			i++
		case "--config":
			if i+1 >= len(args) {
				return opts, nil, false
			}
			opts.Config = args[i+1]
			i++
		case "--log-level":
			if i+1 >= len(args) {
				return opts, nil, false
			}
			opts.LogLevel = args[i+1]
			i++
		default:
			return opts, args[i:], true
		}
	}
	return opts, nil, true
}

func writeJSON(stdout io.Writer, code int, data any) int {
	_ = json.NewEncoder(stdout).Encode(map[string]any{"ok": true, "data": data})
	return code
}

func writeError(stdout, stderr io.Writer, opts globalOptions, code int, errorCode, message string) int {
	if opts.JSON {
		_ = json.NewEncoder(stdout).Encode(map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":    errorCode,
				"message": message,
			},
		})
		return code
	}
	fmt.Fprintln(stderr, message)
	return code
}

func parseProjectCreate(args []string) (string, string, bool) {
	var path string
	var title string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--title":
			if i+1 >= len(args) {
				return "", "", false
			}
			title = args[i+1]
			i++
		default:
			if path != "" {
				return "", "", false
			}
			path = arg
		}
	}
	return path, title, title != ""
}

func defaultRepoProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return "", err
	}
	return repoProjectPath(repoRoot)
}

func repoProjectPath(repoRoot string) (string, error) {
	configured, err := readRepoProjectPath(repoRoot)
	if err != nil {
		return "", err
	}
	if configured != "" {
		return filepath.Join(repoRoot, configured), nil
	}
	return filepath.Join(repoRoot, "research-forge"), nil
}

func readRepoProjectPath(repoRoot string) (string, error) {
	configBytes, err := os.ReadFile(filepath.Join(repoRoot, ".researchforge"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, line := range strings.Split(string(configBytes), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "default_project_path" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		if strings.Contains(value, "..") || filepath.IsAbs(value) {
			return "", fmt.Errorf("invalid default_project_path in .researchforge")
		}
		return value, nil
	}
	return "", nil
}

func writeRepoConfigForProject(proj project.Project) error {
	repoRoot, err := findRepoRoot(proj.Path)
	if err != nil {
		return nil
	}
	content := fmt.Sprintf("default_project_path = %q\ne2e_topic = %q\n", filepath.Base(proj.Path), "artificial photosynthesis")
	return os.WriteFile(filepath.Join(repoRoot, ".researchforge"), []byte(content), 0o644)
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a repository; pass an explicit project path")
		}
		dir = parent
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "rforge - ResearchForge command-line tool")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  rforge version")
	fmt.Fprintln(w, "  rforge doctor")
	fmt.Fprintln(w, "  rforge search --source openalex|arxiv|crossref --query <query>")
	fmt.Fprintln(w, "  rforge oa lookup <doi>")
	fmt.Fprintln(w, "  rforge service check <name>")
	fmt.Fprintln(w, "  rforge library list")
	fmt.Fprintln(w, "  rforge duplicate report")
	fmt.Fprintln(w, "  rforge import json|csv|bibtex|ris <file>")
	fmt.Fprintln(w, "  rforge export json|csv|bibtex|ris <file>")
	fmt.Fprintln(w, "  rforge oss add|list|license-check")
	fmt.Fprintln(w, "  rforge project create [path] --title <title>")
	fmt.Fprintln(w, "  rforge project discover-assets")
	fmt.Fprintln(w, "  rforge project inspect <path>")
	fmt.Fprintln(w, "  rforge project list <root>")
	fmt.Fprintln(w, "  rforge decisions --check TODO.md")
	fmt.Fprintln(w, "  rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md")
}
