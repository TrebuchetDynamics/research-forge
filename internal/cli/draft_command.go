package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// executeDraft generates a structured research-note draft from a resolved paper.
func executeDraft(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(stdout, `Generate a structured research-note draft from a paper ID or search directory.

Usage:
  rforge draft <doi|arxiv-id> [--out <draft.md>]
  rforge draft --search-dir <dir> [--out <draft.md>]

Examples:
  rforge draft 10.1136/bmj.n160 --out draft.md
  rforge draft 2301.12345
  rforge draft --search-dir research/my-topic --out research/my-topic/draft.md`)
		return 0
	}

	outFile := ""
	paperInput := ""
	searchDir := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 < len(args) {
				outFile = args[i+1]
				i++
			}
		case "--search-dir":
			if i+1 < len(args) {
				searchDir = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				paperInput = args[i]
			}
		}
	}

	if searchDir != "" {
		return draftFromSearchDir(searchDir, outFile, stdout, stderr, opts)
	}
	if paperInput == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge draft <doi|arxiv-id> [--out <draft.md>]")
	}

	httpClient := sources.NewHTTPClient(sources.HTTPClientOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resolution, err := sources.ResolvePaper(ctx, httpClient, paperInput)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "draft_resolve_failed", fmt.Sprintf("resolve paper: %v", err))
	}
	if !resolution.Resolved {
		return writeError(stdout, stderr, opts, 1, "draft_unresolved", fmt.Sprintf("could not resolve %q", paperInput))
	}

	draft := buildPaperDraft(resolution.Paper, resolution)

	if outFile != "" {
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "draft_write_failed", err.Error())
		}
		if err := os.WriteFile(outFile, []byte(draft), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "draft_write_failed", err.Error())
		}
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"paperTitle": resolution.Paper.Title,
			"sources":    len(resolution.Sources),
			"draftSize":  len(draft),
			"out":        outFile,
		})
	}
	fmt.Fprint(stdout, draft)
	if outFile != "" {
		fmt.Fprintf(stdout, "\nwrote draft to %s\n", outFile)
	}
	return 0
}

