package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/automation"
	"github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/forge"
	"github.com/TrebuchetDynamics/research-forge/internal/knowledge"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/report"
	"github.com/TrebuchetDynamics/research-forge/internal/reviewpkg"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
	rwatch "github.com/TrebuchetDynamics/research-forge/internal/watch"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var topLevelCommands = []string{"version", "decisions", "benchmark", "automation", "completion", "forge", "project", "doctor", "service", "library", "search", "citations", "oa", "duplicate", "import", "export", "oss", "pdf", "parse", "index", "knowledge", "graph", "retrieve", "research", "protocol", "screen", "prisma", "extraction", "extract", "evidence", "analysis", "report", "package", "archive", "ui", "watch", "inbox", "fetch", "provenance", "journal", "goal", "meta"}

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
	case "benchmark":
		return executeBenchmark(remaining[1:], stdout, stderr, opts)
	case "automation":
		return executeAutomation(remaining[1:], stdout, stderr, opts)
	case "completion":
		return executeCompletion(remaining[1:], stdout, stderr, opts)
	case "forge":
		return executeForge(remaining[1:], stdout, stderr, opts)
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
	case "knowledge":
		return executeKnowledge(remaining[1:], stdout, stderr, opts)
	case "graph":
		return executeGraph(remaining[1:], stdout, stderr, opts)
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
	case "package":
		return executePackage(remaining[1:], stdout, stderr, opts)
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
	case "provenance":
		return executeProvenance(remaining[1:], stdout, stderr, opts)
	case "journal":
		return executeJournal(remaining[1:], stdout, stderr, opts)
	case "goal":
		return executeGoal(remaining[1:], stdout, stderr, opts)
	case "meta":
		return executeMeta(remaining[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_command", fmt.Sprintf("unknown command %q", remaining[0]))
	}
}

func executeAutomation(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "policy" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge automation policy [--action <action>]")
	}
	action := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--action":
			if i+1 >= len(args) {
				return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge automation policy [--action <action>]")
			}
			action = args[i+1]
			i++
		default:
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge automation policy [--action <action>]")
		}
	}
	if action != "" {
		decision := automation.Evaluate(action)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"decision": decision})
		}
		fmt.Fprintf(stdout, "%s\t%s\tactor=%s\thuman=%t\tgate=%s\taudit=%s\n", decision.Action, decision.Class, decision.AllowedActor, decision.RequiresHuman, decision.Gate, decision.AuditArtifact)
		fmt.Fprintln(stdout, decision.Reason)
		return 0
	}
	policy := automation.DefaultPolicy()
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"policy": policy})
	}
	fmt.Fprintln(stdout, "ResearchForge hybrid automation policy")
	for _, decision := range policy {
		fmt.Fprintf(stdout, "%s\t%s\tactor=%s\thuman=%t", decision.Action, decision.Class, decision.AllowedActor, decision.RequiresHuman)
		if decision.Gate != "" {
			fmt.Fprintf(stdout, "\tgate=%s", decision.Gate)
		}
		fmt.Fprintf(stdout, "\taudit=%s\n", decision.AuditArtifact)
	}
	return 0
}

func executeBenchmark(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "cross-tool" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge benchmark cross-tool --out <report.json>")
	}
	out, ok := parseBenchmarkCrossTool(args[1:])
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge benchmark cross-tool --out <report.json>")
	}
	report := benchmarks.BuildCrossToolBenchmarkReport(benchmarks.DefaultCrossToolBenchmarkInput())
	if err := writeJSONFile(out, report); err != nil {
		return writeError(stdout, stderr, opts, 1, "benchmark_write_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"benchmark": "cross-tool", "report": report, "out": out})
	}
	fmt.Fprintf(stdout, "wrote cross-tool benchmark report to %s\n", out)
	return 0
}

func parseBenchmarkCrossTool(args []string) (string, bool) {
	if len(args) != 2 || args[0] != "--out" || strings.TrimSpace(args[1]) == "" {
		return "", false
	}
	return args[1], true
}

func executeKnowledge(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge knowledge query|path --project <path>")
	}
	switch args[0] {
	case "query":
		projectPath, term, ok := parseKnowledgeQuery(args[1:])
		if !ok || projectPath == "" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge knowledge query --project <path> [--term <text>]")
		}
		graph, err := knowledge.LoadProjectKnowledgeGraphFromProject(projectPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "knowledge_graph_failed", err.Error())
		}
		if term != "" {
			graph = knowledge.QueryProjectKnowledgeGraph(graph, term)
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"graph": graph})
		}
		fmt.Fprintf(stdout, "nodes: %d\nedges: %d\n", len(graph.Nodes), len(graph.Edges))
		for _, node := range graph.Nodes {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", node.Kind, node.ID, node.Label)
		}
		return 0
	case "path":
		projectPath, fromID, toID, ok := parseKnowledgePath(args[1:])
		if !ok || projectPath == "" || fromID == "" || toID == "" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge knowledge path --project <path> --from <node-id> --to <node-id>")
		}
		graph, err := knowledge.LoadProjectKnowledgeGraphFromProject(projectPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "knowledge_graph_failed", err.Error())
		}
		path, found := knowledge.ShortestPathIDs(graph, fromID, toID)
		if !found {
			return writeError(stdout, stderr, opts, 1, "knowledge_path_not_found", "no path found")
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"path": path})
		}
		fmt.Fprintln(stdout, strings.Join(path, " -> "))
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge knowledge query|path --project <path>")
	}
}

func parseKnowledgeQuery(args []string) (string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project", "--term":
			if i+1 >= len(args) {
				return "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", false
		}
	}
	return values["--project"], values["--term"], true
}

func parseKnowledgePath(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project", "--from", "--to":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--project"], values["--from"], values["--to"], true
}

