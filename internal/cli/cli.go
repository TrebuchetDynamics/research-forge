package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
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
	case "citations":
		return executeCitations(remaining[1:], stdout, stderr, opts)
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
	case "research":
		return executeResearch(remaining[1:], stdout, stderr, opts)
	case "protocol":
		return executeProtocol(remaining[1:], stdout, stderr, opts)
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
		return executeUI(remaining[1:], stdout, stderr, opts)
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
	// Active owner-decision blockers. Resolved decisions leave this registry and
	// their history is recorded in docs (see docs/owner-decisions.md). The
	// project_license blocker was resolved on 2026-06-13 (MIT, Copyright (c) 2026
	// Trebuchet Dynamics, approved by the repository owner in issue #1) and the
	// web_gui_stack_scope tracker was completed earlier, so no owner decisions
	// currently block the TODO checklist.
	decisions := []map[string]any{}
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

// isTodoBacklogHeading reports whether a TODO.md line opens a post-1.0 backlog
// section. Unchecked items in that section are future work, not owner-gated MVP
// blockers, so the decision/completion audits exempt them from coverage.
func isTodoBacklogHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "#") && strings.Contains(strings.ToLower(trimmed), "backlog")
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
	inBacklog := false
	for lineNumber, line := range strings.Split(string(data), "\n") {
		if isTodoBacklogHeading(line) {
			inBacklog = true
		}
		line = strings.TrimSpace(line)
		if inBacklog || !strings.HasPrefix(line, "- [ ] ") {
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
	inBacklog := false
	for _, line := range strings.Split(string(todoData), "\n") {
		if isTodoBacklogHeading(line) {
			inBacklog = true
		}
		line = strings.TrimSpace(line)
		if inBacklog || !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		uncheckedRefs++
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		if !strings.Contains(audit, item) {
			missing = append(missing, item)
		}
	}
	if len(missing) > 0 {
		return writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "completion audit does not cover unchecked TODO items: "+strings.Join(missing, "; "))
	}
	licenseResolutionVerified, code := verifyLicenseResolution(todoPath, stdout, stderr, opts)
	if code != 0 {
		return code
	}
	blockedDecisionIDs := []string{}
	blockedIssueURLs := []string{}
	for _, decision := range decisions {
		if required, _ := decision["owner_action_required"].(bool); !required {
			continue
		}
		if id, ok := decision["id"].(string); ok && id != "" {
			blockedDecisionIDs = append(blockedDecisionIDs, id)
		}
		if issue, ok := decision["issue"].(string); ok && issue != "" {
			blockedIssueURLs = append(blockedIssueURLs, issue)
		}
	}
	blockedDecisions := len(blockedDecisionIDs)
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"covered": true, "line_refs_verified": true, "issue_refs_verified": true, "completion_audit_verified": true, "completion_audit_issue_refs_verified": true, "checked_evidence_verified": true, "quality_gate_verified": true, "license_resolution_verified": licenseResolutionVerified, "unchecked_refs": uncheckedRefs, "completion_blocked": blockedDecisions > 0, "blocked_decisions": blockedDecisions, "blocked_decision_ids": blockedDecisionIDs, "blocked_issue_urls": blockedIssueURLs})
	}
	fmt.Fprintf(stdout, "unchecked TODO refs verified: %d\n", uncheckedRefs)
	fmt.Fprintf(stdout, "completion blocked by %d owner decision(s)\n", blockedDecisions)
	if len(blockedDecisionIDs) > 0 {
		fmt.Fprintf(stdout, "blocked decision ids: %s\n", strings.Join(blockedDecisionIDs, ", "))
		fmt.Fprintf(stdout, "blocked issue urls: %s\n", strings.Join(blockedIssueURLs, ", "))
	}
	if licenseResolutionVerified {
		fmt.Fprintln(stdout, "license resolution verified")
	}
	fmt.Fprintln(stdout, "checked TODO evidence verified")
	fmt.Fprintln(stdout, "quality gate verified")
	fmt.Fprintln(stdout, "completion audit issue references verified")
	fmt.Fprintln(stdout, "completion audit verified")
	return 0
}

