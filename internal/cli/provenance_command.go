package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func executeProvenance(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge provenance <note|validate>")
	}
	switch args[0] {
	case "note":
		return executeProvenanceNote(args[1:], stdout, stderr, opts)
	case "validate":
		return executeProvenanceValidate(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_provenance_subcommand", fmt.Sprintf("unknown provenance subcommand %q", args[0]))
	}
}

func executeProvenanceNote(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for provenance note")
	}
	message, actor, ok := parseProvenanceNote(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge provenance note --message <text> [--actor <name>]")
	}
	if err := provenance.Note(opts.Project, message, actor); err != nil {
		return writeError(stdout, stderr, opts, 1, "provenance_note_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"action":  "provenance.researcher.note",
			"message": message,
			"actor":   actor,
		})
	}
	fmt.Fprintln(stdout, "note recorded")
	return 0
}

func parseProvenanceNote(args []string) (message, actor string, ok bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--message", "--actor":
			if i+1 >= len(args) {
				return "", "", false
			}
			if args[i] == "--message" {
				message = args[i+1]
			} else {
				actor = args[i+1]
			}
			i++
		default:
			return "", "", false
		}
	}
	return message, actor, message != ""
}

var validProvenanceDepths = map[string]bool{"quick": true, "standard": true, "comprehensive": true}

// executeProvenanceValidate checks a search-output provenance.json against the
// versioned schema: required fields, enum depth, string-only errors, and a
// present rforge_version. It is the enforcement lever for the provenance
// schema that the research-forge skill template makes advisory.
func executeProvenanceValidate(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge provenance validate <provenance.json>")
	}
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "provenance_validate_read_failed", err.Error())
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return writeError(stdout, stderr, opts, 1, "provenance_validate_parse_failed", fmt.Sprintf("invalid JSON: %v", err))
	}
	if problems := validateProvenanceRecord(record); len(problems) != 0 {
		return writeError(stdout, stderr, opts, 1, "provenance_validation_failed", strings.Join(problems, "; "))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"path": path, "valid": true})
	}
	fmt.Fprintf(stdout, "provenance valid: %s\n", path)
	return 0
}

func validateProvenanceRecord(record map[string]any) []string {
	var problems []string
	for _, field := range []string{"schema_version", "question", "depth", "sources", "timestamp", "rforge_version", "outputs", "errors"} {
		if _, ok := record[field]; !ok {
			problems = append(problems, fmt.Sprintf("missing required field %q", field))
		}
	}
	if version, ok := record["schema_version"].(string); !ok || version != "1" {
		problems = append(problems, `schema_version must be "1"`)
	}
	if question, ok := record["question"].(string); !ok || strings.TrimSpace(question) == "" {
		problems = append(problems, "question must be a non-empty string")
	}
	if depth, ok := record["depth"].(string); !ok || !validProvenanceDepths[strings.TrimSpace(depth)] {
		problems = append(problems, "depth must be one of quick, standard, comprehensive")
	}
	if timestamp, ok := record["timestamp"].(string); !ok || strings.TrimSpace(timestamp) == "" {
		problems = append(problems, "timestamp must be a non-empty string")
	}
	version, ok := record["rforge_version"].(map[string]any)
	if !ok {
		problems = append(problems, "rforge_version must be an object copied from `rforge --json version` data")
	} else {
		for _, field := range []string{"version", "commit", "date"} {
			value, ok := version[field].(string)
			if !ok || strings.TrimSpace(value) == "" {
				problems = append(problems, fmt.Sprintf("rforge_version.%s must be a non-empty string", field))
			}
		}
	}
	for _, field := range []string{"sources", "outputs", "errors"} {
		values, ok := record[field].([]any)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s must be an array of strings", field))
			continue
		}
		for i, value := range values {
			if _, ok := value.(string); !ok {
				problems = append(problems, fmt.Sprintf("%s[%d] must be a string, got %T", field, i, value))
			}
		}
	}
	if _, exists := record["queries"]; exists {
		values, ok := record["queries"].([]any)
		if !ok {
			problems = append(problems, "queries must be an array of strings")
		} else {
			for i, value := range values {
				if _, ok := value.(string); !ok {
					problems = append(problems, fmt.Sprintf("queries[%d] must be a string, got %T", i, value))
				}
			}
		}
	}
	return problems
}
