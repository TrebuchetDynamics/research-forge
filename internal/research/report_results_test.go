package research_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// Damaged rows must not silently turn a report into a partial corpus; valid
// sparse/large records and the legacy reader must retain their old semantics.
func TestReportResultsRejectMalformedJSONL(t *testing.T) {
	const alpha = `{"title":"Alpha","year":2026,"identifiers":{"DOI":"10.1/a"}}`
	const beta = `{"title":"Beta","year":2020,"identifiers":{"DOI":"10.1/b"}}`
	longTitle := strings.Repeat("L", 65537)
	for _, tc := range []struct {
		name, rows string
		bad, typed bool
		titles     []string
	}{
		{name: "malformed", rows: alpha + "\n\n \t\n" + `{"title":"private-row",` + "\n" + beta + "\nnot JSON", bad: true},
		{name: "type_invalid", rows: alpha + "\n\n \t\n" + `{"title":"private-row","year":"invalid"}` + "\n" + beta, bad: true, typed: true},
		{name: "empty"},
		{name: "blank", rows: "\n \t\r\n"},
		{name: "sparse_unknown_crlf", rows: "null\r\n{}\r\n{\"unknown\":true}\r\n" + alpha[:len(alpha)-1] + ",\"unknown\":true}\r\n" + beta + "\r\n", titles: []string{"Alpha", "Beta"}},
		{name: "long_row", rows: `{"title":"` + longTitle + `","identifiers":{"DOI":"10.1/long"}}`, titles: []string{longTitle}},
	} {
		for _, operation := range []string{"brief", "compare_first", "compare_second"} {
			if !tc.bad && operation == "compare_second" {
				continue
			}
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				root := t.TempDir()
				a := session(t, root, "a", tc.rows)
				b := session(t, root, "b", tc.rows)
				badDir, wantCode := a, "brief_results_parse_failed"
				if tc.bad && operation != "brief" {
					goodDir := b
					wantCode = "compare_read_0_failed"
					if operation == "compare_second" {
						badDir, goodDir = b, a
						wantCode = "compare_read_1_failed"
					}
					if err := os.WriteFile(filepath.Join(goodDir, "results-deduped.jsonl"), []byte(alpha), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				outputs := []string{"existing"}
				if tc.bad {
					outputs = append(outputs, "absent_parent")
				}
				for _, output := range outputs {
					out := filepath.Join(root, "report.md")
					if output == "existing" {
						if err := os.WriteFile(out, []byte("sentinel"), 0o600); err != nil {
							t.Fatal(err)
						}
					} else {
						out = filepath.Join(root, "new", "report.md")
					}
					var result any
					var zero any
					var titles []string
					var markdown string
					var err error
					if operation == "brief" {
						var got research.BriefResult
						got, err = (research.Service{}).Brief(research.BriefOptions{SearchDir: a, Out: out})
						result, zero, markdown = got, research.BriefResult{}, got.Markdown
						for _, p := range got.Papers {
							titles = append(titles, p.Title)
						}
						if !tc.bad && (got.TotalRecords != 2 || got.Deduped != 2 || got.BriefSize != len(markdown)) {
							t.Error("brief metadata changed")
						}
					} else {
						var got research.CompareResult
						got, err = (research.Service{}).Compare(research.CompareOptions{Dirs: []string{a, b}, Out: out})
						result, zero, markdown = got, research.CompareResult{}, got.Markdown
						for _, p := range got.Papers {
							titles = append(titles, p.Title)
							if !tc.bad && !reflect.DeepEqual(p.In, []string{"a", "b"}) {
								t.Errorf("membership = %v", p.In)
							}
						}
						if !tc.bad && got.Total != len(tc.titles) {
							t.Errorf("total = %d, want %d", got.Total, len(tc.titles))
						}
					}
					if tc.bad {
						var failure *research.Error
						if !errors.As(err, &failure) || failure.Code != wantCode || failure.ExitCode != 1 {
							t.Fatalf("error = %v, want %s / exit 1", err, wantCode)
						}
						path := filepath.Join(badDir, "results-deduped.jsonl")
						if !strings.Contains(err.Error(), path+":4:") || strings.Contains(err.Error(), "private-row") {
							t.Errorf("missing path/physical line or exposed row: %v", err)
						}
						var syntax *json.SyntaxError
						var typeError *json.UnmarshalTypeError
						if tc.typed && !errors.As(err, &typeError) || !tc.typed && !errors.As(err, &syntax) {
							t.Errorf("JSON cause lost: %v", err)
						}
						if !reflect.DeepEqual(result, zero) {
							t.Error("failure returned a partial result")
						}
						if output == "absent_parent" {
							if _, err := os.Stat(filepath.Dir(out)); !errors.Is(err, os.ErrNotExist) {
								t.Errorf("output parent created: %v", err)
							}
						} else if data, err := os.ReadFile(out); err != nil || string(data) != "sentinel" {
							t.Errorf("output changed: %v", err)
						}
					} else {
						if err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(titles, tc.titles) {
							t.Errorf("titles differ (got %d, want %d)", len(titles), len(tc.titles))
						}
						data, err := os.ReadFile(out)
						if err != nil || string(data) != markdown || !strings.HasPrefix(markdown, "# ") {
							t.Errorf("report persistence changed: %v", err)
						}
						if operation != "brief" && !strings.Contains(markdown, fmt.Sprintf("(%d papers)", len(tc.titles))) {
							t.Error("comparison input count changed")
						}
					}
				}
				for _, dir := range []string{a, b} {
					want := tc.rows
					if tc.bad && dir != badDir && operation != "brief" {
						want = alpha
					}
					if data, err := os.ReadFile(filepath.Join(dir, "results-deduped.jsonl")); err != nil || string(data) != want {
						t.Errorf("input changed: %v", err)
					}
				}
			})
		}
	}
	t.Run("legacy_reader_stays_permissive", func(t *testing.T) {
		dir := session(t, t.TempDir(), "legacy", alpha+"\nnot JSON\n"+beta)
		papers, err := research.ReadPaperResults(dir)
		if err != nil || len(papers) != 2 || papers[0].Title != "Alpha" || papers[1].Title != "Beta" {
			t.Fatalf("legacy reader = %+v, %v", papers, err)
		}
	})
}
