package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/protocol"
)

func executeProtocol(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "compile" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge protocol compile --question <text> [--type pico|peco|spider|freeform] [framework flags]")
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

func parseProtocolCompileFlags(args []string) (map[string]string, error) {
	allowed := map[string]bool{
		"--type": true, "--question": true, "--population": true, "--intervention": true,
		"--comparator": true, "--outcome": true, "--exposure": true, "--sample": true,
		"--phenomenon": true, "--design": true, "--evaluation": true, "--research-type": true,
	}
	values := map[string]string{"--type": "freeform"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") || !allowed[arg] {
			return nil, fmt.Errorf("unknown protocol compile flag %q", arg)
		}
		if i+1 >= len(args) {
			return nil, fmt.Errorf("missing value for %s", arg)
		}
		values[arg] = args[i+1]
		i++
	}
	return values, nil
}
