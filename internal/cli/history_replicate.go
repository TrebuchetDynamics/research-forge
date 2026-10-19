package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// sessionRecord is one hit in a history search against a research dir.
type sessionRecord struct {
	Topic   string `json:"topic"`
	Title   string `json:"title"`
	DOI     string `json:"doi,omitempty"`
	ArXivID string `json:"arxivId,omitempty"`
	Year    int    `json:"year"`
	Matches int    `json:"matches"`
}

// executeHistory dispatches subcommands of the history top-level command.
func executeHistory(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "search" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge history search <term> --research-dir <dir> [--limit N] [--out <file>]")
	}
	return executeSessionSearch(args[1:], stdout, stderr, opts)
}

// executeSessionSearch searches across all research directories for matching papers.
func executeSessionSearch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	researchDir := ""
	term := ""
	outFile := ""
	limit := 20

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--research-dir":
			if i+1 < len(args) {
				researchDir = args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(args) {
				outFile = args[i+1]
				i++
			}
		case "--limit":
			if i+1 < len(args) {
				if v, err := fmt.Sscanf(args[i+1], "%d", &limit); err == nil {
					_ = v
				}
				i++
			}
		case "--term":
			if i+1 < len(args) {
				term = args[i+1]
				i++
			}
		default:
			if term == "" && !strings.HasPrefix(args[i], "-") {
				term = args[i]
			}
		}
	}
	if researchDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge history search <term> --research-dir <dir> [--limit N] [--out <file>]")
	}
	if term == "" {
		return writeError(stdout, stderr, opts, 2, "missing_term", "search term is required")
	}
	termLower := strings.ToLower(term)
	termWords := strings.Fields(termLower)

	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "history_read_failed", err.Error())
	}

	var results []sessionRecord
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		topic := e.Name()

		// Search deduped results
		dd, err := os.ReadFile(filepath.Join(researchDir, topic, "results-deduped.jsonl"))
		if err != nil {
			// try results.jsonl
			dd, err = os.ReadFile(filepath.Join(researchDir, topic, "results.jsonl"))
			if err != nil {
				continue
			}
		}
		for _, line := range strings.Split(string(dd), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var doc map[string]any
			if err := json.Unmarshal([]byte(line), &doc); err != nil {
				continue
			}
			title, _ := doc["title"].(string)
			if title == "" {
				// try uppercase key (library.PaperRecord serialization)
				if v, ok := doc["Title"].(string); ok {
					title = v
				}
			}
			abstract, _ := doc["abstract"].(string)
			if abstract == "" {
				if v, ok := doc["Abstract"].(string); ok {
					abstract = v
				}
			}
			identifiers, _ := doc["identifiers"].(map[string]any)
			if identifiers == nil {
				if v, ok := doc["Identifiers"].(map[string]any); ok {
					identifiers = v
				}
			}
			doi, _ := identifiers["DOI"].(string)
			if doi == "" {
				if v, ok := doc["doi"].(string); ok {
					doi = v
				}
			}
			arx, _ := identifiers["ArXivID"].(string)
			if arx == "" {
				if v, ok := doc["arxivId"].(string); ok {
					arx = v
				}
			}
			year, _ := doc["year"].(float64)

			// Score matches: 2 per word in title, 1 per word in abstract
			haystack := strings.ToLower(title + " " + abstract + " " + doi + " " + arx)
			matches := 0
			for _, w := range termWords {
				if strings.Contains(haystack, w) {
					matches++
				}
			}
			// Bonus for exact title match
			if strings.Contains(strings.ToLower(title), termLower) {
				matches += 3
			}
			if matches == 0 {
				continue
			}
			results = append(results, sessionRecord{
				Topic:   topic,
				Title:   title,
				DOI:     doi,
				ArXivID: arx,
				Year:    int(year),
				Matches: matches,
			})
		}
	}

	// Sort: most matches first, then year desc, then title
	sort.Slice(results, func(i, j int) bool {
		if results[i].Matches != results[j].Matches {
			return results[i].Matches > results[j].Matches
		}
		if results[i].Year != results[j].Year {
			return results[i].Year > results[j].Year
		}
		return results[i].Title < results[j].Title
	})
	if len(results) > limit {
		results = results[:limit]
	}

	if outFile != "" {
		if err := writeJSONFile(outFile, map[string]any{"term": term, "count": len(results), "results": results}); err != nil {
			return writeError(stdout, stderr, opts, 1, "history_write_failed", err.Error())
		}
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"term": term, "count": len(results), "results": results})
	}

	if len(results) == 0 {
		fmt.Fprintf(stdout, "no results for %q in %s\n", term, researchDir)
		return 0
	}
	fmt.Fprintf(stdout, "Found %d result(s) for %q\n\n", len(results), term)
	fmt.Fprintf(stdout, "%-3s %-5s %-20s %s\n", "#", "Year", "Topic", "Title")
	fmt.Fprintln(stdout, strings.Repeat("-", 80))
	for i, r := range results {
		yr := ""
		if r.Year > 0 {
			yr = fmt.Sprintf("%d", r.Year)
		}
		fmt.Fprintf(stdout, "%-3d %-5s %-20s %s\n", i+1, yr, truncateString(r.Topic, 18), r.Title)
		if r.DOI != "" || r.ArXivID != "" {
			id := r.DOI
			if id == "" {
				id = "arXiv:" + r.ArXivID
			}
			fmt.Fprintf(stdout, "      %-5s %-20s %s\n", "", "", id)
		}
	}
	return 0
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// executeReplicate generates a deterministic replication plan from a resolved paper ID.
func executeReplicate(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(stdout, `Generate a deterministic replication plan from a resolved paper.

Usage:
  rforge replicate <doi|arxiv-id> [--out <plan.md>] [--depth quick|standard|comprehensive]

Examples:
  rforge replicate 10.1136/bmj.n160 --out replan.md
  rforge replicate 2301.12345`)
		return 0
	}

	paperInput := args[0]
	outFile := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 < len(args) {
				outFile = args[i+1]
				i++
			}
		case "--depth":
			// recognized but unused for now
			i++
		}
	}

	// Resolve the paper
	httpClient := sources.NewHTTPClient(sources.HTTPClientOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resolution, err := sources.ResolvePaper(ctx, httpClient, paperInput)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "replicate_resolve_failed", fmt.Sprintf("resolve paper: %v", err))
	}
	if !resolution.Resolved {
		return writeError(stdout, stderr, opts, 1, "replicate_unresolved", fmt.Sprintf("could not resolve %q", paperInput))
	}

	p := resolution.Paper

	// Find code repos in URLs and metadata
	var codeRepos []string
	githubRE := regexp.MustCompile(`github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+`)
	for _, u := range p.URLs {
		if m := githubRE.FindString(u); m != "" {
			codeRepos = append(codeRepos, m)
		}
	}
	for _, s := range resolution.Sources {
		if m := githubRE.FindString(s.RawRef); m != "" {
			codeRepos = append(codeRepos, m)
		}
	}
	// Deduplicate
	seen := map[string]bool{}
	var uniqueRepos []string
	for _, r := range codeRepos {
		if !seen[r] {
			seen[r] = true
			uniqueRepos = append(uniqueRepos, r)
		}
	}
	codeRepos = uniqueRepos

	// Build the plan
	var sb strings.Builder
	sb.WriteString("# Replication Plan\n\n")
	sb.WriteString(fmt.Sprintf("**Paper:** %s\n\n", p.Title))

	if id := p.Identifiers.DOI; id != "" {
		sb.WriteString(fmt.Sprintf("- DOI: [%s](https://doi.org/%s)\n", id, id))
	}
	if id := p.Identifiers.ArXivID; id != "" {
		sb.WriteString(fmt.Sprintf("- arXiv: [%s](https://arxiv.org/abs/%s)\n", id, id))
	}
	if p.Year > 0 {
		sb.WriteString(fmt.Sprintf("- Year: %d\n", p.Year))
	}
	if p.Venue != "" {
		sb.WriteString(fmt.Sprintf("- Venue: %s\n", p.Venue))
	}
	// No publisher — leave out
	sb.WriteString("\n")

	// OA status
	sb.WriteString("## Open Access Status\n\n")
	if resolution.OA != nil && resolution.OA.IsOA {
		sb.WriteString("- ✅ **Open access**")
		if resolution.OA.PDFURL != "" {
			sb.WriteString(fmt.Sprintf(" ([PDF](%s))", resolution.OA.PDFURL))
		}
		if resolution.OA.License != "" {
			sb.WriteString(fmt.Sprintf(" — license: %s", resolution.OA.License))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("- ⚠️ **No open-access copy detected** — check institutional access\n")
	}
	sb.WriteString("\n")

	// Code repositories
	sb.WriteString("## Code Repositories\n\n")
	if len(codeRepos) > 0 {
		for _, repo := range codeRepos {
			sb.WriteString(fmt.Sprintf("- [%s](https://%s)\n", repo, repo))
		}
	} else {
		sb.WriteString("_No GitHub repositories detected from paper metadata. Use `rforge oss search-plan` or manual lookup._\n")
	}
	sb.WriteString("\n")

	// Method summary from abstract
	sb.WriteString("## Method Summary\n\n")
	if p.Abstract != "" {
		sb.WriteString(p.Abstract + "\n\n")
	} else {
		sb.WriteString("_Abstract not available from source._\n\n")
	}

	// Replication checklist
	sb.WriteString("## Replication Checklist\n\n")
	sb.WriteString("| Step | Status | Notes |\n")
	sb.WriteString("|------|--------|-------|\n")
	sb.WriteString("| Environment (code + dependencies) | ")
	if len(codeRepos) > 0 {
		sb.WriteString("detected")
	} else {
		sb.WriteString("needs identification")
	}
	sb.WriteString(" | See Code Repositories section |\n")

	sb.WriteString("| Dataset acquisition | ")
	if resolution.OA != nil && resolution.OA.IsOA {
		sb.WriteString("partial")
		sb.WriteString(" | Paper is OA; dataset links need separate check |\n")
	} else {
		sb.WriteString("needs library access")
		sb.WriteString(" | Paper access required first |\n")
	}

	sb.WriteString("| Method implementation | needs review | Match abstract description against code |\n")
	sb.WriteString("| Hyperparameters | needs review | Check paper or linked code/configs |\n")
	sb.WriteString("| Evaluation metrics | needs review | Identify from paper methodology section |\n")
	sb.WriteString("| Baseline comparison | needs review | Identify comparison methods from paper |\n")
	sb.WriteString("| Reproducibility check | planned | Run with recorded environment + seeds |\n")
	sb.WriteString("\n")

	// Sources
	sb.WriteString("## Sources\n\n")
	for _, s := range resolution.Sources {
		sb.WriteString(fmt.Sprintf("- %s: resolved via %s\n", s.Source, s.MatchBy))
	}
	sb.WriteString("\n---\n*Generated by `rforge replicate`. Timestamp: ")
	sb.WriteString(time.Now().UTC().Format(time.RFC3339))
	sb.WriteString("*\n")

	plan := sb.String()

	if outFile != "" {
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "replicate_write_failed", err.Error())
		}
		if err := os.WriteFile(outFile, []byte(plan), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "replicate_write_failed", err.Error())
		}
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"paper":   map[string]any{"title": p.Title, "doi": p.Identifiers.DOI, "arxiv": p.Identifiers.ArXivID},
			"oa":      resolution.OA != nil && resolution.OA.IsOA,
			"code":    codeRepos,
			"planLen": len(plan),
			"out":     outFile,
		})
	}
	fmt.Fprint(stdout, plan)
	return 0
}