func executeForge(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge forge init|status|next|approve|reopen|replay|run-dag|source-fixture|reference-fixture|acquisition-fixture|package-fixture --project <path>")
	}
	sub := args[0]
	projectPath, values, ok := parseForgeOptions(args[1:])
	if !ok || projectPath == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge forge "+sub+" --project <path>")
	}
	if sub == "source-fixture" {
		state, err := forge.CompleteFixtureSourceImport(projectPath, values["--actor"])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "forge_source_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"state": state})
		}
		fmt.Fprintf(stdout, "state: %s\nsource import fixture prepared\n", state.CurrentState)
		return 0
	}
	if sub == "reference-fixture" {
		state, err := forge.CompleteFixtureReferenceManager(projectPath, values["--actor"])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "forge_reference_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"state": state})
		}
		fmt.Fprintf(stdout, "state: %s\nreference-manager fixture prepared\n", state.CurrentState)
		return 0
	}
	if sub == "acquisition-fixture" {
		state, err := forge.CompleteFixtureAcquisition(projectPath, values["--actor"])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "forge_acquisition_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"state": state})
		}
		fmt.Fprintf(stdout, "state: %s\nlegal acquisition fixture prepared\n", state.CurrentState)
		return 0
	}
	if sub == "package-fixture" {
		outPath := values["--out"]
		if outPath == "" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge forge package-fixture --project <path> --out <dir>")
		}
		result, err := forge.CompleteFixturePackage(projectPath, outPath, values["--actor"])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "forge_package_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"packageCompletion": result, "state": result.State})
		}
		fmt.Fprintf(stdout, "state: %s\npackage: %s\naudit ok: %t\nreplay ok: %t\n", result.State.CurrentState, result.PackagePath, result.AuditReport.OK, result.ReplayReport.OK)
		return 0
	}
	if sub == "run-dag" {
		maxSteps := 0
		if values["--max-steps"] != "" {
			parsed, err := strconv.Atoi(values["--max-steps"])
			if err != nil || parsed <= 0 {
				return writeError(stdout, stderr, opts, 2, "usage", "--max-steps must be a positive integer")
			}
			maxSteps = parsed
		}
		run, err := forge.RunWorkflowDAG(projectPath, forge.DefaultWorkflowDAG(values["--question"]), forge.RunWorkflowOptions{MaxSteps: maxSteps, Actor: values["--actor"]})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "forge_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"workflowRun": run})
		}
		fmt.Fprintf(stdout, "workflow checkpoints: %d skipped: %d\n", len(run.Checkpoints), run.RestartSafeSkipped)
		return 0
	}
	var state forge.State
	var err error
	switch sub {
	case "init":
		state, err = forge.Init(projectPath, forge.InitOptions{Question: values["--question"], Actor: values["--actor"], SourceChoices: splitCSV(values["--sources"]), ToolChoices: splitCSV(values["--tools"])})
	case "status":
		state, err = forge.Status(projectPath)
	case "next":
		state, err = forge.Next(projectPath, values["--actor"])
	case "approve":
		state, err = forge.Approve(projectPath, forge.ApprovalInput{Gate: values["--gate"], Note: values["--note"], Actor: values["--actor"]})
	case "reopen":
		state, err = forge.Reopen(projectPath, forge.StateID(values["--state"]), values["--reason"], values["--actor"])
	case "replay":
		state, err = forge.Status(projectPath)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_forge_subcommand", fmt.Sprintf("unknown forge subcommand %q", sub))
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "forge_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"state": state, "reviewGates": forge.ReviewGates()})
	}
	fmt.Fprintf(stdout, "state: %s\nquestion: %s\n", state.CurrentState, state.Question)
	for _, gate := range state.BlockedReviewGates {
		fmt.Fprintf(stdout, "blocked gate: %s - %s\n", gate.Gate, gate.RequiredDecision)
	}
	for _, action := range state.NextSafeActions {
		fmt.Fprintf(stdout, "next: %s\n  %s\n", action.Label, action.CLI)
	}
	for _, receipt := range state.ValidationReceipts {
		fmt.Fprintf(stdout, "receipt: %s\n", receipt)
	}
	return 0
}

func parseForgeOptions(args []string) (string, map[string]string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project", "--question", "--sources", "--tools", "--gate", "--note", "--state", "--reason", "--actor", "--max-steps", "--out":
			if i+1 >= len(args) {
				return "", nil, false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", nil, false
		}
	}
	return values["--project"], values, true
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
	fmt.Fprintf(stdout, `_rforge_completion() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local commands="%s"
  COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
}
complete -F _rforge_completion rforge
`, strings.Join(topLevelCommands, " "))
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
	if !(len(args) == 1 && args[0] == "pdfs" || len(args) == 2 && args[0] == "pdfs" && args[1] == "--open-access-only") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge fetch pdfs [--open-access-only]")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for fetch pdfs")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
	}
	records, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_read_failed", err.Error())
	}
	result := fetchProjectPDFs(context.Background(), opts.Project, records)
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"assets": result.assets, "failed": len(result.failures), "failures": result.failures, "fetched": len(result.assets), "skipped": result.skipped})
	}
	fmt.Fprintf(stdout, "fetched %d PDFs; skipped %d; failed %d\n", len(result.assets), result.skipped, len(result.failures))
	return 0
}

type fetchPDFsResult struct {
	assets   []documents.DocumentAsset
	skipped  int
	failures []string
}

type pdfCandidate struct {
	paperID string
	url     string
	license string
	status  string
	arxiv   bool
}

func fetchProjectPDFs(ctx context.Context, projectPath string, records []library.PaperRecord) fetchPDFsResult {
	result := fetchPDFsResult{}
	for _, record := range records {
		candidates := paperPDFCandidates(record)
		if len(candidates) == 0 {
			result.skipped++
			continue
		}
		failures := []string{}
		for _, candidate := range candidates {
			asset, err := fetchPDFCandidate(ctx, projectPath, candidate)
			if err == nil {
				if err := writePDFDerivatives(ctx, projectPath, asset); err != nil {
					failures = append(failures, err.Error())
					continue
				}
				result.assets = append(result.assets, asset)
				failures = nil
				break
			}
			failures = append(failures, err.Error())
		}
		if len(failures) > 0 {
			result.failures = append(result.failures, paperRecordID(record)+": "+strings.Join(failures, "; "))
		}
	}
	return result
}

func fetchPDFCandidate(ctx context.Context, projectPath string, candidate pdfCandidate) (documents.DocumentAsset, error) {
	if candidate.arxiv {
		return documents.FetchArXivAsset(ctx, projectPath, candidate.paperID, candidate.url, "pdf")
	}
	return documents.FetchPDF(ctx, projectPath, candidate.paperID, documents.OpenAccessMetadata{OpenAccess: true, OAStatus: candidate.status, License: candidate.license, PDFURL: candidate.url})
}

func paperPDFCandidates(record library.PaperRecord) []pdfCandidate {
	out := []pdfCandidate{}
	seen := map[string]bool{}
	add := func(candidate pdfCandidate) {
		if candidate.paperID == "" || candidate.url == "" || seen[candidate.paperID+"\x00"+candidate.url] {
			return
		}
		if !candidate.arxiv && (strings.TrimSpace(candidate.license) == "" || !explicitPDFURL(candidate.url)) {
			return
		}
		seen[candidate.paperID+"\x00"+candidate.url] = true
		out = append(out, candidate)
	}
	for _, candidate := range sources.CompareOpenAccessCandidates([]library.PaperRecord{record}).Candidates {
		if candidate.Source == "local" {
			continue
		}
		if candidate.Source == "arxiv" {
			add(pdfCandidate{paperID: record.Identifiers.ArXivID, url: candidate.URL, arxiv: true})
			continue
		}
		add(pdfCandidate{paperID: paperRecordID(record), url: candidate.URL, license: candidate.License, status: candidate.OAStatus})
	}
	if record.OpenAccess && strings.TrimSpace(record.License) != "" {
		for _, rawURL := range record.URLs {
			add(pdfCandidate{paperID: paperRecordID(record), url: rawURL, license: record.License, status: "open"})
		}
	}
	return out
}

func writePDFDerivatives(ctx context.Context, projectPath string, asset documents.DocumentAsset) error {
	name := safeFetchName(asset.PaperID)
	textDir := filepath.Join(projectPath, "documents", "text")
	if err := os.MkdirAll(textDir, 0o755); err != nil {
		return err
	}
	text, err := exec.CommandContext(ctx, pdftotextCommand(), "-layout", asset.LocalPath, "-").Output()
	if err != nil {
		return fmt.Errorf("pdftotext %s: %w", asset.PaperID, err)
	}
	if err := os.WriteFile(filepath.Join(textDir, name+".txt"), text, 0o644); err != nil {
		return err
	}
	imageDir := filepath.Join(projectPath, "documents", "images", name)
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return err
	}
	prefix := filepath.Join(imageDir, "image")
	if err := exec.CommandContext(ctx, pdfimagesCommand(), "-all", asset.LocalPath, prefix).Run(); err != nil {
		return fmt.Errorf("pdfimages %s: %w", asset.PaperID, err)
	}
	return nil
}

func pdfimagesCommand() string {
	if cmd := os.Getenv("RFORGE_PDFIMAGES_CMD"); cmd != "" {
		return cmd
	}
	return "pdfimages"
}

