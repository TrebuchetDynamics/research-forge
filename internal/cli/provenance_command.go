package cli

import (
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func executeProvenance(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge provenance note --message <text> [--actor <name>]")
	}
	switch args[0] {
	case "note":
		return executeProvenanceNote(args[1:], stdout, stderr, opts)
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
