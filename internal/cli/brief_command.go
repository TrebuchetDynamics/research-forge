package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type briefPaper struct {
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Venue       string   `json:"venue"`
	Abstract    string   `json:"abstract"`
	OpenAccess  bool     `json:"openAccess"`
	Source      string   `json:"source"`
	URLs        []string `json:"urls"`
	Identifiers struct {
		DOI     string `json:"DOI"`
		ArXivID string `json:"ArXivID"`
	} `json:"identifiers"`
}

type briefManifest struct {
	Queries   []string `json:"queries"`
	Sources   []string `json:"sources"`
	Results   int      `json:"results"`
	Deduped   int      `json:"deduped"`
	Failures  int      `json:"failures"`
	CreatedAt string   `json:"createdAt"`
}

// executeBrief produces a compact research brief from a search directory.
func executeBrief(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge brief <search-dir> [--out <brief.md>]")
	}
	searchDir := args[0]
	outFile := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--out" && i+1 < len(args) {
			outFile = args[i+1]
			i++
		}
	}

	// Read manifest
	var manifest briefManifest
	manifestData, err := os.ReadFile(filepath.Join(searchDir, "manifest.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "brief_manifest_read_failed", err.Error())
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return writeError(stdout, stderr, opts, 1, "brief_manifest_parse_failed", err.Error())
	}

	// Read deduped results (JSONL)
	var papers []briefPaper
	dedupedData, err := os.ReadFile(filepath.Join(searchDir, "results-deduped.jsonl"))
	if err == nil {
		for _, line := range strings.Split(string(dedupedData), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var p briefPaper
			if err := json.Unmarshal([]byte(line), &p); err == nil && p.Title != "" {
				papers = append(papers, p)
			}
		}
	}

	topic := ""
	if len(manifest.Queries) > 0 {
		topic = manifest.Queries[0]
	}

	// Build the brief
	var sb strings.Builder
	if topic != "" {
		fmt.Fprintf(&sb, "# Research Brief: %s\n\n", topic)
	} else {
		sb.WriteString("# Research Brief\n\n")
	}

	sb.WriteString("## Summary\n\n")
	fmt.Fprintf(&sb, "- **Queries:** %d\n", len(manifest.Queries))
	fmt.Fprintf(&sb, "- **Sources:** %s\n", strings.Join(manifest.Sources, ", "))
	fmt.Fprintf(&sb, "- **Total records:** %d\n", manifest.Results)
	fmt.Fprintf(&sb, "- **Deduplicated:** %d\n", manifest.Deduped)
	fmt.Fprintf(&sb, "- **Failures:** %d\n", manifest.Failures)
	if manifest.CreatedAt != "" {
		fmt.Fprintf(&sb, "- **Searched:** %s\n", manifest.CreatedAt[:10])
	}
	sb.WriteString("\n")

	if len(manifest.Queries) > 1 {
		sb.WriteString("## Queries\n\n")
		for _, q := range manifest.Queries {
			fmt.Fprintf(&sb, "- `%s`\n", q)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Top Papers\n\n")
	sb.WriteString("| # | Title | Year | Venue | OA |\n")
	sb.WriteString("|---|-------|------|-------|----|\n")
	if len(papers) == 0 {
		sb.WriteString("| | _No deduplicated records found_ | | | |\n")
	} else {
		sort.Slice(papers, func(i, j int) bool {
			if papers[i].Year != papers[j].Year {
				return papers[i].Year > papers[j].Year
			}
			return papers[i].Title < papers[j].Title
		})
		limit := 25
		if len(papers) < limit {
			limit = len(papers)
		}
		for i, p := range papers[:limit] {
			oa := ""
			if p.OpenAccess {
				oa = "🔓"
			}
			title := p.Title
			if p.Identifiers.DOI != "" {
				title = fmt.Sprintf("[%s](https://doi.org/%s)", title, p.Identifiers.DOI)
			} else if p.Identifiers.ArXivID != "" {
				title = fmt.Sprintf("[%s](https://arxiv.org/abs/%s)", title, p.Identifiers.ArXivID)
			}
			yearStr := ""
			if p.Year > 0 {
				yearStr = fmt.Sprintf("%d", p.Year)
			}
			fmt.Fprintf(&sb, "| %d | %s | %s | %s | %s |\n", i+1, title, yearStr, p.Venue, oa)
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## Gaps & Limitations\n\n")
	sb.WriteString("- **Source coverage:** ")
	if len(manifest.Sources) > 0 {
		sb.WriteString(strings.Join(manifest.Sources, ", "))
	} else {
		sb.WriteString("none listed")
	}
	sb.WriteString("\n")
	if surplus := manifest.Results - manifest.Deduped; surplus > 0 {
		fmt.Fprintf(&sb, "- **Deduplication removed** %d duplicate records\n", surplus)
	}
	if manifest.Failures > 0 {
		fmt.Fprintf(&sb, "- **%d query/source pair(s) failed** — retry with `rforge search resume --failures %s/failures.jsonl --out .`\n", manifest.Failures, searchDir)
	}
	sb.WriteString("- Title/abstract screening has not been applied\n")
	sb.WriteString("- Full-text review and human inclusion/exclusion decisions are required before drawing conclusions\n\n")

	sb.WriteString("## Next Steps\n\n```sh\n")
	fmt.Fprintf(&sb, "# Screen records before PDF acquisition\nrforge screen queue --dir %s --out %s/queue.csv\n\n", searchDir, searchDir)
	fmt.Fprintf(&sb, "# Fetch open-access PDFs for included papers (run AFTER screening)\nrforge oa fetch --dir %s\n\n", searchDir)
	fmt.Fprintf(&sb, "# Generate numbered bibliography\nrforge citations build --research-dir %s\n```\n", filepath.Dir(searchDir))

	brief := sb.String()

	if outFile != "" {
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "brief_write_failed", err.Error())
		}
		if err := os.WriteFile(outFile, []byte(brief), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "brief_write_failed", err.Error())
		}
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"topic":        topic,
			"queries":      manifest.Queries,
			"sources":      manifest.Sources,
			"totalRecords": manifest.Results,
			"deduped":      manifest.Deduped,
			"failures":     manifest.Failures,
			"briefSize":    len(brief),
			"out":          outFile,
		})
	}
	fmt.Fprintf(stdout, "%s", brief)
	if outFile != "" {
		fmt.Fprintf(stdout, "\nwrote brief to %s\n", outFile)
	}
	return 0
}
