package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// executeCompare parses and displays a comparison of search directories.
func executeCompare(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) < 2 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge compare <search-dir-1> <search-dir-2> [<search-dir-3>...] [--out <diff.md>] [--labels l1,l2,...]")
	}
	request := research.CompareOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 < len(args) {
				request.Out = args[i+1]
				i++
			}
		case "--labels":
			if i+1 < len(args) {
				request.Labels = strings.Split(args[i+1], ",")
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				request.Dirs = append(request.Dirs, args[i])
			}
		}
	}
	if len(request.Dirs) < 2 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge compare <dir1> <dir2> [<dir3>...] [--out <diff.md>] [--labels l1,l2,...]")
	}
	result, err := researchService().Compare(request)
	if err != nil {
		return writeResearchError(stdout, stderr, opts, err)
	}
	if opts.JSON {
		return writeJSON(stdout, 0, result)
	}
	fmt.Fprint(stdout, result.Markdown)
	if request.Out != "" {
		fmt.Fprintf(stdout, "wrote comparison to %s\n", request.Out)
	}
	return 0
}

// The draft adapter also consumes the shared search-result reader.
func readBriefPaperResults(searchDir string) ([]research.Paper, error) {
	return research.ReadPaperResults(searchDir)
}
