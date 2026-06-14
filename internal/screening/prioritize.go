package screening

import (
	"sort"
	"strings"
	"unicode"
)

// ScreeningRecord is the minimal text needed to prioritize records for human screening.
type ScreeningRecord struct {
	ID       string
	Title    string
	Abstract string
}

// PrioritizedRecord is an unscreened record plus its deterministic relevance score.
type PrioritizedRecord struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

// PrioritizeActiveLearningRecords is a deterministic ASReview-style scaffold:
// it ranks unscreened records higher when their title/abstract overlaps with
// included seed records and lower when they overlap with excluded seed records.
func PrioritizeActiveLearningRecords(records []ScreeningRecord, events []DecisionEvent, stage Stage) []PrioritizedRecord {
	decided := map[string]bool{}
	positive := map[string]int{}
	negative := map[string]int{}
	byID := map[string]ScreeningRecord{}
	for _, record := range records {
		byID[record.ID] = record
	}
	for _, event := range events {
		if event.Stage != stage {
			continue
		}
		decided[event.PaperID] = true
		record, ok := byID[event.PaperID]
		if !ok {
			continue
		}
		for _, token := range recordTokens(record) {
			switch event.Decision {
			case DecisionInclude:
				positive[token]++
			case DecisionExclude:
				negative[token]++
			}
		}
	}
	prioritized := []PrioritizedRecord{}
	for _, record := range records {
		if record.ID == "" || decided[record.ID] {
			continue
		}
		score := 0
		seen := map[string]bool{}
		for _, token := range recordTokens(record) {
			if seen[token] {
				continue
			}
			seen[token] = true
			score += positive[token]
			score -= negative[token]
		}
		prioritized = append(prioritized, PrioritizedRecord{ID: record.ID, Score: float64(score)})
	}
	sort.SliceStable(prioritized, func(i, j int) bool {
		if prioritized[i].Score != prioritized[j].Score {
			return prioritized[i].Score > prioritized[j].Score
		}
		return prioritized[i].ID < prioritized[j].ID
	})
	return prioritized
}

func recordTokens(record ScreeningRecord) []string {
	return tokenizeScreeningText(record.Title + " " + record.Abstract)
}

func tokenizeScreeningText(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := []string{}
	for _, field := range fields {
		if len(field) < 3 || screeningStopWords[field] {
			continue
		}
		tokens = append(tokens, field)
	}
	return tokens
}

var screeningStopWords = map[string]bool{
	"and": true, "are": true, "for": true, "from": true, "into": true, "the": true, "this": true, "with": true,
	"using": true, "review": true, "study": true, "paper": true,
}