func safeFetchName(value string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return "paper"
	}
	return strings.Join(parts, "-")
}

func explicitPDFURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	path := strings.ToLower(parsed.Path)
	return strings.HasSuffix(path, ".pdf") || strings.Contains(path, "/pdf/")
}

func paperRecordID(record library.PaperRecord) string {
	for _, value := range []string{record.Identifiers.DOI, record.Identifiers.ArXivID, record.Identifiers.PMID, record.Identifiers.PMCID, record.Identifiers.SemanticScholarID, record.Identifiers.OpenAlexID, record.Identifiers.CrossrefID, record.Title} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "paper"
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
	case "claim-panel":
		tracePath, outPath, ok := parseReportClaimPanel(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report claim-panel --trace <trace.json> --out <panel.json>")
		}
		var view report.CitationEvidenceTraceView
		if err := readJSONFile(tracePath, &view); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_trace_read_failed", err.Error())
		}
		panel := report.BuildClaimTraceabilityPanel(view)
		if err := writeJSONFile(outPath, panel); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_claim_panel_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"claimPanel": panel, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote claim traceability panel to %s\n", outPath)
		return 0
	case "final-export":
		inPath, panelPath, outPath, ok := parseReportFinalExport(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report final-export --in <report.md> --panel <panel.json> --out <final.md>")
		}
		var panel report.ClaimTraceabilityPanel
		if err := readJSONFile(panelPath, &panel); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_claim_panel_read_failed", err.Error())
		}
		if err := report.GuardFinalReportExport(panel); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_final_export_blocked", err.Error())
		}
		data, err := os.ReadFile(inPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "report_final_export_read_failed", err.Error())
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_final_export_write_failed", err.Error())
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_final_export_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"path": outPath})
		}
		fmt.Fprintf(stdout, "wrote final report export to %s\n", outPath)
		return 0
	case "trace":
		claimsPath, analysisPath, outPath, ok := parseReportTrace(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report trace --claims <queue.json> --analysis <run.json> --out <trace.json>")
		}
		var queue evidence.CitationLockedSuggestionQueue
		if err := readJSONFile(claimsPath, &queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_trace_claims_read_failed", err.Error())
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(analysisPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_trace_analysis_read_failed", err.Error())
		}
		var items []evidence.EvidenceItem
		_ = readJSONFile(evidenceItemsPath(opts.Project), &items)
		records := []library.PaperRecord{}
		if store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json")); err == nil {
			records, _ = store.List()
		}
		parsedDocs, _ := readParsedDocuments(filepath.Join(opts.Project, "parsed"))
		view := report.BuildCitationEvidenceTraceView(report.CitationEvidenceTraceInput{Claims: queue.Suggestions, EvidenceItems: items, AnalysisRun: run, ParsedDocuments: parsedDocs, LibraryRecords: records, PDFBaseURL: "/papers"})
		if err := writeJSONFile(outPath, view); err != nil {
			return writeError(stdout, stderr, opts, 1, "report_trace_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"trace": view, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote citation evidence trace to %s\n", outPath)
		return 0
	case "build":
		out, parsedPaths, ok := parseReportBuild(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge report build --out <file> [--parsed <parsed.json>]")
		}
		for _, path := range parsedPaths {
			var doc parsing.ParsedDocument
			if err := readJSONFile(path, &doc); err != nil {
				return writeError(stdout, stderr, opts, 1, "report_parsed_read_failed", err.Error())
			}
			data.PassageProvenance = append(data.PassageProvenance, report.BuildPassageProvenanceFromParsedDocuments([]parsing.ParsedDocument{doc}, path)...)
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

func parseReportClaimPanel(args []string) (string, string, bool) {
	if len(args) != 4 {
		return "", "", false
	}
	values := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		values[args[i]] = args[i+1]
	}
	return values["--trace"], values["--out"], values["--trace"] != "" && values["--out"] != ""
}

func parseReportFinalExport(args []string) (string, string, string, bool) {
	if len(args) != 6 {
		return "", "", "", false
	}
	values := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		values[args[i]] = args[i+1]
	}
	return values["--in"], values["--panel"], values["--out"], values["--in"] != "" && values["--panel"] != "" && values["--out"] != ""
}

func parseReportTrace(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--claims", "--analysis", "--out":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--claims"], values["--analysis"], values["--out"], values["--claims"] != "" && values["--analysis"] != "" && values["--out"] != ""
}

func parseReportBuild(args []string) (string, []string, bool) {
	values := map[string]string{}
	parsed := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out", "--parsed":
			if i+1 >= len(args) {
				return "", nil, false
			}
			if args[i] == "--parsed" {
				parsed = append(parsed, args[i+1])
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return "", nil, false
		}
	}
	return values["--out"], parsed, values["--out"] != ""
}

func executePackage(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package <create|fixture|audit|replay|archive|restore>")
	}
	switch args[0] {
	case "archive":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package archive <package-dir> <archive.tar>")
		}
		if err := reviewpkg.Archive(args[1], args[2]); err != nil {
			return writeError(stdout, stderr, opts, 1, "package_archive_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"ok": true, "path": args[2]})
		}
		fmt.Fprintf(stdout, "archived review package to %s\n", args[2])
		return 0
	case "restore":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package restore <archive.tar> <package-dir>")
		}
		if err := reviewpkg.Restore(args[1], args[2]); err != nil {
			return writeError(stdout, stderr, opts, 1, "package_restore_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"ok": true, "path": args[2]})
		}
		fmt.Fprintf(stdout, "restored review package to %s\n", args[2])
		return 0
	case "audit":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package audit <package-dir>")
		}
		report, err := reviewpkg.Audit(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "package_audit_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"audit": report, "ok": report.OK})
		}
		fmt.Fprintf(stdout, "package audit ok=%t checks=%d\n", report.OK, len(report.Checks))
		if !report.OK {
			return 1
		}
		return 0
	case "replay":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package replay <package-dir>")
		}
		report, err := reviewpkg.Replay(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "package_replay_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"replay": report, "ok": report.OK})
		}
		fmt.Fprintf(stdout, "package replay ok=%t checks=%d\n", report.OK, len(report.Checks))
		if !report.OK {
			return 1
		}
		return 0
	case "fixture":
		outPath, createdBy, question, ok := parsePackageCreate(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package fixture --out <dir> [--created-by <name> --question <text>]")
		}
		pkg, err := reviewpkg.CreateArtificialPhotosynthesisFixturePackage(outPath, reviewpkg.Options{CreatedBy: createdBy, Question: question})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "package_fixture_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"package": pkg, "path": outPath})
		}
		fmt.Fprintf(stdout, "created artificial photosynthesis fixture review package at %s\n", outPath)
		return 0
	case "create":
		if opts.Project == "" {
			return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required")
		}
		outPath, createdBy, question, ok := parsePackageCreate(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package create --out <dir> [--created-by <name> --question <text>]")
		}
		pkg, err := reviewpkg.Create(opts.Project, outPath, reviewpkg.Options{CreatedBy: createdBy, Question: question})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "package_create_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"package": pkg, "path": outPath})
		}
		fmt.Fprintf(stdout, "created review package at %s\n", outPath)
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge package create --out <dir> [--created-by <name> --question <text>]")
	}
}

func parsePackageCreate(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out", "--created-by", "--question":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--out"], values["--created-by"], values["--question"], values["--out"] != ""
}

