package cli

import (
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// executeBrief parses and displays a compact research brief.
func executeBrief(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge brief <search-dir> [--out <brief.md>]")
	}
	request := research.BriefOptions{SearchDir: args[0]}
	for i := 1; i < len(args); i++ {
		if args[i] == "--out" && i+1 < len(args) {
			request.Out = args[i+1]
			i++
		}
	}
	result, err := researchService().Brief(request)
	if err != nil {
		return writeResearchError(stdout, stderr, opts, err)
	}
	if opts.JSON {
		return writeJSON(stdout, 0, result)
	}
	fmt.Fprint(stdout, result.Markdown)
	if request.Out != "" {
		fmt.Fprintf(stdout, "\nwrote brief to %s\n", request.Out)
	}
	return 0
}
