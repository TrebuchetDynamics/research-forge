package screening

import (
	"math"
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
	ID                string  `json:"id"`
	Score             float64 `json:"score"`
	Uncertainty       float64 `json:"uncertainty,omitempty"`
	ExploitationScore float64 `json:"exploitationScore,omitempty"`
	ExplorationScore  float64 `json:"explorationScore,omitempty"`
	Policy            string  `json:"policy,omitempty"`
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

// PrioritizeUncertaintyRecords ranks unscreened records closest to the model boundary.
func PrioritizeUncertaintyRecords(records []ScreeningRecord, events []DecisionEvent, stage Stage) []PrioritizedRecord {
	prioritized := PrioritizeActiveLearningRecords(records, events, stage)
	for i := range prioritized {
		prioritized[i].Uncertainty = 1 / (1 + absFloat(prioritized[i].Score))
	}
	sort.SliceStable(prioritized, func(i, j int) bool {
		if prioritized[i].Uncertainty != prioritized[j].Uncertainty {
			return prioritized[i].Uncertainty > prioritized[j].Uncertainty
		}
		if prioritized[i].Score != prioritized[j].Score {
			return prioritized[i].Score > prioritized[j].Score
		}
		return prioritized[i].ID < prioritized[j].ID
	})
	return prioritized
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

// PrioritizeModelRecords ranks unscreened records with a smoothed naive-Bayes text model.
func PrioritizeModelRecords(records []ScreeningRecord, events []DecisionEvent, stage Stage) []PrioritizedRecord {
	decided := map[string]bool{}
	positiveDocs, negativeDocs := 0, 0
	positiveCounts := map[string]int{}
	negativeCounts := map[string]int{}
	vocab := map[string]bool{}
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
		seen := map[string]bool{}
		for _, token := range recordTokens(record) {
			if seen[token] {
				continue
			}
			seen[token] = true
			vocab[token] = true
			switch event.Decision {
			case DecisionInclude:
				positiveCounts[token]++
			case DecisionExclude:
				negativeCounts[token]++
			}
		}
		switch event.Decision {
		case DecisionInclude:
			positiveDocs++
		case DecisionExclude:
			negativeDocs++
		}
	}
	if positiveDocs == 0 || negativeDocs == 0 {
		return PrioritizeActiveLearningRecords(records, events, stage)
	}
	vocabSize := float64(len(vocab))
	if vocabSize == 0 {
		vocabSize = 1
	}
	positiveDenom := float64(positiveDocs) + vocabSize
	negativeDenom := float64(negativeDocs) + vocabSize
	prior := math.Log(float64(positiveDocs+1) / float64(negativeDocs+1))
	prioritized := []PrioritizedRecord{}
	for _, record := range records {
		if record.ID == "" || decided[record.ID] {
			continue
		}
		logOdds := prior
		seen := map[string]bool{}
		for _, token := range recordTokens(record) {
			if seen[token] {
				continue
			}
			seen[token] = true
			pos := (float64(positiveCounts[token]) + 1) / positiveDenom
			neg := (float64(negativeCounts[token]) + 1) / negativeDenom
			logOdds += math.Log(pos / neg)
		}
		probability := 1 / (1 + math.Exp(-logOdds))
		prioritized = append(prioritized, PrioritizedRecord{ID: record.ID, Score: probability, Uncertainty: 1 - math.Abs(probability-0.5)*2})
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