func executeAnalysis(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) < 2 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis <prepare|run|sensitivity|ready|export>")
	}
	runPath := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+".json")
	switch args[0] {
	case "prepare":
		calc, varianceFloor, moderatorFields, ok := parseAnalysisEffect(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis prepare <run-id> [--effect smd|log-odds-ratio|risk-ratio|mean-difference|risk-difference|fisher-z-correlation|raw-continuous] [--variance-floor 0.0025] [--moderator <field>]")
		}
		var items []evidence.EvidenceItem
		if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
		}
		var run analysis.AnalysisRun
		var err error
		if _, isRaw := calc.(analysis.RawContinuousOutcome); isRaw {
			run, err = analysis.PrepareRawContinuous(args[1], items, varianceFloor, moderatorFields)
		} else {
			run, err = analysis.PrepareWithCalculator(args[1], items, calc)
		}
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
	case "ready":
		if len(args) < 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis ready <run-id> [--required field1,field2,...]")
		}
		var requiredFields []string
		for i := 2; i+1 < len(args); i++ {
			if args[i] == "--required" {
				for _, f := range strings.Split(args[i+1], ",") {
					if f = strings.TrimSpace(f); f != "" {
						requiredFields = append(requiredFields, f)
					}
				}
				break
			}
		}
		var items []evidence.EvidenceItem
		if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
		}
		report := analysis.BenchmarkingReadiness(args[1], items, requiredFields)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"readiness": report})
		}
		if report.Ready {
			fmt.Fprintf(stdout, "ready: %d/%d items pass all required fields\n", report.ReadyItems, report.TotalItems)
		} else {
			fmt.Fprintf(stdout, "not ready: %d items ready, %d issues\n", report.ReadyItems, len(report.Issues))
			for _, issue := range report.Issues {
				fmt.Fprintf(stdout, "  %s: missing %s\n", issue.PaperID, issue.MissingField)
			}
		}
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
		manifest := analysis.NewAnalysisArtifactManifest(run, result)
		if err := analysis.WriteAnalysisArtifactManifest(filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-artifact-manifest.json"), manifest); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_manifest_store_failed", err.Error())
		}
		// Auto-emit a no-floor sensitivity run when floor-imputed rows exist (ADR-0007).
		noFloor := analysis.ExcludeByViSource(run, "floor")
		if len(noFloor.InputRows) >= 2 && len(noFloor.InputRows) < len(run.InputRows) {
			fakeRunner := analysis.FakeRunner{Stdout: "I2=0\ntau2=0\nQ=0\n", Versions: map[string]string{"R": "fake", "metafor": "fake"}}
			sensResult, sensErr := analysis.RunMetafor(filepath.Join(opts.Project, "analysis"), noFloor, fakeRunner)
			if sensErr == nil {
				_ = writeJSONFile(filepath.Join(opts.Project, "analysis", safeFileStem(noFloor.ID)+"-result.json"), sensResult)
			}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"result": result})
		}
		fmt.Fprintln(stdout, "ran analysis")
		return 0
	case "moderators":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis moderators <run-id>")
		}
		var items []evidence.EvidenceItem
		if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
		}
		preview := analysis.ModeratorPreviewFromEvidence(items)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"moderators": preview})
		}
		for _, field := range preview.Fields {
			fmt.Fprintf(stdout, "%s\t%d papers\tnumeric=%t\n", field.Name, field.Papers, field.Numeric)
		}
		return 0
	case "method-workbench":
		category, methods, selection, outPath, ok := parseMethodWorkbenchArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis method-workbench <run-id> --category <name> --out <report.json> [--method <name> --select <method> --reviewer <name> --reason <text>]")
		}
		workbench := analysis.DefaultMethodComparisonWorkbench()
		report := workbench.CompareWithSelection(category, methods, selection)
		if err := writeJSONFile(outPath, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_method_workbench_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"methodComparison": report, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote method comparison workbench to %s\n", outPath)
		return 0
	case "engine-compare":
		outPath, secondaryDelta, tolerance, ok := parseEngineCompareArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis engine-compare <run-id> --out <report.json> [--secondary-delta N --tolerance N]")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		var result analysis.AnalysisResult
		_ = readJSONFile(filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-result.json"), &result)
		primary, err := analysis.BuildMetaforFixtureResult(run, result)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_metafor_fixture_failed", err.Error())
		}
		secondary, err := analysis.BuildPyMAREFixtureResult(run, secondaryDelta)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_pymare_fixture_failed", err.Error())
		}
		report := analysis.CompareAnalysisEngines(run, primary, secondary, tolerance)
		if err := writeJSONFile(outPath, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_engine_compare_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"engineComparison": report, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote engine comparison report to %s\n", outPath)
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
		variable, groups, evidenceField, ok := parseSubgroupArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis subgroup <run-id> --variable <name> [--group <paper>=<group> | --from-evidence <field>]")
		}
		if evidenceField != "" {
			var items []evidence.EvidenceItem
			if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
				return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
			}
			derived, err := analysis.SubgroupValuesFromEvidence(items, evidenceField)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "analysis_subgroup_moderators_failed", err.Error())
			}
			groups = derived
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
		moderator, values, evidenceField, ok := parseMetaRegressionArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis meta-regression <run-id> --moderator <name> [--value <paper>=<number> | --from-evidence <field>]")
		}
		if evidenceField != "" {
			var items []evidence.EvidenceItem
			if err := readJSONFile(evidenceItemsPath(opts.Project), &items); err != nil {
				return writeError(stdout, stderr, opts, 1, "analysis_evidence_read_failed", err.Error())
			}
			derived, err := analysis.MetaRegressionValuesFromEvidence(items, evidenceField)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "analysis_meta_regression_moderators_failed", err.Error())
			}
			values = derived
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
	case "influence":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis influence <run-id>")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		report, err := analysis.InfluenceDiagnostics(run)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_influence_failed", err.Error())
		}
		path := filepath.Join(opts.Project, "analysis", safeFileStem(args[1])+"-influence.json")
		if err := writeJSONFile(path, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_influence_store_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"influence": report, "path": path})
		}
		fmt.Fprintf(stdout, "wrote influence diagnostics to %s\n", path)
		return 0
	case "bayesian":
		method, priorMean, priorVariance, ok := parseBayesianArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis bayesian <run-id> --method normal-approx|grid [--prior-mean 0] [--prior-variance 100]")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		var report analysis.BayesianReport
		var err error
		if method == "grid" || method == "grid-engine" {
			report, err = analysis.RunBayesianEngine(run, analysis.GridBayesianEngine{}, analysis.BayesianEngineOptions{PriorMean: priorMean, PriorVariance: priorVariance})
		} else {
			report, err = analysis.BayesianNormalApproximation(run, priorMean, priorVariance)
		}
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
		method, ok := parsePublicationBiasArgs(args[2:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge analysis publication-bias <run-id> --method egger|begg")
		}
		var run analysis.AnalysisRun
		if err := readJSONFile(runPath, &run); err != nil {
			return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
		}
		var report analysis.PublicationBiasReport
		var err error
		if method == "begg" || method == "begg-rank-correlation" {
			report, err = analysis.BeggRankCorrelation(run)
		} else {
			report, err = analysis.EggerRegression(run)
		}
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

func parseMethodWorkbenchArgs(args []string) (string, []string, analysis.MethodSelectionInput, string, bool) {
	category := ""
	methods := []string{}
	selection := analysis.MethodSelectionInput{}
	out := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--category", "--method", "--select", "--reviewer", "--reason", "--out":
			if i+1 >= len(args) {
				return "", nil, analysis.MethodSelectionInput{}, "", false
			}
			switch args[i] {
			case "--category":
				category = args[i+1]
			case "--method":
				methods = append(methods, args[i+1])
			case "--select":
				selection.SelectedMethod = args[i+1]
			case "--reviewer":
				selection.Reviewer = args[i+1]
			case "--reason":
				selection.Reason = args[i+1]
			case "--out":
				out = args[i+1]
			}
			i++
		default:
			return "", nil, analysis.MethodSelectionInput{}, "", false
		}
	}
	return category, methods, selection, out, strings.TrimSpace(category) != "" && out != ""
}

