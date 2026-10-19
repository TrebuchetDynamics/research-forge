package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// litSession stores one topic dir's data for synthesis.
type litSession struct {
	Topic    string
	Manifest *litManifest
	Papers   []litPaper
	Decided  *litDecisions
}

type litManifest struct {
	Queries   []string `json:"queries"`
	Sources   []string `json:"sources"`
	Results   int      `json:"results"`
	Deduped   int      `json:"deduped"`
	Failures  int      `json:"failures"`
	CreatedAt string   `json:"createdAt"`
}

type litPaper struct {
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Venue       string `json:"venue"`
	Abstract    string `json:"abstract"`
	Identifiers struct {
		DOI     string `json:"DOI"`
		ArXivID string `json:"ArXivID"`
	} `json:"identifiers"`
	OpenAccess bool   `json:"openAccess"`
	Source     string `json:"source"`
}

type litDecisions struct {
	Included  int
	Excluded  int
	Uncertain int
}

// keywordCount tracks a term across papers.
type keywordCount struct {
	Term   string
	Count  int
	Papers int
}

const litStopwords = "the a an is in of to and that for on with as by at from are this these those their its our was be not we or also we have has been"

// executeLit produces a deterministic literature review: consensus themes, open questions, screening status.
func executeLit(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	researchDir := ""
	outDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--research-dir":
			if i+1 < len(args) {
				researchDir = args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(args) {
				outDir = args[i+1]
				i++
			}
		}
	}
	if researchDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge lit --research-dir <dir> [--out <dir>]")
	}
	if outDir == "" {
		outDir = researchDir
	}

	// Collect topic sessions
	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "lit_read_failed", err.Error())
	}

	var sessions []litSession
	totalUniqueDOIs := map[string]struct{}{}
	doiToTopics := map[string][]string{}
	doiToPaper := map[string]litPaper{}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		topic := e.Name()
		s, ok := loadLitSession(filepath.Join(researchDir, topic), topic)
		if !ok {
			continue
		}
		sessions = append(sessions, s)
		for _, p := range s.Papers {
			doi := p.Identifiers.DOI
			if doi == "" {
				doi = p.Identifiers.ArXivID
			}
			if doi == "" {
				continue
			}
			totalUniqueDOIs[doi] = struct{}{}
			doiToTopics[doi] = append(doiToTopics[doi], topic)
			if _, exists := doiToPaper[doi]; !exists {
				doiToPaper[doi] = p
			}
		}
	}

	if len(sessions) == 0 {
		return writeError(stdout, stderr, opts, 1, "lit_no_sessions", "no valid topic directories found")
	}

	// Build keyword consensus: collect tokens from titles + abstracts
	stopSet := map[string]bool{}
	for _, w := range strings.Fields(litStopwords) {
		stopSet[w] = true
	}
	keywordCounts := map[string]*keywordCount{}
	totalPapers := 0
	for _, s := range sessions {
		for _, p := range s.Papers {
			totalPapers++
			seen := map[string]bool{}
			text := strings.ToLower(p.Title + " " + p.Abstract)
			for _, token := range strings.Fields(text) {
				token = strings.Trim(token, ".,;:!?\"'()-[]{}")
				if len(token) < 4 || stopSet[token] {
					continue
				}
				if !isAlphaWord(token) {
					continue
				}
				// Light singularization: plural -> singular for common -s/-es forms,
				// but never mangle words ending in -is/-us (analysis, focus, status).
				if strings.HasSuffix(token, "ies") && len(token) > 4 {
					token = token[:len(token)-3] + "y"
				} else if strings.HasSuffix(token, "es") && len(token) > 4 && !strings.HasSuffix(token, "ies") {
					token = token[:len(token)-2]
				} else if strings.HasSuffix(token, "s") && len(token) > 4 && !strings.HasSuffix(token, "is") && !strings.HasSuffix(token, "us") && !strings.HasSuffix(token, "ss") {
					token = token[:len(token)-1]
				}
				if len(token) < 4 || stopSet[token] {
					continue
				}
				if seen[token] {
					continue
				}
				seen[token] = true
				if _, ok := keywordCounts[token]; !ok {
					keywordCounts[token] = &keywordCount{Term: token}
				}
				keywordCounts[token].Count++
				keywordCounts[token].Papers++
			}
		}
	}

	// Rank keywords by paper count
	var ranked []keywordCount
	for _, kc := range keywordCounts {
		if kc.Papers >= 2 { // at least 2 papers mention it
			ranked = append(ranked, *kc)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Papers != ranked[j].Papers {
			return ranked[i].Papers > ranked[j].Papers
		}
		return ranked[i].Count > ranked[j].Count
	})
	if len(ranked) > 30 {
		ranked = ranked[:30]
	}

	// Cross-topic papers
	crossTopic := map[string][]string{}
	for doi, topics := range doiToTopics {
		if len(topics) > 1 {
			crossTopic[doi] = topics
		}
	}

	// Summary stats
	totalDeduped := 0
	totalResults := 0
	totalFailures := 0
	hasScreening := false
	var sessionSummaries []string
	for _, s := range sessions {
		if s.Manifest != nil {
			totalResults += s.Manifest.Results
			totalDeduped += s.Manifest.Deduped
			totalFailures += s.Manifest.Failures
		}
		sc := ""
		if s.Decided != nil {
			hasScreening = true
			sc = fmt.Sprintf(", %d incl/%d excl/%d uncertain", s.Decided.Included, s.Decided.Excluded, s.Decided.Uncertain)
		}
		sessionSummaries = append(sessionSummaries, fmt.Sprintf("- **%s:** %d papers%s", s.Topic, len(s.Papers), sc))
	}

	// --- Build markdown ---
	var sb strings.Builder
	sb.WriteString("# Literature Synthesis\n\n")
	sb.WriteString(fmt.Sprintf("Generated from **%d topics** with **%d unique papers** across **%d total deduplicated records**.\n\n", len(sessions), len(totalUniqueDOIs), totalDeduped))

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n|---|---|\n"))
	sb.WriteString(fmt.Sprintf("| Topics | %d |\n", len(sessions)))
	sb.WriteString(fmt.Sprintf("| Total records (pre-dedupe) | %d |\n", totalResults))
	sb.WriteString(fmt.Sprintf("| Deduplicated records | %d |\n", totalDeduped))
	sb.WriteString(fmt.Sprintf("| Unique papers | %d |\n", len(totalUniqueDOIs)))
	sb.WriteString(fmt.Sprintf("| Failed query/source pairs | %d |\n", totalFailures))
	if hasScreening {
		sb.WriteString("| Screening decisions present | yes |\n")
	}
	sb.WriteString("\n")

	// Per-topic breakdown
	sb.WriteString("## Topics\n\n")
	for _, s := range sessionSummaries {
		sb.WriteString(s + "\n")
	}
	sb.WriteString("\n")

	// Consensus themes
	sb.WriteString("## Consensus Themes\n\n")
	if len(ranked) == 0 {
		sb.WriteString("_No recurring concepts found across abstracts._\n\n")
	} else {
		sb.WriteString("| Term | Papers | Occurrences |\n")
		sb.WriteString("|------|--------|-------------|\n")
		for _, kc := range ranked {
			if kc.Count > 999 {
				sb.WriteString(fmt.Sprintf("| %s | %d | %d |\n", kc.Term, kc.Papers, kc.Count))
			}
		}
		// Show top 20
		limit := 20
		if len(ranked) < limit {
			limit = len(ranked)
		}
		sb.WriteString("\n**Top recurring terms:** ")
		terms := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			terms = append(terms, fmt.Sprintf("`%s` (%d papers)", ranked[i].Term, ranked[i].Papers))
		}
		sb.WriteString(strings.Join(terms, ", "))
		sb.WriteString("\n\n")
	}

	// Cross-topic papers
	sb.WriteString("## Cross-Topic Papers\n\n")
	if len(crossTopic) == 0 {
		sb.WriteString("_No papers appear in more than one topic._\n\n")
	} else {
		sb.WriteString("| Paper | Topics |\n")
		sb.WriteString("|-------|--------|\n")
		// Sort by number of topics
		type ct struct {
			doi    string
			title  string
			topics []string
		}
		var cts []ct
		for doi, topics := range crossTopic {
			p := doiToPaper[doi]
			cts = append(cts, ct{doi: doi, title: p.Title, topics: topics})
		}
		sort.Slice(cts, func(i, j int) bool {
			return len(cts[i].topics) > len(cts[j].topics)
		})
		for _, c := range cts {
			title := c.title
			if t, ok := doiToPaper[c.doi]; ok && t.Title != "" {
				title = t.Title
			}
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", title, strings.Join(c.topics, ", ")))
		}
		sb.WriteString("\n")
	}

	// Open questions
	sb.WriteString("## Open Questions & Gaps\n\n")
	noAbstract := 0
	for _, s := range sessions {
		for _, p := range s.Papers {
			if strings.TrimSpace(p.Abstract) == "" {
				noAbstract++
			}
		}
	}
	if totalFailures > 0 {
		sb.WriteString(fmt.Sprintf("- **%d failed query/source pair(s)** — retry with `rforge search resume`\n", totalFailures))
	}
	if noAbstract > 0 {
		sb.WriteString(fmt.Sprintf("- **%d records lack abstracts** — these were not available from the source API; consider `rforge oa fetch` + `rforge parse` to extract full text\n", noAbstract))
	}
	if !hasScreening {
		sb.WriteString("- **No screening decisions recorded** — run `rforge screen queue --dir <dir> --out queue.csv`, fill decisions, then `rforge screen import`\n")
	}
	var noOATopics []string
	for _, s := range sessions {
		oaCount := 0
		for _, p := range s.Papers {
			if p.OpenAccess {
				oaCount++
			}
		}
		if oaCount == 0 && len(s.Papers) > 0 {
			noOATopics = append(noOATopics, s.Topic)
		}
	}
	if len(noOATopics) > 0 {
		sb.WriteString(fmt.Sprintf("- **Topics with zero OA papers:** %s\n", strings.Join(noOATopics, ", ")))
	}
	sb.WriteString("\n")

	// Sources table
	sb.WriteString("## Source Coverage\n\n")
	sourceTotals := map[string]int{}
	for _, s := range sessions {
		if s.Manifest != nil {
			for _, src := range s.Manifest.Sources {
				sourceTotals[src]++
			}
		}
	}
	if len(sourceTotals) > 0 {
		sb.WriteString("| Source | Topics Covered |\n")
		sb.WriteString("|--------|----------------|\n")
		for _, src := range sortedKeys(sourceTotals) {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", src, sourceTotals[src]))
		}
		sb.WriteString("\n")
	}

	// Provenance
	sb.WriteString("---\n\n*Generated by `rforge lit`. Date: ")
	sb.WriteString(time.Now().UTC().Format(time.RFC3339))
	sb.WriteString("*\n")

	lit := sb.String()
	outPath := filepath.Join(outDir, "LIT.md")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "lit_write_failed", err.Error())
	}
	if err := os.WriteFile(outPath, []byte(lit), 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "lit_write_failed", err.Error())
	}

	// Write provenance
	prov := map[string]any{
		"schema_version": "1",
		"command":        "rforge lit",
		"research_dir":   researchDir,
		"topics":         len(sessions),
		"total_papers":   totalPapers,
		"unique_dois":    len(totalUniqueDOIs),
		"keyword_themes": len(ranked),
		"cross_topic":    len(crossTopic),
		"failures":       totalFailures,
		"no_abstract":    noAbstract,
		"has_screening":  hasScreening,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}
	provData, _ := json.MarshalIndent(prov, "", "  ")
	provPath := filepath.Join(outDir, "LIT.provenance.json")
	os.WriteFile(provPath, provData, 0o644)

	if opts.JSON {
		return writeJSON(stdout, 0, prov)
	}
	fmt.Fprintf(stdout, "wrote LIT.md and LIT.provenance.json to %s\n", outDir)
	return 0
}