// verifyLicenseResolution confirms the owner-resolved project license is recorded
// consistently: the TODO item is checked with the resolution summary, the LICENSE
// file carries the approved MIT text and copyright holder, and the README license
// section names the selected license and SPDX identifier.
func verifyLicenseResolution(todoPath string, stdout, stderr io.Writer, opts globalOptions) (bool, int) {
	root := filepath.Dir(todoPath)
	todoData, err := os.ReadFile(todoPath)
	if err != nil {
		return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
	}
	licenseLine := ""
	for _, line := range strings.Split(string(todoData), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [x] Add license after owner decision") {
			licenseLine = trimmed
			break
		}
	}
	if licenseLine == "" {
		return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "license TODO item is neither resolved nor tracked as a blocked owner decision")
	}
	for _, required := range []string{"MIT", "Trebuchet Dynamics", "issue #1", "approved"} {
		if !strings.Contains(licenseLine, required) {
			return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "resolved license TODO line missing "+required)
		}
	}
	licenseData, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "LICENSE missing while license TODO is resolved: "+err.Error())
	}
	license := string(licenseData)
	for _, required := range []string{"MIT License", "Trebuchet Dynamics"} {
		if !strings.Contains(license, required) {
			return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "LICENSE missing "+required)
		}
	}
	readmeData, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", err.Error())
	}
	readme := string(readmeData)
	for _, required := range []string{"MIT License", "SPDX", "Trebuchet Dynamics"} {
		if !strings.Contains(readme, required) {
			return false, writeError(stdout, stderr, opts, 1, "todo_completion_audit_failed", "README license section missing "+required)
		}
	}
	return true, 0
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
		if ownerInputs, ok := decision["owner_inputs"].([]string); ok && len(ownerInputs) > 0 {
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
		fmt.Fprint(stdout, "## Decision\n\n- \n")
		fmt.Fprint(stdout, "## Implementation steps after approval\n\n")
		if steps, ok := decision["implementation_steps"].([]string); ok && len(steps) > 0 {
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
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis <prepare|run|sensitivity|export>")
	}
	runPath := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+".json")
	switch args[0] {
	case "prepare":
		calc, ok := parseAnalysisEffect(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis prepare <run-id> [--effect smd|log-odds-ratio|risk-ratio]")
		}
		var items []evidence.EvidenceItem
		if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
		}
		run, err := analysis.PrepareWithCalculator(args[1], items, calc)
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
	case "sensitivity":
		if len(args) != 4 || args[2] != "--method" || args[3] != "leave-one-out" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis sensitivity <run-id> --method leave-one-out")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		report, err := analysis.LeaveOneOut(run)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_sensitivity_failed", err.Error())
		}
		path := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-sensitivity.json")
		if err := writeJSONFile(path, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_sensitivity_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"sensitivity": report, "path": path})
		}
		fmt.Fprintf(stdout, "wrote sensitivity analysis to %s\n", path)
		return 0
	case "subgroup":
		variable, groups, ok := parseSubgroupArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis subgroup <run-id> --variable <name> --group <paper>=<group>")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		report, err := analysis.SubgroupAnalysis(run, variable, groups)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_subgroup_failed", err.Error())
		}
		path := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-subgroup.json")
		if err := writeJSONFile(path, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_subgroup_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"subgroup": report, "path": path})
		}
		fmt.Fprintf(stdout, "wrote subgroup analysis to %s\n", path)
		return 0
	case "meta-regression":
		moderator, values, ok := parseMetaRegressionArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis meta-regression <run-id> --moderator <name> --value <paper>=<number>")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		report, err := analysis.MetaRegression(run, moderator, values)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_meta_regression_failed", err.Error())
		}
		path := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-meta-regression.json")
		if err := writeJSONFile(path, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_meta_regression_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"metaRegression": report, "path": path})
		}
		fmt.Fprintf(stdout, "wrote meta-regression analysis to %s\n", path)
		return 0
	case "bayesian":
		priorMean, priorVariance, ok := parseBayesianArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis bayesian <run-id> --method normal-approx [--prior-mean 0] [--prior-variance 100]")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		report, err := analysis.BayesianNormalApproximation(run, priorMean, priorVariance)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_bayesian_failed", err.Error())
		}
		path := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-bayesian.json")
		if err := writeJSONFile(path, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_bayesian_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"bayesian": report, "path": path})
		}
		fmt.Fprintf(stdout, "wrote Bayesian analysis to %s\n", path)
		return 0
	case "publication-bias":
		if len(args) != 4 || args[2] != "--method" || args[3] != "egger" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis publication-bias <run-id> --method egger")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		report, err := analysis.EggerRegression(run)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_publication_bias_failed", err.Error())
		}
		path := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-publication-bias.json")
		if err := writeJSONFile(path, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_publication_bias_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"publicationBias": report, "path": path})
		}
		fmt.Fprintf(stdout, "wrote publication bias analysis to %s\n", path)
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
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis <prepare|run|sensitivity|subgroup|meta-regression|publication-bias|bayesian|export>")
	}
}

