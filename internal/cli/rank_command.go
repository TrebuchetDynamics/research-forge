package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

func executeRank(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge rank <topic> [--sources openalex,crossref,semantic-scholar] [--limit N] [--out <file>] [--graph <graph.json>]")
	}
	request := research.RankOptions{Topic: args[0], Limit: 20, Project: opts.Project}
	sourceList := "openalex,crossref"
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil && v > 0 {
					request.Limit = v
				}
				i++
			}
		case "--out":
			if i+1 < len(args) {
				request.Out = args[i+1]
				i++
			}
		case "--sources":
			if i+1 < len(args) {
				sourceList = args[i+1]
				i++
			}
		case "--graph":
			if i+1 < len(args) {
				request.GraphPath = args[i+1]
				i++
			}
		}
	}
	request.Sources = strings.Split(sourceList, ",")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := researchService().Rank(ctx, request)
	if err != nil {
		return writeResearchError(stdout, stderr, opts, err)
	}
	if opts.JSON {
		return writeJSON(stdout, 0, result)
	}
	title := fmt.Sprintf("PaperRank: %s", result.Topic)
	fmt.Fprintf(stdout, "%s\n%s\n\n", title, strings.Repeat("=", len(title)))
	for _, p := range result.Rankings {
		symbol := ""
		if p.OpenAccess {
			symbol = " 🔓"
		}
		fmt.Fprintf(stdout, "%3d. [%.2f] %s%s\n", p.Rank, p.RawScore, p.Title, symbol)
		if p.DOI != "" {
			fmt.Fprintf(stdout, "       DOI: %s\n", p.DOI)
		}
		if p.ArXivID != "" {
			fmt.Fprintf(stdout, "       arXiv: %s\n", p.ArXivID)
		}
		if p.Year > 0 {
			fmt.Fprintf(stdout, "       Year: %d", p.Year)
			if p.Venue != "" {
				fmt.Fprintf(stdout, " — %s", p.Venue)
			}
			fmt.Fprintln(stdout)
		}
	}
	fmt.Fprintf(stdout, "\nrankings saved to %s\n", result.Out)
	return 0
}