func isAlphaWord(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
		_ = r
	}
	return len(s) > 0
}

func loadLitSession(dir, topic string) (litSession, bool) {
	s := litSession{Topic: topic}

	// Manifest
	md, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err == nil {
		var m litManifest
		if json.Unmarshal(md, &m) == nil {
			s.Manifest = &m
		}
	}

	// Papers from deduped results
	dd, err := os.ReadFile(filepath.Join(dir, "results-deduped.jsonl"))
	if err != nil {
		// fallback to results.jsonl
		dd, err = os.ReadFile(filepath.Join(dir, "results.jsonl"))
		if err != nil {
			return s, false
		}
	}
	for _, line := range strings.Split(string(dd), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var p litPaper
		if json.Unmarshal([]byte(line), &p) == nil && p.Title != "" {
			s.Papers = append(s.Papers, p)
		}
	}

	// Screening decisions
	sd, err := os.ReadFile(filepath.Join(dir, "screening.jsonl"))
	if err == nil {
		dec := &litDecisions{}
		for _, line := range strings.Split(string(sd), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var entry struct {
				Decision string `json:"decision"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil {
				switch entry.Decision {
				case "include":
					dec.Included++
				case "exclude":
					dec.Excluded++
				default:
					dec.Uncertain++
				}
			}
		}
		s.Decided = dec
	}

	return s, len(s.Papers) > 0
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