func parseEngineCompareArgs(args []string) (string, float64, float64, bool) {
	out := ""
	secondaryDelta := 0.0
	tolerance := 1e-6
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 >= len(args) {
				return "", 0, 0, false
			}
			out = args[i+1]
			i++
		case "--secondary-delta":
			if i+1 >= len(args) {
				return "", 0, 0, false
			}
			parsed, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return "", 0, 0, false
			}
			secondaryDelta = parsed
			i++
		case "--tolerance":
			if i+1 >= len(args) {
				return "", 0, 0, false
			}
			parsed, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil || parsed <= 0 {
				return "", 0, 0, false
			}
			tolerance = parsed
			i++
		default:
			return "", 0, 0, false
		}
	}
	return out, secondaryDelta, tolerance, out != ""
}

func parsePublicationBiasArgs(args []string) (string, bool) {
	if len(args) != 2 || args[0] != "--method" {
		return "", false
	}
	switch args[1] {
	case "egger", "begg", "begg-rank-correlation":
		return args[1], true
	default:
		return "", false
	}
}

func parseBayesianArgs(args []string) (string, float64, float64, bool) {
	priorMean := 0.0
	priorVariance := 100.0
	method := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--method":
			if i+1 >= len(args) || (args[i+1] != "normal-approx" && args[i+1] != "grid" && args[i+1] != "grid-engine") {
				return "", 0, 0, false
			}
			method = args[i+1]
			i++
		case "--prior-mean":
			if i+1 >= len(args) {
				return "", 0, 0, false
			}
			parsed, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return "", 0, 0, false
			}
			priorMean = parsed
			i++
		case "--prior-variance":
			if i+1 >= len(args) {
				return "", 0, 0, false
			}
			parsed, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil || parsed <= 0 {
				return "", 0, 0, false
			}
			priorVariance = parsed
			i++
		default:
			return "", 0, 0, false
		}
	}
	return method, priorMean, priorVariance, method != ""
}

func parseSubgroupArgs(args []string) (string, map[string]string, string, bool) {
	variable := ""
	groups := map[string]string{}
	evidenceField := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--variable":
			if i+1 >= len(args) {
				return "", nil, "", false
			}
			variable = args[i+1]
			i++
		case "--group":
			if i+1 >= len(args) {
				return "", nil, "", false
			}
			paper, group, ok := strings.Cut(args[i+1], "=")
			if !ok || strings.TrimSpace(paper) == "" || strings.TrimSpace(group) == "" {
				return "", nil, "", false
			}
			groups[strings.TrimSpace(paper)] = strings.TrimSpace(group)
			i++
		case "--from-evidence":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", nil, "", false
			}
			evidenceField = args[i+1]
			i++
		default:
			return "", nil, "", false
		}
	}
	return variable, groups, evidenceField, strings.TrimSpace(variable) != "" && (len(groups) > 0 || strings.TrimSpace(evidenceField) != "")
}

func parseMetaRegressionArgs(args []string) (string, map[string]float64, string, bool) {
	moderator := ""
	values := map[string]float64{}
	evidenceField := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--moderator":
			if i+1 >= len(args) {
				return "", nil, "", false
			}
			moderator = args[i+1]
			i++
		case "--value":
			if i+1 >= len(args) {
				return "", nil, "", false
			}
			paper, raw, ok := strings.Cut(args[i+1], "=")
			if !ok || strings.TrimSpace(paper) == "" {
				return "", nil, "", false
			}
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return "", nil, "", false
			}
			values[strings.TrimSpace(paper)] = parsed
			i++
		case "--from-evidence":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", nil, "", false
			}
			evidenceField = args[i+1]
			i++
		default:
			return "", nil, "", false
		}
	}
	return moderator, values, evidenceField, strings.TrimSpace(moderator) != "" && (len(values) > 0 || strings.TrimSpace(evidenceField) != "")
}

