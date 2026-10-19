package cli

import (
	"bytes"
	"io"
)

// executeWorkflowCommand records successful workflow runs, but not help, in the
// project's command ledger. Hold success output until the event is durable so a
// failed append cannot emit both a success and an error JSON response.
func executeWorkflowCommand(command string, args []string, stdout, stderr io.Writer, opts globalOptions) int {
	run := map[string]func([]string, io.Writer, io.Writer, globalOptions) int{
		"paper":     executePaper,
		"rank":      executeRank,
		"brief":     executeBrief,
		"compare":   executeCompare,
		"lit":       executeLit,
		"note":      executeNote,
		"history":   executeHistory,
		"replicate": executeReplicate,
		"draft":     executeDraft,
	}[command]
	if opts.Project == "" || len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		return run(args, stdout, stderr, opts)
	}
	var output bytes.Buffer
	code := run(args, &output, stderr, opts)
	if code == 0 {
		name := command
		if command == "history" || command == "note" {
			name += " " + args[0]
		}
		if err := recordCLICommand(opts.Project, name, opts, map[string]any{"exitCode": code}); err != nil {
			return writeError(stdout, stderr, opts, 1, "provenance_append_failed", err.Error())
		}
	}
	_, _ = io.Copy(stdout, &output)
	return code
}
