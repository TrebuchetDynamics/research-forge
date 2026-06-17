package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/protocol"
)

func executeProtocol(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 1 && args[0] == "capabilities" {
		return writeCapabilities(protocol.DefaultConnectorCapabilityRegistry(), stdout, opts)
	}
	if len(args) > 0 && args[0] == "live-smoke-snapshot" {
		return executeLiveSmokeSnapshot(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "suggest-expansions" {
		return executeSuggestExpansions(args[1:], stdout, stderr, opts)
	}
	if len(args) == 0 || (args[0] != "compile" && args[0] != "plan-sources") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge protocol compile|plan-sources|capabilities --question <text> [--type pico|peco|spider|freeform] [framework flags]")
	}
	values, err := parseProtocolCompileFlags(args[1:])
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	plan, err := protocol.CompileQuestion(protocol.QuestionInput{
		Framework:    values["--type"],
		Question:     values["--question"],
		Population:   values["--population"],
		Intervention: values["--intervention"],
		Comparator:   values["--comparator"],
		Outcome:      values["--outcome"],
		Exposure:     values["--exposure"],
		Sample:       values["--sample"],
		Phenomenon:   values["--phenomenon"],
		Design:       values["--design"],
		Evaluation:   values["--evaluation"],
		ResearchType: values["--research-type"],
	})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "protocol_compile_failed", err.Error())
	}
	if args[0] == "plan-sources" {
		return writeSourcePlan(protocol.CompileSourcePlan(plan), stdout, opts)
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"plan": plan})
	}
	fmt.Fprintf(stdout, "Framework: %s\n", plan.Framework)
	fmt.Fprintf(stdout, "Question: %s\n", plan.Question)
	fmt.Fprintln(stdout, "\nSource queries:")
	for _, query := range plan.SortedSourceQueries() {
		fmt.Fprintf(stdout, "- %s: %s\n", query.Source, query.Query)
	}
	fmt.Fprintln(stdout, "\nInclusion criteria:")
	for _, criterion := range plan.InclusionCriteria {
		fmt.Fprintf(stdout, "- %s\n", criterion)
	}
	fmt.Fprintln(stdout, "\nExclusion criteria:")
	for _, criterion := range plan.ExclusionCriteria {
		fmt.Fprintf(stdout, "- %s\n", criterion)
	}
	fmt.Fprintf(stdout, "\nExtraction schema: %s\n", plan.ExtractionSchema.Name)
	for _, field := range plan.ExtractionSchema.Fields {
		required := "optional"
		if field.Required {
			required = "required"
		}
		fmt.Fprintf(stdout, "- %s (%s, %s)\n", field.Name, field.Type, required)
	}
	fmt.Fprintln(stdout, "\nReviewer prompts:")
	for _, prompt := range plan.ReviewerPrompts {
		fmt.Fprintf(stdout, "- %s [%s]\n", prompt.Text, prompt.ProvenanceTag)
	}
	fmt.Fprintln(stdout, "\nReviewer approval required: true")
	fmt.Fprintln(stdout, "Auto-accepted claims: false")
	return 0
}

func executeSuggestExpansions(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	values, err := parseKeyValueFlags(args, map[string]bool{"--question": true, "--source-text": true, "--source-ref": true, "--paper-id": true})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	link := protocol.SourceTextLink{ID: values["--source-ref"], PassageRef: values["--source-ref"], PaperID: values["--paper-id"], Text: values["--source-text"]}
	if link.ID == "" {
		link.ID = "source-text-1"
	}
	suggestions, err := protocol.DraftQueryExpansionSuggestions(protocol.QueryExpansionInput{Question: values["--question"], SourceTexts: []protocol.SourceTextLink{link}})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "query_expansion_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"suggestions": suggestions})
	}
	fmt.Fprintln(stdout, "Query expansion suggestions:")
	for _, suggestion := range suggestions {
		fmt.Fprintf(stdout, "- %s: %s\n", suggestion.Assistant, suggestion.SuggestedTerm)
		fmt.Fprintf(stdout, "  Source text links: %d\n", len(suggestion.SourceTextLinks))
		fmt.Fprintln(stdout, "  Reviewer approval required before source plan changes: true")
	}
	return 0
}

