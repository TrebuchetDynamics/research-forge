package library

import "testing"

func TestScoreDuplicateMatchesExactDOI(t *testing.T) {
	left := mustPaper(t, PaperRecordInput{Title: "Catalyst A", Identifiers: Identifiers{DOI: "https://doi.org/10.1000/EXAMPLE"}})
	right := mustPaper(t, PaperRecordInput{Title: "Different title", Identifiers: Identifiers{DOI: "10.1000/example"}})

	match := ScoreDuplicate(left, right)
	if !match.Duplicate || match.Score != 1 || match.Reason != "exact_doi" {
		t.Fatalf("match = %#v", match)
	}
}

func TestScoreDuplicateMatchesNormalizedArXivID(t *testing.T) {
	left := mustPaper(t, PaperRecordInput{Title: "Preprint A", Identifiers: Identifiers{ArXivID: "arXiv:2401.00001"}})
	right := mustPaper(t, PaperRecordInput{Title: "Preprint A v2", Identifiers: Identifiers{ArXivID: "2401.00001v2"}})

	match := ScoreDuplicate(left, right)
	if !match.Duplicate || match.Score != 0.95 || match.Reason != "exact_arxiv" {
		t.Fatalf("match = %#v", match)
	}
}

func TestScoreDuplicateUsesFuzzyTitleAuthorYearWithFalsePositiveBoundary(t *testing.T) {
	left := mustPaper(t, PaperRecordInput{
		Title:       "Artificial photosynthesis catalyst review",
		Identifiers: Identifiers{DOI: "10.1000/left"},
		Authors:     []Author{{Family: "Lovelace"}},
		Year:        2026,
	})
	right := mustPaper(t, PaperRecordInput{
		Title:       "Artificial photosynthesis catalysts: a review",
		Identifiers: Identifiers{DOI: "10.1000/right"},
		Authors:     []Author{{Family: "Lovelace"}},
		Year:        2026,
	})
	unrelated := mustPaper(t, PaperRecordInput{
		Title:       "Perovskite solar cell durability",
		Identifiers: Identifiers{DOI: "10.1000/other"},
		Authors:     []Author{{Family: "Lovelace"}},
		Year:        2026,
	})

	match := ScoreDuplicate(left, right)
	if !match.Duplicate || match.Reason != "fuzzy_title_author_year" {
		t.Fatalf("fuzzy match = %#v", match)
	}
	boundary := ScoreDuplicate(left, unrelated)
	if boundary.Duplicate {
		t.Fatalf("false positive duplicate = %#v", boundary)
	}
}

func TestMergeDuplicatePreservesIdentifiersAndSourceProvenance(t *testing.T) {
	left := mustPaper(t, PaperRecordInput{
		Title:       "Artificial photosynthesis catalyst review",
		Identifiers: Identifiers{DOI: "10.1000/example", OpenAlexID: "W123"},
		SourceRefs:  []SourceRef{{Source: "openalex", RawPayloadRef: "openalex:/works?search=x"}},
	})
	right := mustPaper(t, PaperRecordInput{
		Title:       "Artificial photosynthesis catalyst review",
		Identifiers: Identifiers{ArXivID: "2401.00001", CrossrefID: "10.1000/example"},
		SourceRefs:  []SourceRef{{Source: "crossref", RawPayloadRef: "crossref:/works?query=x"}},
	})

	merged := MergeDuplicate(left, right)
	if merged.Identifiers.DOI != "10.1000/example" || merged.Identifiers.OpenAlexID != "W123" || merged.Identifiers.ArXivID != "2401.00001" || merged.Identifiers.CrossrefID != "10.1000/example" {
		t.Fatalf("identifiers = %#v", merged.Identifiers)
	}
	if len(merged.SourceRefs) != 2 || merged.SourceRefs[0].Source != "openalex" || merged.SourceRefs[1].Source != "crossref" {
		t.Fatalf("source refs = %#v", merged.SourceRefs)
	}
}

func mustPaper(t *testing.T, input PaperRecordInput) PaperRecord {
	t.Helper()
	paper, err := NewPaperRecord(input)
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	return paper
}
