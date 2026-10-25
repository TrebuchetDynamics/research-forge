package research_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

func TestCompareRejectsDuplicateLabelsBeforeRead(t *testing.T) {
	for _, tc := range []struct {
		name   string
		labels []string
	}{
		{"default basenames", nil},
		{"explicit duplicates", []string{"same", "same"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			// Missing inputs make read-before-validation fail regardless of permissions.
			dirs := []string{filepath.Join(root, "a", "run"), filepath.Join(root, "b", "run")}
			out := filepath.Join(root, "comparison.md")
			if err := os.WriteFile(out, []byte("sentinel"), 0600); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(root, "absent")
			for _, path := range []string{out, filepath.Join(parent, "comparison.md")} {
				_, err := (research.Service{}).Compare(research.CompareOptions{Dirs: dirs, Labels: tc.labels, Out: path})
				var usage *research.Error
				if !errors.As(err, &usage) || usage.Code != "duplicate_labels" || usage.ExitCode != 2 {
					t.Fatalf("Compare error = %v, want duplicate_labels / exit 2", err)
				}
				if !strings.Contains(err.Error(), "--labels") || !strings.Contains(err.Error(), "distinct") {
					t.Errorf("error lacks distinct --labels remedy: %v", err)
				}
			}
			if data, err := os.ReadFile(out); err != nil || string(data) != "sentinel" {
				t.Errorf("output changed: %q, %v", data, err)
			}
			if _, err := os.Stat(parent); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("absent output parent created: %v", err)
			}
		})
	}

	for _, labels := range [][]string{{"first", "second"}, {"Run", "run"}, {"run", "run "}} {
		t.Run(strings.Join(labels, ","), func(t *testing.T) {
			root := t.TempDir()
			a := session(t, root, "a/run", `{"title":"Shared","identifiers":{"DOI":"10.1/shared"}}
{"title":"First","identifiers":{"DOI":"10.1/first"}}`)
			b := session(t, root, "b/run", `{"title":"Shared","identifiers":{"DOI":"10.1/shared"}}
{"title":"Second","identifiers":{"DOI":"10.1/second"}}`)
			got, err := (research.Service{}).Compare(research.CompareOptions{Dirs: []string{a, b}, Labels: labels})
			if err != nil {
				t.Fatal(err)
			}
			want := []research.ComparisonPaper{
				{Title: "Shared", ID: "10.1/shared", In: []string{labels[0], labels[1]}},
				{Title: "First", ID: "10.1/first", In: []string{labels[0]}},
				{Title: "Second", ID: "10.1/second", In: []string{labels[1]}},
			}
			if got.Total != 3 || !reflect.DeepEqual(got.Labels, labels) || !reflect.DeepEqual(got.Papers, want) {
				t.Fatalf("comparison membership = %+v, want %+v", got, want)
			}
			for _, fragment := range []string{
				"| # | Title | " + labels[0] + " | " + labels[1] + " | ID | Abs |",
				"| 1 | Shared | ✓ | ✓ |", "| 2 | First | ✓ |   |", "| 3 | Second |   | ✓ |",
				"**Total unique papers across all dirs:** 3", "**In all directories:** 1", "**Unique to one directory:** 2",
			} {
				if !strings.Contains(got.Markdown, fragment) {
					t.Errorf("Markdown missing %q:\n%s", fragment, got.Markdown)
				}
			}
		})
	}
}