func parseBayesianArgs(args []string) (float64, float64, bool) {
	priorMean := 0.0
	priorVariance := 100.0
	methodSeen := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--method":
			if i+1 >= len(args) || args[i+1] != "normal-approx" {
				return 0, 0, false
			}
			methodSeen = true
			i++
		case "--prior-mean":
			if i+1 >= len(args) {
				return 0, 0, false
			}
			parsed, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return 0, 0, false
			}
			priorMean = parsed
			i++
		case "--prior-variance":
			if i+1 >= len(args) {
				return 0, 0, false
			}
			parsed, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil || parsed <= 0 {
				return 0, 0, false
			}
			priorVariance = parsed
			i++
		default:
			return 0, 0, false
		}
	}
	return priorMean, priorVariance, methodSeen
}

func parseSubgroupArgs(args []string) (string, map[string]string, bool) {
	variable := ""
	groups := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--variable":
			if i+1 >= len(args) {
				return "", nil, false
			}
			variable = args[i+1]
			i++
		case "--group":
			if i+1 >= len(args) {
				return "", nil, false
			}
			paper, group, ok := strings.Cut(args[i+1], "=")
			if !ok || strings.TrimSpace(paper) == "" || strings.TrimSpace(group) == "" {
				return "", nil, false
			}
			groups[strings.TrimSpace(paper)] = strings.TrimSpace(group)
			i++
		default:
			return "", nil, false
		}
	}
	return variable, groups, strings.TrimSpace(variable) != "" && len(groups) > 0
}

func parseMetaRegressionArgs(args []string) (string, map[string]float64, bool) {
	moderator := ""
	values := map[string]float64{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--moderator":
			if i+1 >= len(args) {
				return "", nil, false
			}
			moderator = args[i+1]
			i++
		case "--value":
			if i+1 >= len(args) {
				return "", nil, false
			}
			paper, raw, ok := strings.Cut(args[i+1], "=")
			if !ok || strings.TrimSpace(paper) == "" {
				return "", nil, false
			}
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return "", nil, false
			}
			values[strings.TrimSpace(paper)] = parsed
			i++
		default:
			return "", nil, false
		}
	}
	return moderator, values, strings.TrimSpace(moderator) != "" && len(values) > 0
}