func buildPaperDraft(p library.PaperRecord, resolution sources.PaperResolution) string {
	var sb strings.Builder
	sb.WriteString("# Research Note\n\n")
	sb.WriteString("## Paper\n\n")
	fmt.Fprintf(&sb, "**%s**\n\n", p.Title)
	if p.Year > 0 {
		fmt.Fprintf(&sb, "**Year:** %d  \n", p.Year)
	}
	if p.Venue != "" {
		fmt.Fprintf(&sb, "**Venue:** %s  \n", p.Venue)
	}
	if p.Publisher != "" {
		fmt.Fprintf(&sb, "**Publisher:** %s  \n", p.Publisher)
	}
	sb.WriteString("\n")

	sb.WriteString("### Identifiers\n\n")
	if id := p.Identifiers.DOI; id != "" {
		fmt.Fprintf(&sb, "- DOI: [%s](https://doi.org/%s)\n", id, id)
	}
	if id := p.Identifiers.ArXivID; id != "" {
		fmt.Fprintf(&sb, "- arXiv: [%s](https://arxiv.org/abs/%s)\n", id, id)
	}
	if id := p.Identifiers.OpenAlexID; id != "" {
		fmt.Fprintf(&sb, "- OpenAlex: %s\n", id)
	}
	sb.WriteString("\n")

	sb.WriteString("### Open Access\n\n")
	if resolution.OA != nil && resolution.OA.IsOA {
		fmt.Fprintf(&sb, "- ✅ Open access (%s)", resolution.OA.OAStatus)
		if resolution.OA.PDFURL != "" {
			fmt.Fprintf(&sb, " — [PDF](%s)", resolution.OA.PDFURL)
		}
		if resolution.OA.License != "" {
			fmt.Fprintf(&sb, " — license: %s", resolution.OA.License)
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("- ⚠️ No open-access copy detected\n")
	}
	if resolution.OA != nil && len(resolution.OA.OtherOAURLs) > 0 {
		for _, u := range resolution.OA.OtherOAURLs {
			fmt.Fprintf(&sb, "  - %s\n", u)
		}
	}
	sb.WriteString("\n")

	if len(p.URLs) > 0 {
		sb.WriteString("### URLs\n\n")
		for _, u := range p.URLs {
			fmt.Fprintf(&sb, "- %s\n", u)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Abstract\n\n")
	if p.Abstract != "" {
		sb.WriteString(p.Abstract + "\n\n")
	} else {
		sb.WriteString("_Abstract not available._\n\n")
	}

	sb.WriteString("## Method Signals\n\n")
	sb.WriteString(fmt.Sprintln(extractMethodSignals(p.Abstract)))
	sb.WriteString("\n")

	sb.WriteString("## Key Claims\n\n")
	sb.WriteString("| # | Claim | Evidence | Certainty |\n")
	sb.WriteString("|---|-------|---------|-----------|\n")
	sb.WriteString("| 1 | _Fill from paper_ | _Fill from results_ | _High/Medium/Low_ |\n")
	sb.WriteString("| 2 | _Fill from paper_ | _Fill from results_ | _High/Medium/Low_ |\n")
	sb.WriteString("| 3 | _Fill from paper_ | _Fill from results_ | _High/Medium/Low_ |\n\n")

	sb.WriteString("## Limitations\n\n")
	sb.WriteString("| Limitation | Impact | Mitigation |\n")
	sb.WriteString("|------------|--------|------------|\n")
	sb.WriteString("| _Fill_ | _Fill_ | _Fill_ |\n")
	sb.WriteString("| _Fill_ | _Fill_ | _Fill_ |\n\n")

	sb.WriteString("## Sources\n\n")
	for _, s := range resolution.Sources {
		fmt.Fprintf(&sb, "- **%s:** resolved via %s\n", s.Source, s.MatchBy)
	}
	sb.WriteString("\n---\n")
	fmt.Fprintf(&sb, "*Draft generated by `rforge draft`. Date: %s*\n", time.Now().UTC().Format(time.RFC3339))

	return sb.String()
}

func draftFromSearchDir(searchDir, outFile string, stdout, stderr io.Writer, opts globalOptions) int {
	manifestData, err := os.ReadFile(filepath.Join(searchDir, "manifest.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "draft_manifest_read_failed", err.Error())
	}
	var manifest struct {
		Queries  []string `json:"queries"`
		Sources  []string `json:"sources"`
		Results  int      `json:"results"`
		Deduped  int      `json:"deduped"`
		Failures int      `json:"failures"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return writeError(stdout, stderr, opts, 1, "draft_manifest_parse_failed", err.Error())
	}

	topic := ""
	if len(manifest.Queries) > 0 {
		topic = manifest.Queries[0]
	}

	papers, _ := readBriefPaperResults(searchDir)

	var sb strings.Builder
	sb.WriteString("# Research Note\n\n")
	fmt.Fprintf(&sb, "**Topic:** %s\n\n", topic)
	sb.WriteString("## Summary\n\n")
	fmt.Fprintf(&sb, "- **Queries:** %d  \n", len(manifest.Queries))
	fmt.Fprintf(&sb, "- **Sources:** %s  \n", strings.Join(manifest.Sources, ", "))
	fmt.Fprintf(&sb, "- **Total records:** %d  \n", manifest.Results)
	fmt.Fprintf(&sb, "- **Deduplicated:** %d  \n", manifest.Deduped)
	fmt.Fprintf(&sb, "- **Failures:** %d  \n", manifest.Failures)
	sb.WriteString("\n")

	sb.WriteString("## Queries\n\n")
	for _, q := range manifest.Queries {
		fmt.Fprintf(&sb, "- `%s`\n", q)
	}
	sb.WriteString("\n")

	sb.WriteString("## Top Papers\n\n")
	sb.WriteString("| Title | Year | OA |\n")
	sb.WriteString("|-------|------|----|\n")
	for _, p := range papers {
		oa := ""
		if p.OpenAccess {
			oa = "🔓"
		}
		yr := ""
		if p.Year > 0 {
			yr = fmt.Sprintf("%d", p.Year)
		}
		fmt.Fprintf(&sb, "| %s | %s | %s |\n", p.Title, yr, oa)
	}
	sb.WriteString("\n")

	sb.WriteString("## Claims & Evidence\n\n")
	sb.WriteString("| # | Claim | Evidence Location | Certainty |\n")
	sb.WriteString("|---|-------|------------------|-----------|\n")
	sb.WriteString("| 1 | _Fill after full-text review_ | _Fill_ | _Fill_ |\n\n")

	sb.WriteString("## Limitations & Gaps\n\n")
	if manifest.Failures > 0 {
		fmt.Fprintf(&sb, "- **%d query/source pair(s) failed** — results may be incomplete\n", manifest.Failures)
	}
	sb.WriteString("- Screening decisions not yet applied\n\n")

	sb.WriteString("## Next Steps\n\n")
	sb.WriteString("1. Screen papers for inclusion/exclusion\n")
	sb.WriteString("2. Fetch open-access PDFs for included papers\n")
	sb.WriteString("3. Extract evidence from selected papers\n")
	sb.WriteString("4. Synthesize findings\n\n")

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "*Draft generated by `rforge draft`. Date: %s*\n", time.Now().UTC().Format(time.RFC3339))

	draft := sb.String()
	if outFile != "" {
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "draft_write_failed", err.Error())
		}
		if err := os.WriteFile(outFile, []byte(draft), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "draft_write_failed", err.Error())
		}
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"topic":     topic,
			"papers":    len(papers),
			"draftSize": len(draft),
			"out":       outFile,
		})
	}
	fmt.Fprint(stdout, draft)
	if outFile != "" {
		fmt.Fprintf(stdout, "\nwrote draft to %s\n", outFile)
	}
	return 0
}

func extractMethodSignals(abstract string) string {
	if abstract == "" {
		return "_No abstract — method signals not available_"
	}
	signals := []string{}
	keywords := []string{"dataset", "corpus", "benchmark", "experiment", "evaluation", "metric",
		"propose", "method", "approach", "architecture", "model", "training",
		"supervised", "unsupervised", "reinforcement", "classification",
		"neural", "deep learning", "transformer", "fine-tune", "embedding"}
	lower := strings.ToLower(abstract)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			signals = append(signals, kw)
		}
	}
	if len(signals) == 0 {
		return "_No method signals detected in abstract_"
	}
	seen := map[string]bool{}
	uniq := make([]string, 0, len(signals))
	for _, s := range signals {
		if !seen[s] {
			seen[s] = true
			uniq = append(uniq, s)
		}
	}
	return strings.Join(uniq, ", ")
}

// readBriefPaperResults reads up to 20 deduped paper records from a search dir.