func parseAnalysisEffect(args []string) (analysis.EffectSizeCalculator, float64, []string, bool) {
	if len(args) == 0 {
		return analysis.StandardizedMeanDifference{}, 0, nil, true
	}
	if len(args) < 2 || args[0] != "--effect" {
		return nil, 0, nil, false
	}
	effectName := args[1]
	remaining := args[2:]

	// Parse optional --variance-floor and --moderator flags (only meaningful for raw-continuous).
	varianceFloor := 0.0025
	var moderatorFields []string
	var unrecognized []string
	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--variance-floor":
			if i+1 >= len(remaining) {
				return nil, 0, nil, false
			}
			f, err := strconv.ParseFloat(remaining[i+1], 64)
			if err != nil || f <= 0 {
				return nil, 0, nil, false
			}
			varianceFloor = f
			i++
		case "--moderator":
			if i+1 >= len(remaining) || remaining[i+1] == "" {
				return nil, 0, nil, false
			}
			moderatorFields = append(moderatorFields, remaining[i+1])
			i++
		default:
			unrecognized = append(unrecognized, remaining[i])
		}
	}
	// For arm-pair calculators, no extra flags are allowed.
	if (len(unrecognized) != 0 || len(moderatorFields) != 0) && effectName != "raw-continuous" {
		return nil, 0, nil, false
	}
	switch effectName {
	case "smd", "standardized-mean-difference":
		return analysis.StandardizedMeanDifference{}, 0, nil, true
	case "log-odds-ratio":
		return analysis.LogOddsRatio{}, 0, nil, true
	case "risk-ratio", "rr":
		return analysis.RiskRatio{}, 0, nil, true
	case "mean-difference", "md":
		return analysis.MeanDifference{}, 0, nil, true
	case "risk-difference", "rd":
		return analysis.RiskDifference{}, 0, nil, true
	case "fisher-z-correlation", "fisher-z", "correlation":
		return analysis.FisherZCorrelation{}, 0, nil, true
	case "raw-continuous":
		return analysis.RawContinuousOutcome{VarianceFloor: varianceFloor}, varianceFloor, moderatorFields, true
	default:
		return nil, 0, nil, false
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
	if len(args) == 0 || opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence <audit|risk-bias-templates|risk-bias-suggest|risk-bias-review>")
	}
	switch args[0] {
	case "audit":
		if len(args) != 1 {
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
	case "gaps":
		gapArgs, ok := parseEvidenceGaps(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence gaps --out <report.json> [--question <text> --screened-in <paper-id> --parsed-paper <paper-id> --outcome <name> --comparator <name> --full-text <paper-id> --available-full-text <paper-id> --claims <queue.json> --analysis <run.json>]")
		}
		var items []evidence.EvidenceItem
		_ = readJSONFile(evidenceItemsPath(opts.Project), &items)
		claims := []evidence.CitationLockedSuggestion{}
		if gapArgs.ClaimsPath != "" {
			var queue evidence.CitationLockedSuggestionQueue
			if err := readJSONFile(gapArgs.ClaimsPath, &queue); err != nil {
				return writeError(stdout, stderr, opts, 1, "claims_read_failed", err.Error())
			}
			claims = queue.Suggestions
		}
		included := []string{}
		if gapArgs.AnalysisPath != "" {
			var run analysis.AnalysisRun
			if err := readJSONFile(gapArgs.AnalysisPath, &run); err != nil {
				return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
			}
			for _, row := range run.InputRows {
				included = append(included, row.PaperID)
			}
		}
		report := evidence.AnalyzeEvidenceGaps(evidence.EvidenceGapAnalysisInput{ResearchQuestion: gapArgs.Question, ScreenedInPaperIDs: gapArgs.ScreenedInPaperIDs, ParsedPassagePaperIDs: gapArgs.ParsedPaperIDs, Items: items, Claims: claims, RequiredOutcomes: gapArgs.Outcomes, RequiredComparators: gapArgs.Comparators, FullTextRequiredPaperIDs: gapArgs.FullTextRequired, AvailableFullTextPaperIDs: gapArgs.FullTextAvailable, AnalysisIncludedPaperIDs: included})
		if err := writeJSONFile(gapArgs.OutPath, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "evidence_gaps_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"gaps": report, "path": gapArgs.OutPath})
		}
		fmt.Fprintf(stdout, "wrote evidence gap report with %d gaps to %s\n", len(report.Gaps), gapArgs.OutPath)
		return 0
	case "grid":
		parsedPaths, analysisPath, outPath, ok := parseEvidenceGrid(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence grid --out <grid.json> [--parsed <parsed.json>] [--analysis <run.json>]")
		}
		var items []evidence.EvidenceItem
		_ = readJSONFile(evidenceItemsPath(opts.Project), &items)
		docs := []parsing.ParsedDocument{}
		for _, path := range parsedPaths {
			var doc parsing.ParsedDocument
			if err := readJSONFile(path, &doc); err != nil {
				return writeError(stdout, stderr, opts, 1, "parsed_read_failed", err.Error())
			}
			docs = append(docs, doc)
		}
		included := []string{}
		if analysisPath != "" {
			var run analysis.AnalysisRun
			if err := readJSONFile(analysisPath, &run); err != nil {
				return writeError(stdout, stderr, opts, 1, "analysis_read_failed", err.Error())
			}
			for _, row := range run.InputRows {
				included = append(included, row.PaperID)
			}
		}
		grid := evidence.BuildExtractionGrid(evidence.ExtractionGridInput{Items: items, ParsedDocuments: docs, AnalysisIncludedPaperIDs: included, PDFBaseURL: "/papers"})
		if err := writeJSONFile(outPath, grid); err != nil {
			return writeError(stdout, stderr, opts, 1, "evidence_grid_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"grid": grid, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote evidence extraction grid with %d rows to %s\n", len(grid.Rows), outPath)
		return 0
	case "citation-suggest":
		request, outPath, ok := parseCitationSuggest(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence citation-suggest --paper <id> --support <ref=text> --out <queue.json> [--kind extraction|report_prose --prompt <text> --model <name> --version <version>]")
		}
		queue, err := evidence.DraftCitationLockedSuggestions(request)
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "citation_suggest_invalid", err.Error())
		}
		if err := writeJSONFile(outPath, queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_queue_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": queue, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote %d citation-locked suggestions to %s\n", len(queue.Suggestions), outPath)
		return 0
	case "citation-review":
		queuePath, input, outPath, ok := parseCitationReview(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence citation-review --queue <queue.json> --id <suggestion-id> --decision accepted|rejected|corrected --reviewer <name> --out <queue.json> [--note <text>]")
		}
		var queue evidence.CitationLockedSuggestionQueue
		if err := readJSONFile(queuePath, &queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_queue_read_failed", err.Error())
		}
		reviewed, err := evidence.ReviewCitationLockedSuggestion(queue, input)
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "citation_review_invalid", err.Error())
		}
		if err := writeJSONFile(outPath, reviewed); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_review_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": reviewed, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote reviewed citation-locked queue to %s\n", outPath)
		return 0
	case "entity-suggest":
		parsedPath, model, version, outPath, ok := parseEntitySuggest(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence entity-suggest --parsed <parsed.json> --out <queue.json> [--model <name> --version <version>]")
		}
		var doc parsing.ParsedDocument
		if err := readJSONFile(parsedPath, &doc); err != nil {
			return writeError(stdout, stderr, opts, 1, "parsed_read_failed", err.Error())
		}
		passages := []parsing.Passage{}
		for _, section := range doc.Sections {
			passages = append(passages, section.Passages...)
		}
		queue := evidence.DraftScientificEntitySuggestions(evidence.ScientificEntitySuggestionRequest{PaperID: doc.PaperID, Passages: passages, ModelName: model, ModelVersion: version})
		if err := writeJSONFile(outPath, queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "entity_queue_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": queue, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote %d scientific entity suggestions to %s\n", len(queue.Suggestions), outPath)
		return 0
	case "entity-review":
		queuePath, input, outPath, ok := parseEntityReview(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence entity-review --queue <queue.json> --id <suggestion-id> --decision accepted|rejected|corrected --reviewer <name> --out <queue.json> [--note <text>]")
		}
		var queue evidence.ScientificEntitySuggestionQueue
		if err := readJSONFile(queuePath, &queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "entity_queue_read_failed", err.Error())
		}
		reviewed, err := evidence.ReviewScientificEntitySuggestion(queue, input)
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "entity_review_invalid", err.Error())
		}
		if err := writeJSONFile(outPath, reviewed); err != nil {
			return writeError(stdout, stderr, opts, 1, "entity_review_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": reviewed, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote reviewed scientific entity queue to %s\n", outPath)
		return 0
	case "risk-bias-templates":
		outPath, ok := parseSingleFlag(args[1:], "--out")
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence risk-bias-templates --out <templates.json>")
		}
		templates := evidence.DefaultRiskOfBiasSchemaTemplates()
		if err := writeJSONFile(outPath, templates); err != nil {
			return writeError(stdout, stderr, opts, 1, "risk_bias_templates_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"templates": templates, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote %d risk-of-bias templates to %s\n", len(templates), outPath)
		return 0
	case "risk-bias-suggest":
		request, outPath, ok := parseRiskBiasSuggest(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence risk-bias-suggest --paper <id> --support <ref=text> --out <queue.json> [--model <name> --version <version>]")
		}
		queue := evidence.DraftRiskOfBiasSuggestionQueue(request)
		if err := writeJSONFile(outPath, queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "risk_bias_queue_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": queue, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote %d risk-of-bias suggestions to %s\n", len(queue.Suggestions), outPath)
		return 0
	case "risk-bias-review":
		queuePath, input, outPath, ok := parseRiskBiasReview(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence risk-bias-review --queue <queue.json> --id <suggestion-id> --decision accepted|rejected|corrected --reviewer <name> --out <queue.json> [--note <text>]")
		}
		var queue evidence.RiskOfBiasSuggestionQueue
		if err := readJSONFile(queuePath, &queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "risk_bias_queue_read_failed", err.Error())
		}
		reviewed, err := evidence.ReviewRiskOfBiasSuggestion(queue, input)
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "risk_bias_review_invalid", err.Error())
		}
		if err := writeJSONFile(outPath, reviewed); err != nil {
			return writeError(stdout, stderr, opts, 1, "risk_bias_review_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": reviewed, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote reviewed risk-of-bias queue to %s\n", outPath)
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge evidence <audit|risk-bias-templates|risk-bias-suggest|risk-bias-review>")
	}
}

type evidenceGapsArgs struct {
	Question           string
	ScreenedInPaperIDs []string
	ParsedPaperIDs     []string
	Outcomes           []string
	Comparators        []string
	FullTextRequired   []string
	FullTextAvailable  []string
	ClaimsPath         string
	AnalysisPath       string
	OutPath            string
}

