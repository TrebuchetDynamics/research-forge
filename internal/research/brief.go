package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Paper is a search-result record used by briefs and comparisons.
type Paper struct {
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

type searchManifest struct {
	Queries   []string `json:"queries"`
	Sources   []string `json:"sources"`
	Results   int      `json:"results"`
	Deduped   int      `json:"deduped"`
	Failures  int      `json:"failures"`
	CreatedAt string   `json:"createdAt"`
}

type BriefOptions struct{ SearchDir, Out string }
type BriefResult struct {
	Topic        string   `json:"topic"`
	Queries      []string `json:"queries"`
	Sources      []string `json:"sources"`
	TotalRecords int      `json:"totalRecords"`
	Deduped      int      `json:"deduped"`
	Failures     int      `json:"failures"`
	BriefSize    int      `json:"briefSize"`
	Out          string   `json:"out"`
	Papers       []Paper  `json:"-"`
	Markdown     string   `json:"-"`
}

// Brief builds and optionally persists a compact summary of one search session.
func (s Service) Brief(opts BriefOptions) (BriefResult, error) {
	searchDir, outFile := opts.SearchDir, opts.Out
	// Read manifest
	var manifest searchManifest
	manifestData, err := os.ReadFile(filepath.Join(searchDir, "manifest.json"))
	if err != nil {
		return BriefResult{}, failure("brief_manifest_read_failed", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return BriefResult{}, failure("brief_manifest_parse_failed", err)
	}

	// Read deduped results (JSONL)
	resultsPath := filepath.Join(searchDir, "results-deduped.jsonl")
	dedupedData, err := os.ReadFile(resultsPath)
	if err != nil {
		return BriefResult{}, failure("brief_results_read_failed", err)
	}
	papers, err := decodeReportResults(dedupedData, resultsPath)
	if err != nil {
		return BriefResult{}, failure("brief_results_parse_failed", err)
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
		date := manifest.CreatedAt
		if len(date) > 10 {
			date = date[:10]
		}
		fmt.Fprintf(&sb, "- **Searched:** %s\n", date)
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
			return BriefResult{}, failure("brief_write_failed", err)
		}
		if err := os.WriteFile(outFile, []byte(brief), 0o644); err != nil {
			return BriefResult{}, failure("brief_write_failed", err)
		}
	}

	return BriefResult{
		Topic: topic, Queries: manifest.Queries, Sources: manifest.Sources,
		TotalRecords: manifest.Results, Deduped: manifest.Deduped, Failures: manifest.Failures,
		BriefSize: len(brief), Out: outFile, Papers: papers, Markdown: brief,
	}, nil
}
