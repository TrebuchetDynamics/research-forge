package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HistoryRecord is one matching paper in a research topic.
type HistoryRecord struct {
	Topic   string `json:"topic"`
	Title   string `json:"title"`
	DOI     string `json:"doi,omitempty"`
	ArXivID string `json:"arxivId,omitempty"`
	Year    int    `json:"year"`
	Matches int    `json:"matches"`
}

type HistoryOptions struct {
	ResearchDir, Term, Out string
	// Limit is required; zero returns no results, matching CLI --limit 0.
	Limit int
}
type HistoryResult struct {
	Term    string          `json:"term"`
	Count   int             `json:"count"`
	Results []HistoryRecord `json:"results"`
}

// History searches persisted sessions, retaining topic identity and match counts.
func (s Service) History(opts HistoryOptions) (HistoryResult, error) {
	researchDir, term, outFile, limit := opts.ResearchDir, opts.Term, opts.Out, opts.Limit
	if researchDir == "" {
		return HistoryResult{}, invalid("usage", "research directory is required")
	}
	if term == "" {
		return HistoryResult{}, invalid("missing_term", "search term is required")
	}
	if limit < 0 {
		return HistoryResult{}, invalid("invalid_limit", "limit must not be negative")
	}
	termLower := strings.ToLower(term)
	termWords := strings.Fields(termLower)

	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return HistoryResult{}, failure("history_read_failed", err)
	}

	var results []HistoryRecord
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
			results = append(results, HistoryRecord{
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
		if err := writeJSONArtifact(outFile, HistoryResult{Term: term, Count: len(results), Results: results}, nil); err != nil {
			return HistoryResult{}, failure("history_write_failed", err)
		}
	}

	return HistoryResult{Term: term, Count: len(results), Results: results}, nil
}
