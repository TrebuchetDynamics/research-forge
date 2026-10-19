package cli

import (
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// executeLit parses and displays a persisted literature synthesis.
func executeLit(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	request := research.LitOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--research-dir":
			if i+1 < len(args) {
				request.ResearchDir = args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(args) {
				request.Out = args[i+1]
				i++
			}
		}
	}
	if request.ResearchDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge lit --research-dir <dir> [--out <dir>]")
	}
	if request.Out == "" {
		request.Out = request.ResearchDir
	}
	result, err := researchService().Lit(request)
	if err != nil {
		return writeResearchError(stdout, stderr, opts, err)
	}
	if opts.JSON {
		return writeJSON(stdout, 0, result)
	}
	fmt.Fprintf(stdout, "wrote LIT.md and LIT.provenance.json to %s\n", request.Out)
	return 0
}
