package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func executeVault(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge vault build --research-dir <dir> --out <vault-dir>")
	}
	switch args[0] {
	case "build":
		return executeVaultBuild(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_vault_subcommand", fmt.Sprintf("unknown vault subcommand %q", args[0]))
	}
}

// ── build ─────────────────────────────────────────────────────────────────────

type vaultPaper struct {
	Key      string // DOI or arxiv_id — used for dedup
	Title    string
	Authors  string
	Year     int
	DOI      string
	ArXivID  string
	Abstract string
	Topics   []string // topic dirs that contain this paper
	Slug     string   // filename slug (no .md)
}

func executeVaultBuild(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	researchDir, outDir := "", ""
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--research-dir":
			researchDir = args[i+1]
			i++
		case "--out":
			outDir = args[i+1]
			i++
		}
	}
	if researchDir == "" || outDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge vault build --research-dir <dir> --out <vault-dir>")
	}

	// Discover topic subdirs
	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "research_dir_read_failed", err.Error())
	}

	// doi/arxiv → paper, tracking which topics each paper appears in
	byKey := map[string]*vaultPaper{}
	topicPapers := map[string][]string{} // topic → ordered keys

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		topic := entry.Name()
		recs, err := readResultsJSONL(filepath.Join(researchDir, topic, "results.jsonl"))
		if err != nil {
			continue
		}
		for _, rec := range recs {
			key := vaultPaperKey(rec)
			if key == "" {
				continue
			}
			if existing, ok := byKey[key]; ok {
				existing.Topics = appendIfMissing(existing.Topics, topic)
			} else {
				p := vaultPaperFromRecord(rec, key, topic)
				byKey[key] = &p
			}
			topicPapers[topic] = appendKeyIfMissing(topicPapers[topic], key)
		}
	}

	if len(byKey) == 0 {
		return writeError(stdout, stderr, opts, 1, "no_papers", "no papers found in research dir")
	}

	// Assign unique slugs
	slugsSeen := map[string]int{}
	for _, p := range byKey {
		base := vaultSlug(p.Title)
		if base == "" {
			base = "paper"
		}
		n := slugsSeen[base]
		slugsSeen[base]++
		if n == 0 {
			p.Slug = base
		} else {
			p.Slug = fmt.Sprintf("%s-%d", base, n+1)
		}
	}

	// Create output dirs
	papersDir := filepath.Join(outDir, "papers")
	if err := os.MkdirAll(papersDir, 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "mkdir_failed", err.Error())
	}

	// Write per-paper notes
	for _, p := range byKey {
		sort.Strings(p.Topics)
		content := vaultPaperNote(p)
		path := filepath.Join(papersDir, p.Slug+".md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "paper_note_write_failed", err.Error())
		}
	}

	// Slug lookup by key for wikilinks
	slugByKey := map[string]string{}
	for key, p := range byKey {
		slugByKey[key] = p.Slug
	}
	titleByKey := map[string]string{}
	for key, p := range byKey {
		titleByKey[key] = p.Title
	}

	// Write per-topic index notes
	topics := make([]string, 0, len(topicPapers))
	for t := range topicPapers {
		topics = append(topics, t)
	}
	sort.Strings(topics)

	topicCounts := map[string]int{}
	for _, topic := range topics {
		keys := topicPapers[topic]
		topicCounts[topic] = len(keys)
		content := vaultTopicNote(topic, keys, byKey)
		path := filepath.Join(outDir, topic+".md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "topic_note_write_failed", err.Error())
		}
	}

	// Cross-topic papers (appear in 2+ topics)
	var crossTopicPapers []*vaultPaper
	for _, p := range byKey {
		if len(p.Topics) >= 2 {
			crossTopicPapers = append(crossTopicPapers, p)
		}
	}
	sort.Slice(crossTopicPapers, func(i, j int) bool {
		if len(crossTopicPapers[i].Topics) != len(crossTopicPapers[j].Topics) {
			return len(crossTopicPapers[i].Topics) > len(crossTopicPapers[j].Topics)
		}
		return crossTopicPapers[i].Title < crossTopicPapers[j].Title
	})

	// Write main index
	indexContent := vaultMainIndex(topics, topicCounts, crossTopicPapers)
	if err := os.WriteFile(filepath.Join(outDir, "index.md"), []byte(indexContent), 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "index_write_failed", err.Error())
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"vault":       outDir,
			"papers":      len(byKey),
			"topics":      len(topics),
			"cross_topic": len(crossTopicPapers),
		})
	}
	fmt.Fprintf(stdout, "vault written to %s\n", outDir)
	fmt.Fprintf(stdout, "  %d papers  %d topics  %d cross-topic\n", len(byKey), len(topics), len(crossTopicPapers))
	return 0
}

// ── note templates ────────────────────────────────────────────────────────────