func executeLiveSmokeSnapshot(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	values, err := parseKeyValueFlags(args, map[string]bool{"--output": true, "--connector": true, "--status": true, "--message": true, "--fields": true})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	output := strings.TrimSpace(values["--output"])
	if output == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "--output is required")
	}
	registry := protocol.DefaultConnectorCapabilityRegistry()
	now := time.Now().UTC()
	snapshot := protocol.NewLiveSmokeSnapshot(registry, now)
	if connector := strings.TrimSpace(values["--connector"]); connector != "" {
		capability, ok := registry.ByID(connector)
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", fmt.Sprintf("unknown connector %q", connector))
		}
		status := strings.TrimSpace(values["--status"])
		if status == "" {
			status = protocol.LiveSmokeSkipped
		}
		snapshot.UpsertResult(protocol.ConnectorLiveSmokeResult{ConnectorID: connector, Label: capability.Label, Status: status, CheckedAt: now, Message: values["--message"], ObservedFields: splitCSV(values["--fields"])})
	}
	if err := protocol.SaveLiveSmokeSnapshot(output, snapshot); err != nil {
		return writeError(stdout, stderr, opts, 1, "live_smoke_snapshot_failed", err.Error())
	}
	alerts := protocol.ConnectorLiveSmokeAlerts(registry, snapshot, now)
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"snapshot": snapshot, "alerts": alerts})
	}
	fmt.Fprintf(stdout, "Wrote live-smoke snapshot: %s\n", output)
	fmt.Fprintf(stdout, "Connector alerts: %d\n", len(alerts))
	return 0
}

func writeCapabilities(registry protocol.ConnectorCapabilityRegistry, stdout io.Writer, opts globalOptions) int {
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"registry": registry})
	}
	fmt.Fprintln(stdout, "Connector capabilities:")
	for _, connector := range registry.Connectors {
		fmt.Fprintf(stdout, "- %s (%s)\n", connector.Label, connector.Kind)
		fmt.Fprintf(stdout, "  Entities: %s\n", strings.Join(connector.SupportedEntities, ", "))
		fmt.Fprintf(stdout, "  Rate limits: %s\n", connector.RateLimitPolicy)
		fmt.Fprintf(stdout, "  Auth: %s\n", connector.AuthNeeds)
		fmt.Fprintf(stdout, "  Live smoke: %s\n", connector.LiveSmokeStatus)
		fmt.Fprintf(stdout, "  License/shareability: %s\n", connector.LicenseShareabilityPolicy)
		fmt.Fprintf(stdout, "  Cacheability: %s\n", connector.Cacheability)
		fmt.Fprintf(stdout, "  Provenance: %s\n", strings.Join(connector.ProvenanceFields, ", "))
	}
	return 0
}

func writeSourcePlan(plan protocol.SourcePlan, stdout io.Writer, opts globalOptions) int {
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"sourcePlan": plan})
	}
	fmt.Fprintf(stdout, "Question: %s\n", plan.Question)
	fmt.Fprintln(stdout, "\nSource plan preview:")
	for _, source := range plan.Sources {
		fmt.Fprintf(stdout, "- %s (%s)\n", source.Label, source.SourceKind)
		if source.Query != "" {
			fmt.Fprintf(stdout, "  Query: %s\n", source.Query)
		}
		fmt.Fprintf(stdout, "  Dry run: %s\n", source.DryRunEstimate)
		fmt.Fprintf(stdout, "  Rate limits: %s\n", source.RateLimitPolicy)
		fmt.Fprintf(stdout, "  Auth: %s\n", source.AuthRequirement)
		fmt.Fprintf(stdout, "  Live smoke: %s\n", source.LiveSmokeStatus)
		fmt.Fprintf(stdout, "  License/shareability: %s\n", source.LicenseShareabilityPolicy)
		fmt.Fprintf(stdout, "  Cacheability: %s\n", source.Cacheability)
		fmt.Fprintf(stdout, "  Provenance: %s\n", strings.Join(source.ProvenanceFields, ", "))
		fmt.Fprintf(stdout, "  Privacy: %s\n", source.PrivacyWarning)
		fmt.Fprintf(stdout, "  CLI: %s\n", source.CLICommand)
	}
	fmt.Fprintln(stdout, "\nReviewer approval required before network calls, imports, downloads, or package inclusion.")
	return 0
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseKeyValueFlags(args []string, allowed map[string]bool) (map[string]string, error) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") || !allowed[arg] {
			return nil, fmt.Errorf("unknown flag %q", arg)
		}
		if i+1 >= len(args) {
			return nil, fmt.Errorf("missing value for %s", arg)
		}
		values[arg] = args[i+1]
		i++
	}
	return values, nil
}

func parseProtocolCompileFlags(args []string) (map[string]string, error) {
	allowed := map[string]bool{
		"--type": true, "--question": true, "--population": true, "--intervention": true,
		"--comparator": true, "--outcome": true, "--exposure": true, "--sample": true,
		"--phenomenon": true, "--design": true, "--evaluation": true, "--research-type": true,
	}
	values, err := parseKeyValueFlags(args, allowed)
	if err != nil {
		return nil, fmt.Errorf("%s", strings.NewReplacer("unknown flag", "unknown protocol compile flag").Replace(err.Error()))
	}
	if values["--type"] == "" {
		values["--type"] = "freeform"
	}
	return values, nil
}
