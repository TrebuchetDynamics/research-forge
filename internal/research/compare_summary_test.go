package research_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// Catches unique rows counted as shared, shared-in-two mistaken for in-all,
// and duplicate/no-ID input records leaking into identifiable counts.
func TestCompareSharedCounts(t *testing.T) {
	const (
		ab   = `{"title":"AB","identifiers":{"DOI":"10.1/ab"}}`
		abc  = `{"title":"ABC","identifiers":{"DOI":"10.1/abc"}}`
		a    = `{"title":"A-only","identifiers":{"ArXivID":"2301.00001"}}`
		b    = `{"title":"B-only","identifiers":{"DOI":"10.1/b"}}`
		noID = `{"title":"No identifier"}`
	)
	for _, tc := range []struct {
		name    string
		inputs  []string
		summary string
		papers  []research.ComparisonPaper
	}{
		{
			name: "disjoint", inputs: []string{a, b},
			summary: `- **Total unique papers across all dirs:** 2
- **In all directories:** 0
- **Unique to one directory:** 2
- **A:** 1 papers (0 shared, 1 unique)
- **B:** 1 papers (0 shared, 1 unique)`,
			papers: []research.ComparisonPaper{
				{Title: "A-only", ID: "2301.00001", In: []string{"A"}},
				{Title: "B-only", ID: "10.1/b", In: []string{"B"}},
			},
		},
		{
			name: "mixed two-way", inputs: []string{ab + "\n" + a, ab + "\n" + b},
			summary: `- **Total unique papers across all dirs:** 3
- **In all directories:** 1
- **Unique to one directory:** 2
- **A:** 2 papers (1 shared, 1 unique)
- **B:** 2 papers (1 shared, 1 unique)`,
			papers: []research.ComparisonPaper{
				{Title: "AB", ID: "10.1/ab", In: []string{"A", "B"}},
				{Title: "A-only", ID: "2301.00001", In: []string{"A"}},
				{Title: "B-only", ID: "10.1/b", In: []string{"B"}},
			},
		},
		{
			name: "partial three-way", inputs: []string{ab + "\n" + abc + "\n" + a, ab + "\n" + abc, abc},
			summary: `- **Total unique papers across all dirs:** 3
- **In all directories:** 1
- **Unique to one directory:** 1
- **A:** 3 papers (2 shared, 1 unique)
- **B:** 2 papers (2 shared, 0 unique)
- **C:** 1 papers (1 shared, 0 unique)`,
			papers: []research.ComparisonPaper{
				{Title: "ABC", ID: "10.1/abc", In: []string{"A", "B", "C"}},
				{Title: "AB", ID: "10.1/ab", In: []string{"A", "B"}},
				{Title: "A-only", ID: "2301.00001", In: []string{"A"}},
			},
		},
		{
			name: "all-way", inputs: []string{abc, abc, abc},
			summary: `- **Total unique papers across all dirs:** 1
- **In all directories:** 1
- **Unique to one directory:** 0
- **A:** 1 papers (1 shared, 0 unique)
- **B:** 1 papers (1 shared, 0 unique)
- **C:** 1 papers (1 shared, 0 unique)`,
			papers: []research.ComparisonPaper{
				{Title: "ABC", ID: "10.1/abc", In: []string{"A", "B", "C"}},
			},
		},
		{
			name: "empty", inputs: []string{"", ""},
			summary: `- **Total unique papers across all dirs:** 0
- **In all directories:** 0
- **Unique to one directory:** 0
- **A:** 0 papers (0 shared, 0 unique)
- **B:** 0 papers (0 shared, 0 unique)`,
			papers: []research.ComparisonPaper{},
		},
		{
			name: "one empty", inputs: []string{a, ""},
			summary: `- **Total unique papers across all dirs:** 1
- **In all directories:** 0
- **Unique to one directory:** 1
- **A:** 1 papers (0 shared, 1 unique)
- **B:** 0 papers (0 shared, 0 unique)`,
			papers: []research.ComparisonPaper{
				{Title: "A-only", ID: "2301.00001", In: []string{"A"}},
			},
		},
		{
			name: "duplicates and no-ID records",
			inputs: []string{
				strings.Join([]string{ab, ab, a, a, noID}, "\n"),
				strings.Join([]string{ab, noID}, "\n"),
			},
			summary: `- **Total unique papers across all dirs:** 2
- **In all directories:** 1
- **Unique to one directory:** 1
- **A:** 5 papers (1 shared, 1 unique)
- **B:** 2 papers (1 shared, 0 unique)`,
			papers: []research.ComparisonPaper{
				{Title: "AB", ID: "10.1/ab", In: []string{"A", "B"}},
				{Title: "A-only", ID: "2301.00001", In: []string{"A"}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			labels := []string{"A", "B", "C"}[:len(tc.inputs)]
			var dirs []string
			for i, input := range tc.inputs {
				dirs = append(dirs, session(t, root, labels[i], input))
			}
			got, err := (research.Service{}).Compare(research.CompareOptions{Dirs: dirs, Labels: labels})
			if err != nil {
				t.Fatal(err)
			}
			_, summary, found := strings.Cut(got.Markdown, "## Summary\n\n")
			summary, _, _ = strings.Cut(summary, "\n\n## Gaps")
			if !found || summary != tc.summary {
				t.Errorf("summary =\n%s\nwant:\n%s", summary, tc.summary)
			}
			if got.Total != len(tc.papers) || !reflect.DeepEqual(got.Labels, labels) || !reflect.DeepEqual(got.Papers, tc.papers) {
				t.Errorf("comparison membership/order = %+v, want %+v", got, tc.papers)
			}
		})
	}
}
