package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func executeMeta(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge meta overlap --research-dir <dir> [--min-topics N]")
	}
	switch args[0] {
	case "overlap":
		return executeMetaOverlap(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_meta_subcommand", fmt.Sprintf("unknown meta subcommand %q", args[0]))
	}
}

type metaOverlapEntry struct {
	DOI        string   `json:"doi"`
	Title      string   `json:"title"`
	TopicCount int      `json:"topicCount"`
	Topics     []string `json:"topics"`
}

func executeMetaOverlap(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	researchDir := ""
	minTopics := 1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--research-dir":
			if i+1 < len(args) {
				researchDir = args[i+1]
				i++
			}
		case "--min-topics":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &minTopics)
				i++
			}
		}
	}
	if researchDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge meta overlap --research-dir <dir> [--min-topics N]")
	}

	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "meta_overlap_read_failed", err.Error())
	}

	// doi → {title, set of topics}
	type paperState struct {
		title  string
		topics map[string]struct{}
	}
	papers := map[string]*paperState{}
	topicCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		topic := entry.Name()
		resultsPath := filepath.Join(researchDir, topic, "results.jsonl")
		records, err := readResultsJSONL(resultsPath)
		if err != nil || len(records) == 0 {
			continue
		}
		topicCount++
		seen := map[string]struct{}{}
		for _, r := range records {
			doi := strings.TrimSpace(r.Identifiers.DOI)
			if doi == "" || r.Title == "" {
				continue
			}
			if _, already := seen[doi]; already {
				continue
			}
			seen[doi] = struct{}{}
			if _, ok := papers[doi]; !ok {
				papers[doi] = &paperState{title: r.Title, topics: map[string]struct{}{}}
			}
			papers[doi].topics[topic] = struct{}{}
		}
	}

	// Build sorted result list
	var results []metaOverlapEntry
	for doi, ps := range papers {
		if len(ps.topics) < minTopics {
			continue
		}
		topics := make([]string, 0, len(ps.topics))
		for t := range ps.topics {
			topics = append(topics, t)
		}
		sort.Strings(topics)
		results = append(results, metaOverlapEntry{
			DOI:        doi,
			Title:      ps.title,
			TopicCount: len(ps.topics),
			Topics:     topics,
		})
	}
	// Sort: most topics first, then alphabetically by title
	sort.Slice(results, func(i, j int) bool {
		if results[i].TopicCount != results[j].TopicCount {
			return results[i].TopicCount > results[j].TopicCount
		}
		return results[i].Title < results[j].Title
	})

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"topics":    topicCount,
			"totalDOIs": len(papers),
			"papers":    results,
		})
	}

	fmt.Fprintf(stdout, "%d topics · %d unique papers · %d appear in >1 topic\n\n",
		topicCount, len(papers), countAbove(results, 1))
	for _, r := range results {
		fmt.Fprintf(stdout, "[%d] %s\n    %s\n    %s\n",
			r.TopicCount, r.Title, r.DOI, strings.Join(r.Topics, ", "))
	}
	return 0
}

func countAbove(results []metaOverlapEntry, threshold int) int {
	n := 0
	for _, r := range results {
		if r.TopicCount > threshold {
			n++
		}
	}
	return n
}