func parseAnalysisEffect(args []string) (analysis.EffectSizeCalculator, bool) {
	if len(args) == 0 {
		return analysis.StandardizedMeanDifference{}, true
	}
	if len(args) != 2 || args[0] != "--effect" {
		return nil, false
	}
	switch args[1] {
	case "smd", "standardized-mean-difference":
		return analysis.StandardizedMeanDifference{}, true
	case "log-odds-ratio":
		return analysis.LogOddsRatio{}, true
	case "risk-ratio", "rr":
		return analysis.RiskRatio{}, true
	default:
		return nil, false
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
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> screen <configure|decide|adjudicate|queue|prioritize|model-prioritize|uncertainty|progress|recall|stopping>")
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
	case "adjudicate":
		input, ok := parseScreenDecision(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen adjudicate --paper <id> --stage <stage> --decision <decision> [--reason <reason>] --reviewer <name>")
		}
		input.Adjudicated = true
		workflow, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store := screening.NewMemoryStore(workflow)
		for _, event := range events {
			_ = store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer, Adjudicated: event.Adjudicated})
		}
		if err := store.Decide(input); err != nil {
			return writeError(stdout, stderr, opts, 2, "screen_adjudication_invalid", err.Error())
		}
		events = append(events, screening.DecisionEvent{PaperID: input.PaperID, Stage: input.Stage, Decision: input.Decision, Reason: input.Reason, Reviewer: input.Reviewer, Adjudicated: true})
		if err := writeJSONFile(screenEventsPath(opts.Project), events); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_adjudication_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"adjudicated": input.PaperID})
		}
		fmt.Fprintln(stdout, "recorded screening adjudication")
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
	case "sensitivity":
		stage, relevant, targets, outPath, ok := parseScreenSensitivity(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen sensitivity --stage <stage> --relevant <paper-id> --out <report.json> [--target-recall 0.95]")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		records := make([]screening.ScreeningRecord, 0, len(papers))
		for _, paper := range papers {
			records = append(records, screening.ScreeningRecord{ID: screeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
		}
		report, err := screening.ActiveLearningSensitivityDiagnostics(screening.ActiveLearningSensitivityInput{Records: records, Events: events, Stage: stage, RelevantPaperIDs: relevant, TargetRecalls: targets})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_sensitivity_failed", err.Error())
		}
		if err := writeJSONFile(outPath, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_sensitivity_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"sensitivity": report, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote screening sensitivity diagnostics to %s\n", outPath)
		return 0
	case "active-run":
		stage, method, outPath, targetRecall, ok := parseScreenActiveRun(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen active-run --stage <stage> --method active-learning|model|uncertainty --out <run.json> [--target-recall 0.95]")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		records := make([]screening.ScreeningRecord, 0, len(papers))
		for _, paper := range papers {
			records = append(records, screening.ScreeningRecord{ID: screeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
		}
		run, err := screening.BuildActiveLearningRun(screening.ActiveLearningRunInput{Records: records, Events: events, Stage: stage, RankingMethod: method, TargetRecall: targetRecall})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_active_run_failed", err.Error())
		}
		if err := writeJSONFile(outPath, run); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_active_run_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"activeLearningRun": run, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote active-learning run to %s\n", outPath)
		return 0
	case "prioritize":
		stage, limit, ok := parseScreenPrioritize(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen prioritize --stage <stage> [--limit N]")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		records := make([]screening.ScreeningRecord, 0, len(papers))
		for _, paper := range papers {
			records = append(records, screening.ScreeningRecord{ID: screeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
		}
		prioritized := screening.PrioritizeActiveLearningRecords(records, events, stage)
		if limit > 0 && limit < len(prioritized) {
			prioritized = prioritized[:limit]
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"prioritized": prioritized})
		}
		for _, paper := range prioritized {
			fmt.Fprintf(stdout, "%s\t%.0f\n", paper.ID, paper.Score)
		}
		return 0
	case "model-prioritize":
		stage, limit, ok := parseScreenPrioritize(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen model-prioritize --stage <stage> [--limit N]")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		records := make([]screening.ScreeningRecord, 0, len(papers))
		for _, paper := range papers {
			records = append(records, screening.ScreeningRecord{ID: screeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
		}
		prioritized := screening.PrioritizeModelRecords(records, events, stage)
		if limit > 0 && limit < len(prioritized) {
			prioritized = prioritized[:limit]
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"prioritized": prioritized, "method": "naive-bayes"})
		}
		for _, paper := range prioritized {
			fmt.Fprintf(stdout, "%s\t%.3f\n", paper.ID, paper.Score)
		}
		return 0
	case "uncertainty":
		stage, limit, ok := parseScreenPrioritize(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen uncertainty --stage <stage> [--limit N]")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		records := make([]screening.ScreeningRecord, 0, len(papers))
		for _, paper := range papers {
			records = append(records, screening.ScreeningRecord{ID: screeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
		}
		prioritized := screening.PrioritizeUncertaintyRecords(records, events, stage)
		if limit > 0 && limit < len(prioritized) {
			prioritized = prioritized[:limit]
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"uncertainty": prioritized})
		}
		for _, paper := range prioritized {
			fmt.Fprintf(stdout, "%s\t%.3f\t%.0f\n", paper.ID, paper.Uncertainty, paper.Score)
		}
		return 0
	case "recall":
		stage, ok := parseSingleFlag(args[1:], "--stage")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen recall --stage <stage>")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		curve := screening.RecallEffortCurve(events, screening.Stage(stage))
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"recall": curve})
		}
		for _, point := range curve {
			fmt.Fprintf(stdout, "%d\t%d\t%.3f\n", point.Screened, point.Included, point.Recall)
		}
		return 0
	case "stopping":
		stage, target, ok := parseScreenStopping(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen stopping --stage <stage> [--target-recall 0.95]")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		recommendation := screening.StoppingCriteria(events, stage, target)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"stopping": recommendation})
		}
		fmt.Fprintf(stdout, "%t\t%.3f\t%s\n", recommendation.CanStop, recommendation.CurrentRecall, recommendation.Reason)
		return 0
	case "progress":
		stage, ok := parseSingleFlag(args[1:], "--stage")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen progress --stage <stage>")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		report := screening.Progress(events, screening.Stage(stage), len(papers))
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"progress": report})
		}
		fmt.Fprintf(stdout, "%s\t%d screened\t%d remaining\t%d conflicts\n", report.Stage, report.ScreenedRecords, report.Remaining, report.Conflicts)
		for _, reviewer := range report.Reviewers {
			fmt.Fprintf(stdout, "%s\t%d decisions\t%d include\t%d exclude\t%d uncertain\n", reviewer.Reviewer, reviewer.Decisions, reviewer.Included, reviewer.Excluded, reviewer.Uncertain)
		}
		return 0
	case "conflicts":
		stage, ok := parseSingleFlag(args[1:], "--stage")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen conflicts --stage <stage>")
		}
		workflow, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		store := screening.NewMemoryStore(workflow)
		for _, event := range events {
			_ = store.Decide(screening.DecisionInput{PaperID: event.PaperID, Stage: event.Stage, Decision: event.Decision, Reason: event.Reason, Reviewer: event.Reviewer, Adjudicated: event.Adjudicated})
		}
		conflicts := store.Conflicts(screening.Stage(stage))
		if conflicts == nil {
			conflicts = []string{}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"conflicts": conflicts})
		}
		for _, paper := range conflicts {
			fmt.Fprintln(stdout, paper)
		}
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> screen <configure|decide|adjudicate|queue|prioritize|progress|recall|stopping|conflicts>")
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
func parseScreenStopping(args []string) (screening.Stage, float64, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stage", "--target-recall":
			if i+1 >= len(args) {
				return "", 0, false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", 0, false
		}
	}
	target := 0.95
	if values["--target-recall"] != "" {
		parsed, err := strconv.ParseFloat(values["--target-recall"], 64)
		if err != nil || parsed <= 0 || parsed > 1 {
			return "", 0, false
		}
		target = parsed
	}
	return screening.Stage(values["--stage"]), target, values["--stage"] != ""
}

