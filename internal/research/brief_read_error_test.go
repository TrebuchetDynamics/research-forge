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

// Lost inputs must not become empty reports or modify existing artifacts.
func TestBriefRejectsResultsReadFailure(t *testing.T) {
	for _, tc := range []struct {
		name, rows, wantCode   string
		directory, badManifest bool
		wantPapers             int
	}{
		{name: "missing", wantCode: "brief_results_read_failed"},
		{name: "directory", directory: true, wantCode: "brief_results_read_failed"},
		{name: "manifest_first", badManifest: true, wantCode: "brief_manifest_parse_failed"},
		{name: "empty"},
		{name: "blank", rows: "\n \t\n"},
		{name: "readable_rows", rows: "not JSON\n{}\n{\"title\":\"Paper\",\"year\":2026}\n", wantPapers: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := session(t, root, "search", tc.rows)
			manifestPath := filepath.Join(dir, "manifest.json")
			resultsPath := filepath.Join(dir, "results-deduped.jsonl")
			if tc.wantCode != "" {
				if err := os.Remove(resultsPath); err != nil {
					t.Fatal(err)
				}
			}
			if tc.directory {
				if err := os.Mkdir(resultsPath, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if tc.badManifest {
				if err := os.WriteFile(manifestPath, []byte("{"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			manifest, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, output := range []string{"existing", "absent_parent"} {
				out := filepath.Join(root, "brief.md")
				if output == "existing" {
					if err := os.WriteFile(out, []byte("sentinel"), 0o600); err != nil {
						t.Fatal(err)
					}
				} else {
					out = filepath.Join(root, "new", "brief.md")
				}
				got, err := (research.Service{}).Brief(research.BriefOptions{SearchDir: dir, Out: out})
				if tc.wantCode != "" {
					var failure *research.Error
					if !errors.As(err, &failure) || failure.Code != tc.wantCode || failure.ExitCode != 1 {
						t.Fatalf("%s: error = %v, want %s / exit 1", output, err, tc.wantCode)
					}
					if !reflect.DeepEqual(got, research.BriefResult{}) {
						t.Errorf("failure returned nonzero result: %+v", got)
					}
					if !tc.badManifest {
						var cause *os.PathError
						if !errors.As(err, &cause) || cause.Path != resultsPath || cause.Err == nil || !strings.Contains(err.Error(), resultsPath) {
							t.Fatalf("missing results filename/cause: %v", err)
						}
						if !tc.directory && !errors.Is(err, os.ErrNotExist) {
							t.Errorf("missing-file cause lost: %v", err)
						}
					}
					if output == "existing" {
						if data, err := os.ReadFile(out); err != nil || string(data) != "sentinel" {
							t.Errorf("output changed: %q, %v", data, err)
						}
					} else if _, err := os.Stat(filepath.Dir(out)); !os.IsNotExist(err) {
						t.Errorf("output parent created: %v", err)
					}
				} else {
					if err != nil || len(got.Papers) != tc.wantPapers {
						t.Fatalf("readable results: %+v, %v", got, err)
					}
					if strings.Contains(got.Markdown, "_No deduplicated records found_") != (tc.wantPapers == 0) || !strings.Contains(got.Markdown, "- **Searched:** 2026-01-01\n") {
						t.Errorf("unexpected readable brief: %s", got.Markdown)
					}
					if data, err := os.ReadFile(out); err != nil || string(data) != got.Markdown {
						t.Errorf("saved brief differs: %v", err)
					}
				}
			}
			if after, err := os.ReadFile(manifestPath); err != nil || string(after) != string(manifest) {
				t.Errorf("manifest changed: %v", err)
			}
			switch {
			case tc.directory:
				if entries, err := os.ReadDir(resultsPath); err != nil || len(entries) != 0 {
					t.Errorf("results directory changed: %v, %v", entries, err)
				}
			case tc.wantCode != "":
				if _, err := os.Stat(resultsPath); !os.IsNotExist(err) {
					t.Errorf("missing results created: %v", err)
				}
			default:
				if data, err := os.ReadFile(resultsPath); err != nil || string(data) != tc.rows {
					t.Errorf("results changed: %q, %v", data, err)
				}
			}
		})
	}
}