func vaultPaperNote(p *vaultPaper) string {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", p.Title))
	sb.WriteString(fmt.Sprintf("authors: %q\n", p.Authors))
	if p.Year > 0 {
		sb.WriteString(fmt.Sprintf("year: %d\n", p.Year))
	}
	if p.DOI != "" {
		sb.WriteString(fmt.Sprintf("doi: %q\n", p.DOI))
	}
	if p.ArXivID != "" {
		sb.WriteString(fmt.Sprintf("arxiv_id: %q\n", p.ArXivID))
	}
	sb.WriteString("topics:\n")
	for _, t := range p.Topics {
		sb.WriteString(fmt.Sprintf("  - %s\n", t))
	}
	sb.WriteString("tags:\n")
	sb.WriteString("  - paper\n")
	for _, t := range p.Topics {
		sb.WriteString(fmt.Sprintf("  - topic/%s\n", t))
	}
	if p.Year > 0 {
		sb.WriteString(fmt.Sprintf("  - year/%d\n", p.Year))
	}
	sb.WriteString("---\n\n")

	// Body
	sb.WriteString(fmt.Sprintf("# %s\n\n", p.Title))
	if p.Authors != "" {
		sb.WriteString(fmt.Sprintf("**Authors:** %s\n", p.Authors))
	}
	if p.Year > 0 {
		sb.WriteString(fmt.Sprintf("**Year:** %d\n", p.Year))
	}
	if p.DOI != "" {
		sb.WriteString(fmt.Sprintf("**DOI:** [%s](https://doi.org/%s)\n", p.DOI, p.DOI))
	}
	if p.ArXivID != "" {
		sb.WriteString(fmt.Sprintf("**arXiv:** [%s](https://arxiv.org/abs/%s)\n", p.ArXivID, p.ArXivID))
	}

	if len(p.Topics) > 0 {
		sb.WriteString("\n**Research topics:**")
		for _, t := range p.Topics {
			sb.WriteString(fmt.Sprintf(" [[%s]]", t))
		}
		sb.WriteString("\n")
	}

	if p.Abstract != "" {
		sb.WriteString("\n## Abstract\n\n")
		sb.WriteString(p.Abstract)
		sb.WriteString("\n")
	}

	return sb.String()
}

func vaultTopicNote(topic string, keys []string, byKey map[string]*vaultPaper) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("type: topic-index\n")
	sb.WriteString(fmt.Sprintf("topic: %s\n", topic))
	sb.WriteString(fmt.Sprintf("papers: %d\n", len(keys)))
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# %s\n\n", topic))
	sb.WriteString(fmt.Sprintf("**%d papers**\n\n", len(keys)))
	sb.WriteString("[[index]]\n\n")
	sb.WriteString("## Papers\n\n")

	for _, key := range keys {
		p := byKey[key]
		if p == nil {
			continue
		}
		crossMark := ""
		if len(p.Topics) > 1 {
			crossMark = " ✦"
		}
		sb.WriteString(fmt.Sprintf("- [[papers/%s|%s]]%s\n", p.Slug, p.Title, crossMark))
	}

	return sb.String()
}

func vaultMainIndex(topics []string, topicCounts map[string]int, crossTopicPapers []*vaultPaper) string {
	var sb strings.Builder

	totalTopics := len(topics)
	totalCross := len(crossTopicPapers)

	sb.WriteString("---\n")
	sb.WriteString("type: research-index\n")
	sb.WriteString(fmt.Sprintf("topics: %d\n", totalTopics))
	sb.WriteString(fmt.Sprintf("cross_topic_papers: %d\n", totalCross))
	sb.WriteString("---\n\n")

	sb.WriteString("# Research Index\n\n")
	sb.WriteString("## Topics\n\n")
	for _, t := range topics {
		sb.WriteString(fmt.Sprintf("- [[%s]] — %d papers\n", t, topicCounts[t]))
	}

	if len(crossTopicPapers) > 0 {
		sb.WriteString("\n## Cross-topic papers\n\n")
		sb.WriteString("Papers appearing in multiple research topics — likely core signal.\n\n")
		for _, p := range crossTopicPapers {
			topicsStr := strings.Join(topicsAsWikilinks(p.Topics), ", ")
			sb.WriteString(fmt.Sprintf("- [[papers/%s|%s]] (%d topics: %s)\n", p.Slug, p.Title, len(p.Topics), topicsStr))
		}
	}

	return sb.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func vaultPaperKey(rec library.PaperRecord) string {
	doi := strings.TrimSpace(rec.Identifiers.DOI)
	if doi != "" {
		return "doi:" + doi
	}
	arxiv := strings.TrimSpace(rec.Identifiers.ArXivID)
	if arxiv != "" {
		return "arxiv:" + arxiv
	}
	return ""
}

func vaultPaperFromRecord(rec library.PaperRecord, key, topic string) vaultPaper {
	authors := screenDirFormatAuthors(rec)
	return vaultPaper{
		Key:      key,
		Title:    rec.Title,
		Authors:  authors,
		Year:     rec.Year,
		DOI:      strings.TrimSpace(rec.Identifiers.DOI),
		ArXivID:  strings.TrimSpace(rec.Identifiers.ArXivID),
		Abstract: rec.Abstract,
		Topics:   []string{topic},
	}
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func vaultSlug(title string) string {
	s := strings.ToLower(title)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		// truncate at last hyphen before 80 chars
		s = s[:80]
		if idx := strings.LastIndex(s, "-"); idx > 40 {
			s = s[:idx]
		}
	}
	return s
}

func appendIfMissing(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func appendKeyIfMissing(slice []string, key string) []string {
	for _, v := range slice {
		if v == key {
			return slice
		}
	}
	return append(slice, key)
}

func topicsAsWikilinks(topics []string) []string {
	out := make([]string, len(topics))
	for i, t := range topics {
		out[i] = fmt.Sprintf("[[%s]]", t)
	}
	return out
}