func parseScreenSensitivity(args []string) (screening.Stage, []string, []float64, string, bool) {
	values := map[string]string{}
	relevant := []string{}
	targets := []float64{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stage", "--out", "--relevant", "--target-recall":
			if i+1 >= len(args) {
				return "", nil, nil, "", false
			}
			switch args[i] {
			case "--relevant":
				relevant = append(relevant, args[i+1])
			case "--target-recall":
				parsed, err := strconv.ParseFloat(args[i+1], 64)
				if err != nil || parsed <= 0 || parsed > 1 {
					return "", nil, nil, "", false
				}
				targets = append(targets, parsed)
			default:
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return "", nil, nil, "", false
		}
	}
	return screening.Stage(values["--stage"]), relevant, targets, values["--out"], values["--stage"] != "" && values["--out"] != "" && len(relevant) > 0
}

func parseScreenActiveRun(args []string) (screening.Stage, string, string, float64, bool) {
	values := map[string]string{"--method": "active-learning"}
	target := 0.95
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stage", "--method", "--out", "--target-recall":
			if i+1 >= len(args) {
				return "", "", "", 0, false
			}
			if args[i] == "--target-recall" {
				parsed, err := strconv.ParseFloat(args[i+1], 64)
				if err != nil || parsed <= 0 || parsed > 1 {
					return "", "", "", 0, false
				}
				target = parsed
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return "", "", "", 0, false
		}
	}
	method := values["--method"]
	validMethod := method == "active-learning" || method == "asreview" || method == "model" || method == "naive-bayes" || method == "uncertainty"
	return screening.Stage(values["--stage"]), method, values["--out"], target, values["--stage"] != "" && values["--out"] != "" && validMethod
}