func parseEvidenceGaps(args []string) (evidenceGapsArgs, bool) {
	parsed := evidenceGapsArgs{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--question", "--screened-in", "--parsed-paper", "--outcome", "--comparator", "--full-text", "--available-full-text", "--claims", "--analysis", "--out":
			if i+1 >= len(args) {
				return evidenceGapsArgs{}, false
			}
			switch args[i] {
			case "--question":
				parsed.Question = args[i+1]
			case "--screened-in":
				parsed.ScreenedInPaperIDs = append(parsed.ScreenedInPaperIDs, args[i+1])
			case "--parsed-paper":
				parsed.ParsedPaperIDs = append(parsed.ParsedPaperIDs, args[i+1])
			case "--outcome":
				parsed.Outcomes = append(parsed.Outcomes, args[i+1])
			case "--comparator":
				parsed.Comparators = append(parsed.Comparators, args[i+1])
			case "--full-text":
				parsed.FullTextRequired = append(parsed.FullTextRequired, args[i+1])
			case "--available-full-text":
				parsed.FullTextAvailable = append(parsed.FullTextAvailable, args[i+1])
			case "--claims":
				parsed.ClaimsPath = args[i+1]
			case "--analysis":
				parsed.AnalysisPath = args[i+1]
			case "--out":
				parsed.OutPath = args[i+1]
			}
			i++
		default:
			return evidenceGapsArgs{}, false
		}
	}
	return parsed, parsed.OutPath != ""
}

func parseEvidenceGrid(args []string) ([]string, string, string, bool) {
	parsed := []string{}
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--analysis", "--out":
			if i+1 >= len(args) {
				return nil, "", "", false
			}
			if args[i] == "--parsed" {
				parsed = append(parsed, args[i+1])
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return nil, "", "", false
		}
	}
	return parsed, values["--analysis"], values["--out"], values["--out"] != ""
}

func parseCitationSuggest(args []string) (evidence.CitationLockedSuggestionRequest, string, bool) {
	values := map[string]string{"--kind": string(evidence.CitationLockedExtraction)}
	supports := []evidence.CitationLockedSupport{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--kind", "--prompt", "--support", "--model", "--version", "--out":
			if i+1 >= len(args) {
				return evidence.CitationLockedSuggestionRequest{}, "", false
			}
			if args[i] == "--support" {
				parts := strings.SplitN(args[i+1], "=", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return evidence.CitationLockedSuggestionRequest{}, "", false
				}
				supports = append(supports, evidence.CitationLockedSupport{Ref: parts[0], ExactText: parts[1]})
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return evidence.CitationLockedSuggestionRequest{}, "", false
		}
	}
	request := evidence.CitationLockedSuggestionRequest{PaperID: values["--paper"], Kind: evidence.CitationLockedSuggestionKind(values["--kind"]), Prompt: values["--prompt"], Supports: supports, ModelName: values["--model"], ModelVersion: values["--version"]}
	return request, values["--out"], request.PaperID != "" && len(supports) > 0 && values["--out"] != ""
}

func parseCitationReview(args []string) (string, evidence.CitationLockedReviewInput, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--queue", "--id", "--decision", "--reviewer", "--note", "--out":
			if i+1 >= len(args) {
				return "", evidence.CitationLockedReviewInput{}, "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", evidence.CitationLockedReviewInput{}, "", false
		}
	}
	input := evidence.CitationLockedReviewInput{SuggestionID: values["--id"], Decision: evidence.Status(values["--decision"]), Reviewer: values["--reviewer"], Note: values["--note"]}
	return values["--queue"], input, values["--out"], values["--queue"] != "" && values["--id"] != "" && values["--decision"] != "" && values["--reviewer"] != "" && values["--out"] != ""
}

func parseEntitySuggest(args []string) (string, string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--model", "--version", "--out":
			if i+1 >= len(args) {
				return "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", false
		}
	}
	return values["--parsed"], values["--model"], values["--version"], values["--out"], values["--parsed"] != "" && values["--out"] != ""
}

func parseEntityReview(args []string) (string, evidence.ScientificEntityReviewInput, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--queue", "--id", "--decision", "--reviewer", "--note", "--out":
			if i+1 >= len(args) {
				return "", evidence.ScientificEntityReviewInput{}, "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", evidence.ScientificEntityReviewInput{}, "", false
		}
	}
	input := evidence.ScientificEntityReviewInput{SuggestionID: values["--id"], Decision: evidence.EntitySuggestionStatus(values["--decision"]), Reviewer: values["--reviewer"], Note: values["--note"]}
	return values["--queue"], input, values["--out"], values["--queue"] != "" && values["--id"] != "" && values["--decision"] != "" && values["--reviewer"] != "" && values["--out"] != ""
}

func parseRiskBiasSuggest(args []string) (evidence.RiskOfBiasSuggestionRequest, string, bool) {
	values := map[string]string{}
	passages := []evidence.SupportText{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--support", "--model", "--version", "--out":
			if i+1 >= len(args) {
				return evidence.RiskOfBiasSuggestionRequest{}, "", false
			}
			if args[i] == "--support" {
				parts := strings.SplitN(args[i+1], "=", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return evidence.RiskOfBiasSuggestionRequest{}, "", false
				}
				passages = append(passages, evidence.SupportText{Ref: parts[0], Text: parts[1]})
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return evidence.RiskOfBiasSuggestionRequest{}, "", false
		}
	}
	request := evidence.RiskOfBiasSuggestionRequest{PaperID: values["--paper"], Passages: passages, ModelName: values["--model"], ModelVersion: values["--version"]}
	return request, values["--out"], request.PaperID != "" && len(passages) > 0 && values["--out"] != ""
}

