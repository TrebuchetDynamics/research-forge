package library

import (
	"math"
	"strings"
)

// DuplicateMatch describes duplicate scoring between two PaperRecords.
type DuplicateMatch struct {
	Duplicate bool
	Score     float64
	Reason    string
}

// ScoreDuplicate scores whether two paper records represent the same scholarly work.
func ScoreDuplicate(left, right PaperRecord) DuplicateMatch {
	leftDOI := normalizeDOI(left.Identifiers.DOI)
	rightDOI := normalizeDOI(right.Identifiers.DOI)
	if leftDOI != "" && rightDOI != "" && leftDOI == rightDOI {
		return DuplicateMatch{Duplicate: true, Score: 1, Reason: "exact_doi"}
	}
	leftArXiv := normalizeArXivDuplicateID(left.Identifiers.ArXivID)
	rightArXiv := normalizeArXivDuplicateID(right.Identifiers.ArXivID)
	if leftArXiv != "" && rightArXiv != "" && leftArXiv == rightArXiv {
		return DuplicateMatch{Duplicate: true, Score: 0.95, Reason: "exact_arxiv"}
	}
	if left.Year != 0 && right.Year != 0 && left.Year == right.Year && firstAuthorFamily(left) != "" && firstAuthorFamily(left) == firstAuthorFamily(right) {
		similarity := tokenJaccard(left.Title, right.Title)
		if similarity >= 0.6 {
			return DuplicateMatch{Duplicate: true, Score: round2(0.7 + similarity*0.2), Reason: "fuzzy_title_author_year"}
		}
	}
	return DuplicateMatch{Duplicate: false, Score: 0, Reason: "no_match"}
}

// MergeDuplicate merges metadata for two duplicate papers while preserving all identifiers and provenance.
func MergeDuplicate(left, right PaperRecord) PaperRecord {
	merged := left
	if merged.Identifiers.DOI == "" {
		merged.Identifiers.DOI = right.Identifiers.DOI
	}
	if merged.Identifiers.ArXivID == "" {
		merged.Identifiers.ArXivID = right.Identifiers.ArXivID
	}
	if merged.Identifiers.PMID == "" {
		merged.Identifiers.PMID = right.Identifiers.PMID
	}
	if merged.Identifiers.OpenAlexID == "" {
		merged.Identifiers.OpenAlexID = right.Identifiers.OpenAlexID
	}
	if merged.Identifiers.CrossrefID == "" {
		merged.Identifiers.CrossrefID = right.Identifiers.CrossrefID
	}
	if merged.Identifiers.SemanticScholarID == "" {
		merged.Identifiers.SemanticScholarID = right.Identifiers.SemanticScholarID
	}
	merged.SourceRefs = append(append([]SourceRef{}, left.SourceRefs...), right.SourceRefs...)
	return merged
}

func normalizeArXivDuplicateID(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "arXiv:")
	if idx := strings.LastIndex(value, "v"); idx > 0 {
		allDigits := true
		for _, r := range value[idx+1:] {
			if r < '0' || r > '9' {
				allDigits = false
			}
		}
		if allDigits {
			value = value[:idx]
		}
	}
	return value
}

func firstAuthorFamily(paper PaperRecord) string {
	if len(paper.Authors) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(paper.Authors[0].Family))
}

func tokenJaccard(left, right string) float64 {
	leftTokens := tokenSet(left)
	rightTokens := tokenSet(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}
	intersection := 0
	for token := range leftTokens {
		if rightTokens[token] {
			intersection++
		}
	}
	union := len(leftTokens) + len(rightTokens) - intersection
	return float64(intersection) / float64(union)
}

func tokenSet(value string) map[string]bool {
	set := map[string]bool{}
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if token != "" && token != "a" && token != "the" {
			set[token] = true
		}
	}
	return set
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
