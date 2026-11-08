package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSearchStatsCheckpointReadErrors(t *testing.T) {
	const updated = "2026-10-23T12:34:56Z"
	const interrupted = `{"queries":["alpha","beta","gamma"],"sources":["openalex"],"completed":[{"source":"openalex","query":"alpha"}],"failures":[{"source":"openalex","query":"beta","error":"timeout"}],"interrupted":true,"updatedAt":"` + updated + `"}`
	const completed = `{"queries":["alpha"],"sources":["openalex"],"completed":[{"source":"openalex","query":"alpha"}],"updatedAt":"` + updated + `"}`
	const legacy = `{"queries":["alpha"],"sources":["openalex"],"completed":[{"source":"openalex","query":"alpha"}]}`
	const failures = `{"source":"openalex","query":"beta","error":"timeout"}` + "\n" + `{"source":"arxiv","query":"gamma","error":"unavailable"}` + "\n"
	for _, mode := range []string{"text", "json"} {
		for _, tc := range []struct {
			name       string
			content    string
			diagnostic string
			want       *searchBatchCheckpointSummaryResult
		}{
			{name: "directory", diagnostic: "is a directory"},
			{name: "empty", diagnostic: "unexpected end of JSON input"},
			{name: "whitespace", content: " \n\t", diagnostic: "unexpected end of JSON input"},
			{name: "invalid", content: "not JSON", diagnostic: "invalid character"},
			{name: "truncated", content: `{"queries":[`, diagnostic: "unexpected end of JSON input"},
			{name: "field type", content: `{"queries":17}`, diagnostic: "cannot unmarshal number"},
			{name: "missing topic"},
			{name: "missing checkpoint"},
			{name: "empty object", content: `{}`, want: &searchBatchCheckpointSummaryResult{}},
			{name: "null", content: `null`, want: &searchBatchCheckpointSummaryResult{}},
			{name: "legacy", content: legacy, want: &searchBatchCheckpointSummaryResult{Completed: 1, Queries: 1, Sources: []string{"openalex"}}},
			{name: "interrupted", content: interrupted, want: &searchBatchCheckpointSummaryResult{Completed: 1, Failed: 1, Pending: 1, Interrupted: true, UpdatedAt: updated, Queries: 3, Sources: []string{"openalex"}, ResumeStream: true}},
			{name: "completed", content: completed, want: &searchBatchCheckpointSummaryResult{Completed: 1, UpdatedAt: updated, Queries: 1, Sources: []string{"openalex"}}},
			{name: "symlink", content: completed, want: &searchBatchCheckpointSummaryResult{Completed: 1, UpdatedAt: updated, Queries: 1, Sources: []string{"openalex"}}},
			{name: "dangling symlink"},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				root := t.TempDir()
				dir := filepath.Join(root, "topic")
				if tc.name != "missing topic" {
					writeSearchRawFile(t, dir, "openalex", []string{"10.1000/a", "10.1000/b"})
					writeSearchRawFile(t, dir, "arxiv", []string{"10.1000/a"})
					results := strings.Repeat(`{"Title":"Paper","Identifiers":{"DOI":"10.1000/a"}}`+"\n", 2) + "{\n"
					if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte(results), 0o600); err != nil {
						t.Fatal(err)
					}
					writeSearchFailures(t, dir, failures)
				}
				path := filepath.Join(dir, "batch-state.json")
				switch tc.name {
				case "missing topic", "missing checkpoint":
				case "directory":
					if err := os.Mkdir(path, 0o755); err != nil {
						t.Fatal(err)
					}
				case "symlink", "dangling symlink":
					target := filepath.Join(root, "checkpoint.json")
					if tc.name == "symlink" {
						if err := os.WriteFile(target, []byte(tc.content), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				default:
					if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
						t.Fatal(err)
					}
				}

				// Include every path, including jobs/locks and link targets.
				snapshot := func() map[string]string {
					t.Helper()
					entries := map[string]string{}
					err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
						if err != nil {
							return err
						}
						var content string
						if entry.Type()&os.ModeSymlink != 0 {
							content, err = os.Readlink(path)
						} else if !entry.IsDir() {
							var data []byte
							data, err = os.ReadFile(path)
							content = string(data)
						}
						entries[path] = entry.Type().String() + ":" + content
						return err
					})
					if err != nil {
						t.Fatal(err)
					}
					return entries
				}
				before := snapshot()
				t.Cleanup(func() {
					if !reflect.DeepEqual(snapshot(), before) {
						t.Error("stats changed fixture paths/bytes or created jobs/locks")
					}
				})

				args := []string{"search", "stats", "--dir", dir}
				if mode == "json" {
					args = append([]string{"--json"}, args...)
				}
				stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
				code := Execute(args, stdout, stderr)
				wantCode := 0
				if tc.diagnostic != "" {
					wantCode = 1
				}
				if code != wantCode {
					t.Fatalf("exit=%d, want %d; stdout=%s stderr=%s", code, wantCode, stdout, stderr)
				}
				if wantCode == 1 {
					message := stderr.String()
					if mode == "json" {
						var envelope map[string]json.RawMessage
						if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
							t.Fatal(err)
						}
						var failure struct{ Code, Message string }
						if err := json.Unmarshal(envelope["error"], &failure); err != nil {
							t.Fatal(err)
						}
						if len(envelope) != 2 || string(envelope["ok"]) != "false" || failure.Code != "search_stats_state_read_failed" || stderr.Len() != 0 {
							t.Fatalf("expected error only, no partial stats: stdout=%s stderr=%s", stdout, stderr)
						}
						message = failure.Message
					} else if stdout.Len() != 0 {
						t.Fatalf("unexpected success stdout: %s", stdout)
					}
					if !strings.HasPrefix(message, "read batch-state.json: ") || !strings.Contains(message, path) || !strings.Contains(message, tc.diagnostic) {
						t.Fatalf("missing checkpoint path/original cause: %s", message)
					}
					return
				}

				if stderr.Len() != 0 {
					t.Fatalf("unexpected stderr: %s", stderr)
				}
				if mode == "json" {
					var envelope struct {
						OK   bool           `json:"ok"`
						Data map[string]any `json:"data"`
					}
					if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
						t.Fatal(err)
					}
					want := map[string]any{"sources": map[string]int{"arxiv": 1, "openalex": 2}, "sourceFiles": map[string]int{"arxiv": 1, "openalex": 1}, "totalUniqueDOIs": 2, "libraryRecords": 2, "malformedRecords": 1, "failures": []searchBatchFailure{{Source: "openalex", Query: "beta", Error: "timeout"}, {Source: "arxiv", Query: "gamma", Error: "unavailable"}}}
					if tc.name == "missing topic" {
						want = map[string]any{"sources": map[string]int{}, "sourceFiles": map[string]int{}, "totalUniqueDOIs": 0, "libraryRecords": 0, "malformedRecords": 0, "failures": []searchBatchFailure{}}
					}
					if s := tc.want; s != nil {
						want["batchState"] = map[string]any{
							"completed": s.Completed, "failed": s.Failed, "pending": s.Pending,
							"interrupted": s.Interrupted, "updatedAt": s.UpdatedAt,
							"queries": s.Queries, "sources": s.Sources, "resumeRecommended": s.ResumeStream,
						}
					}
					wantJSON, err := json.Marshal(want)
					if err != nil {
						t.Fatal(err)
					}
					var wantData map[string]any
					if err := json.Unmarshal(wantJSON, &wantData); err != nil {
						t.Fatal(err)
					}
					if !envelope.OK || !reflect.DeepEqual(envelope.Data, wantData) {
						t.Fatalf("stats=%s, want %s", stdout, wantJSON)
					}
				} else {
					want := fmt.Sprintf("Source coverage for %s\n  %-24s 1 records (1 files)\n  %-24s 2 records (1 files)\n\nTotal unique DOIs: 2\nLibrary (results.jsonl): 2 records (1 malformed line skipped)\n\nFailed queries (2):\n  [openalex] %q — timeout\n  [arxiv] %q — unavailable\n", dir, "arxiv", "openalex", "beta", "gamma")
					if tc.name == "missing topic" {
						want = fmt.Sprintf("Source coverage for %s\n\nTotal unique DOIs: 0\nLibrary (results.jsonl): 0 records\n", dir)
					}
					if s := tc.want; s != nil {
						want += fmt.Sprintf("\nBatch checkpoint: %d completed, %d failed, %d pending pair(s)\n", s.Completed, s.Failed, s.Pending)
						if s.Interrupted {
							want += "  sweep interrupted at " + updated + "\n"
						}
						if s.ResumeStream {
							want += "  resume with: rforge search batch --resume --queries <file> --sources openalex --out " + dir + "\n"
						}
					}
					if stdout.String() != want {
						t.Fatalf("stats=%q, want %q", stdout.String(), want)
					}
				}
			})
		}
	}
}