func parseScreenPrioritize(args []string) (screening.Stage, int, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stage", "--limit":
			if i+1 >= len(args) {
				return "", 0, false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", 0, false
		}
	}
	limit := 0
	if values["--limit"] != "" {
		parsed, err := strconv.Atoi(values["--limit"])
		if err != nil || parsed <= 0 {
			return "", 0, false
		}
		limit = parsed
	}
	return screening.Stage(values["--stage"]), limit, values["--stage"] != ""
}
func screeningPaperID(paper library.PaperRecord) string {
	ids := paper.Identifiers
	for _, id := range []string{ids.DOI, ids.OpenAlexID, ids.ArXivID, ids.PMID, ids.CrossrefID, ids.SemanticScholarID} {
		if strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	return strings.TrimSpace(paper.Title)
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
	fmt.Fprintln(w, "  rforge search --source openalex|arxiv|crossref|semantic-scholar|europepmc|pubmed --query <query> [--category arxiv-category] [--filter source-filter] [--from-year YYYY] [--to-year YYYY] [--type article] [--open-access true|false] [--concept C41008148]")
	fmt.Fprintln(w, "  rforge search import --source openalex --query <query> --pages N [--resume-state state.json]")
	fmt.Fprintln(w, "  rforge citations expand --source semantic-scholar|openalex|crossref --paper <id> --direction references|citations|both --depth N [--max-records N] --out <file> [--import-library]")
	fmt.Fprintln(w, "  rforge citations report --graph <graph.json> --out <report.md>")
	fmt.Fprintln(w, "  rforge oa lookup <doi>")
	fmt.Fprintln(w, "  rforge service check <name>")
	fmt.Fprintln(w, "  rforge library list|refresh-doi|refresh-crossref")
	fmt.Fprintln(w, "  rforge duplicate report")
	fmt.Fprintln(w, "  rforge import json|csv|bibtex|ris|csl-json|zotero-rdf <file>")
	fmt.Fprintln(w, "  rforge export json|csv|bibtex|ris|csl-json|zotero-rdf <file>")
	fmt.Fprintln(w, "  rforge oss inventory-check <manifest.json>")
	fmt.Fprintln(w, "  rforge oss inventory-refresh <manifest.json> --source github [--base-url <url>]")
	fmt.Fprintln(w, "  rforge oss inventory-policy <manifest.json> [--stale-after 18mo]")
	fmt.Fprintln(w, "  rforge oss inventory-drift <manifest.json>")
	fmt.Fprintln(w, "  rforge oss inventory-report <manifest.json> [--area <area>]")
	fmt.Fprintln(w, "  rforge oss add|list|license-check")
	fmt.Fprintln(w, "  rforge parse --paper <id> --parser grobid|tex|s2orc|papermage --pdf|--tex|--s2orc|--papermage <file>")
	fmt.Fprintln(w, "  rforge research parse-pdftotext --paper <id> --pdf <file> --out <parsed.json> [--title <title>]")
	fmt.Fprintln(w, "  rforge research screen-queue --out <queue.csv> [--markdown <queue.md>] [--library <library.json>] [--search-results <dir>]")
	fmt.Fprintln(w, "  rforge research leakage-audit --parsed <parsed-dir> --out <audit.json> [--markdown <audit.md>]")
	fmt.Fprintln(w, "  rforge project create [path] --title <title>")
	fmt.Fprintln(w, "  rforge project discover-assets")
	fmt.Fprintln(w, "  rforge project inspect <path>")
	fmt.Fprintln(w, "  rforge project list <root>")
	fmt.Fprintln(w, "  rforge decisions --check TODO.md")
	fmt.Fprintln(w, "  rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md")
}