func parseRiskBiasReview(args []string) (string, evidence.RiskOfBiasReviewInput, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--queue", "--id", "--decision", "--reviewer", "--note", "--out":
			if i+1 >= len(args) {
				return "", evidence.RiskOfBiasReviewInput{}, "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", evidence.RiskOfBiasReviewInput{}, "", false
		}
	}
	input := evidence.RiskOfBiasReviewInput{SuggestionID: values["--id"], Decision: evidence.RiskOfBiasSuggestionStatus(values["--decision"]), Reviewer: values["--reviewer"], Note: values["--note"]}
	return values["--queue"], input, values["--out"], values["--queue"] != "" && values["--id"] != "" && values["--decision"] != "" && values["--reviewer"] != "" && values["--out"] != ""
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
	// Dir-based CSV workflow — works without a project.
	if len(args) >= 1 {
		switch args[0] {
		case "queue":
			if screenDirHasFlag(args[1:], "--dir") {
				return executeScreenDirQueue(args[1:], stdout, stderr, opts)
			}
		case "import":
			return executeScreenDirImport(args[1:], stdout, stderr, opts)
		case "progress":
			if screenDirHasFlag(args[1:], "--dir") {
				return executeScreenDirProgress(args[1:], stdout, stderr, opts)
			}
		}
	}
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
	case "assign":
		stage, reviewers, perRecord, outPath, ok := parseScreenAssign(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen assign --stage <stage> --reviewer <name> [--reviewer <name>] [--per-record N] --out <assignments.json>")
		}
		records, err := screeningRecordsFromLibrary(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		assignments := screening.AssignReviewers(records, reviewers, perRecord)
		for i := range assignments {
			assignments[i].Stage = stage
		}
		if err := writeJSONFile(outPath, assignments); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_assign_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"assignments": assignments, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote %d reviewer assignments to %s\n", len(assignments), outPath)
		return 0
	case "panel":
		stage, outPath, ok := parseScreenPanel(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen panel --stage <stage> --out <panel.json>")
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		panel := screening.BuildConflictAdjudicationPanel(events, stage)
		if err := writeJSONFile(outPath, panel); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_panel_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"panel": panel, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote conflict/adjudication panel to %s\n", outPath)
		return 0
	case "audit-bundle":
		stage, assignmentsPath, activeRunPath, outPath, ok := parseScreenAuditBundle(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen audit-bundle --stage <stage> --out <bundle.json> [--assignments <assignments.json> --active-run <run.json>]")
		}
		records, err := screeningRecordsFromLibrary(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		_, events, err := loadScreening(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_load_failed", err.Error())
		}
		assignments := []screening.ReviewerAssignment{}
		if assignmentsPath != "" {
			if err := readJSONFile(assignmentsPath, &assignments); err != nil {
				return writeError(stdout, stderr, opts, 1, "screen_assignments_read_failed", err.Error())
			}
		}
		var activeRun screening.ActiveLearningRun
		if activeRunPath != "" {
			if err := readJSONFile(activeRunPath, &activeRun); err != nil {
				return writeError(stdout, stderr, opts, 1, "screen_active_run_read_failed", err.Error())
			}
		}
		bundle := screening.BuildScreeningAuditBundle(screening.ScreeningAuditBundleInput{Records: records, Events: events, Assignments: assignments, Stage: stage, ActiveRun: activeRun})
		if err := writeJSONFile(outPath, bundle); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_audit_bundle_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"auditBundle": bundle, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote screening audit bundle to %s\n", outPath)
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
func parseScreenAssign(args []string) (screening.Stage, []string, int, string, bool) {
	values := map[string]string{}
	reviewers := []string{}
	perRecord := 1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stage", "--reviewer", "--per-record", "--out":
			if i+1 >= len(args) {
				return "", nil, 0, "", false
			}
			switch args[i] {
			case "--reviewer":
				reviewers = append(reviewers, args[i+1])
			case "--per-record":
				parsed, err := strconv.Atoi(args[i+1])
				if err != nil || parsed <= 0 {
					return "", nil, 0, "", false
				}
				perRecord = parsed
			default:
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return "", nil, 0, "", false
		}
	}
	return screening.Stage(values["--stage"]), reviewers, perRecord, values["--out"], values["--stage"] != "" && values["--out"] != "" && len(reviewers) > 0
}

func parseScreenPanel(args []string) (screening.Stage, string, bool) {
	if len(args) != 4 {
		return "", "", false
	}
	values := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		values[args[i]] = args[i+1]
	}
	return screening.Stage(values["--stage"]), values["--out"], values["--stage"] != "" && values["--out"] != ""
}

func parseScreenAuditBundle(args []string) (screening.Stage, string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stage", "--assignments", "--active-run", "--out":
			if i+1 >= len(args) {
				return "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", false
		}
	}
	return screening.Stage(values["--stage"]), values["--assignments"], values["--active-run"], values["--out"], values["--stage"] != "" && values["--out"] != ""
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
func screeningRecordsFromLibrary(project string) ([]screening.ScreeningRecord, error) {
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		return nil, err
	}
	papers, err := store.List()
	if err != nil {
		return nil, err
	}
	records := make([]screening.ScreeningRecord, 0, len(papers))
	for _, paper := range papers {
		records = append(records, screening.ScreeningRecord{ID: screeningPaperID(paper), Title: paper.Title, Abstract: paper.Abstract})
	}
	return records, nil
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
	content := fmt.Sprintf("default_project_path = %q\n", filepath.Base(proj.Path))
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
	fmt.Fprintln(w, "  rforge automation policy [--action <action>]")
	fmt.Fprintln(w, "  rforge search --source openalex|arxiv|crossref|semantic-scholar|europepmc|pubmed --query <query> [--category arxiv-category] [--filter source-filter] [--from-year YYYY] [--to-year YYYY] [--type article] [--open-access true|false] [--concept C41008148]")
	fmt.Fprintln(w, "  rforge search import --source openalex --query <query> --pages N [--resume-state state.json]")
	fmt.Fprintln(w, "  rforge --project <path> search batch --queries <file> --sources <preset|list> --out <dir> [--fetch-pdfs]")
	fmt.Fprintln(w, "  rforge citations expand --source semantic-scholar|openalex|crossref --paper <id> --direction references|citations|both --depth N [--max-records N] --out <file> [--import-library]")
	fmt.Fprintln(w, "  rforge citations report --graph <graph.json> --out <report.md>")
	fmt.Fprintln(w, "  rforge oa lookup <doi>")
	fmt.Fprintln(w, "  rforge oa resolve-plan <doi>")
	fmt.Fprintln(w, "  rforge oa sources")
	fmt.Fprintln(w, "  rforge service check <name>")
	fmt.Fprintln(w, "  rforge library list|refresh-doi|refresh-crossref")
	fmt.Fprintln(w, "  rforge fetch pdfs")
	fmt.Fprintln(w, "  rforge duplicate report")
	fmt.Fprintln(w, "  rforge import json|csv|bibtex|ris|csl-json|zotero-rdf <file>")
	fmt.Fprintln(w, "  rforge export json|csv|bibtex|ris|csl-json|zotero-rdf <file>")
	fmt.Fprintln(w, "  rforge oss inventory-check <manifest.json>")
	fmt.Fprintln(w, "  rforge oss inventory-refresh <manifest.json> --source github [--base-url <url>]")
	fmt.Fprintln(w, "  rforge oss inventory-policy <manifest.json> [--stale-after 18mo]")
	fmt.Fprintln(w, "  rforge oss inventory-drift <manifest.json>")
	fmt.Fprintln(w, "  rforge oss inventory-roadmap <manifest.json> --todo TODO.md")
	fmt.Fprintln(w, "  rforge oss inventory-report <manifest.json> [--area <area>]")
	fmt.Fprintln(w, "  rforge oss add|list|license-check")
	fmt.Fprintln(w, "  rforge parse --paper <id> --parser grobid|tex|s2orc|papermage --pdf|--tex|--s2orc|--papermage <file>")
	fmt.Fprintln(w, "  rforge knowledge query --project <path> [--term <text>]")
	fmt.Fprintln(w, "  rforge knowledge path --project <path> --from <node-id> --to <node-id>")
	fmt.Fprintln(w, "  rforge research acquire-pdftotext --doi <doi> --pdf-url <url> --license <license> --oa-status <status> --out <parsed.json>")
	fmt.Fprintln(w, "  rforge research parse-pdftotext --paper <id> --pdf <file> --out <parsed.json> [--title <title>]")
	fmt.Fprintln(w, "  rforge research screen-queue --out <queue.csv> [--markdown <queue.md>] [--library <library.json>] [--search-results <dir>]")
	fmt.Fprintln(w, "  rforge research leakage-audit (--parsed <parsed-dir> | --text <text-dir>) --out <audit.json> [--markdown <audit.md>]")
	fmt.Fprintln(w, "  rforge graph papers")
	fmt.Fprintln(w, "  rforge project create [path] --title <title>")
	fmt.Fprintln(w, "  rforge project discover-assets")
	fmt.Fprintln(w, "  rforge project inspect <path>")
	fmt.Fprintln(w, "  rforge project list <root>")
	fmt.Fprintln(w, "  rforge decisions --check TODO.md")
	fmt.Fprintln(w, "  rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md")
}
