package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func executePaper(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	// Parse flags: --file <ids.txt> and --out <file>
	inputs := []string{}
	filePath := ""
	outPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file", "batch":
			if args[i] == "--file" && i+1 < len(args) {
				filePath = args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(args) {
				outPath = args[i+1]
				i++
			}
		case "--help", "-h":
			printPaperHelp(stdout, opts)
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				return writeError(stdout, stderr, opts, 2, "invalid_flag", "unknown flag "+args[i])
			}
			inputs = append(inputs, args[i])
		}
	}

	// Load IDs from file if --file given
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "paper_file_read_failed", err.Error())
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				inputs = append(inputs, line)
			}
		}
	}

	if len(inputs) == 0 {
		printPaperHelp(stdout, opts)
		return 0
	}

	httpClient := sources.NewHTTPClient(sources.HTTPClientOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	results := make([]sources.PaperResolution, 0, len(inputs))
	for _, arg := range inputs {
		result, err := sources.ResolvePaper(ctx, httpClient, arg)
		if err == nil {
			results = append(results, result)
		}
	}

	if outPath != "" {
		if err := writeJSONFile(outPath, map[string]any{"resolutions": results, "count": len(results), "requested": len(inputs)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "paper_write_failed", err.Error())
		}
	}

	if opts.JSON {
		if len(inputs) == 1 && len(results) == 1 {
			return writeJSON(stdout, 0, map[string]any{"resolution": results[0]})
		}
		return writeJSON(stdout, 0, map[string]any{"resolutions": results, "count": len(results), "requested": len(inputs)})
	}
	for _, result := range results {
		printPaperResult(stdout, result)
		fmt.Fprintln(stdout)
	}
	if outPath != "" {
		fmt.Fprintf(stdout, "wrote %d resolutions to %s\n", len(results), outPath)
	}
	return 0
}

func printPaperHelp(w io.Writer, opts globalOptions) {
	help := `Resolve one or more papers by identifier (DOI, arXiv ID, OpenAlex ID, URL, or title).

Usage:
  rforge paper <id> [<id>...]
  rforge paper --file <ids.txt>

Flags:
  --file <ids.txt>  Resolve many IDs from a file (one per line, # comments)
  --json            Output as JSON
  --out <file>      Save resolution to file

Examples:
  rforge paper 10.7717/peerj.4375
  rforge paper 2301.99999
  rforge paper W123456789
  rforge paper "transformer scaling laws"
  rforge paper https://arxiv.org/abs/2301.99999
  rforge paper --file papers.txt`
	if opts.JSON {
		writeJSON(w, 0, map[string]any{"help": help})
		return
	}
	fmt.Fprintln(w, help)
}

func printPaperResult(w io.Writer, result sources.PaperResolution) {
	if !result.Resolved {
		fmt.Fprintf(w, "✗ %s → not found\n", result.Input)
		return
	}
	p := result.Paper
	fmt.Fprintf(w, "✓ %s\n", p.Title)
	fmt.Fprintf(w, "  ID:    %s\n", result.Input)
	if id := p.Identifiers.DOI; id != "" {
		fmt.Fprintf(w, "  DOI:   %s\n", id)
	}
	if id := p.Identifiers.ArXivID; id != "" {
		fmt.Fprintf(w, "  arXiv: %s\n", id)
	}
	if id := p.Identifiers.OpenAlexID; id != "" {
		fmt.Fprintf(w, "  OA:    %s\n", id)
	}
	if p.Year > 0 {
		fmt.Fprintf(w, "  Year:  %d\n", p.Year)
	}
	if p.Venue != "" {
		fmt.Fprintf(w, "  Venue: %s\n", p.Venue)
	}
	if len(p.URLs) > 0 {
		fmt.Fprintf(w, "  URLs:  %s\n", p.URLs[0])
	}
	if result.OA != nil && result.OA.IsOA {
		fmt.Fprintf(w, "  OA:    yes")
		if result.OA.PDFURL != "" {
			fmt.Fprintf(w, " (%s)", result.OA.PDFURL)
		}
		fmt.Fprintln(w)
	}
	if len(result.Sources) > 0 {
		srcs := make([]string, len(result.Sources))
		for i, s := range result.Sources {
			srcs[i] = s.Source
		}
		fmt.Fprintf(w, "  From:  %s\n", formatSources(srcs))
	}
	// Show a preview of the abstract
	if p.Abstract != "" {
		preview := p.Abstract
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Fprintf(w, "  Abs:   %s\n", preview)
	}
}

func formatSources(sources []string) string {
	switch len(sources) {
	case 0:
		return ""
	case 1:
		return sources[0]
	default:
		out := ""
		for i, s := range sources {
			if i > 0 {
				out += ", "
			}
			out += s
		}
		return out
	}
}

// init is not needed — executePaper is dispatched from the switch in cli.go
