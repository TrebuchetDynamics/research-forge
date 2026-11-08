package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func TestSearchRefreshWriteFailure(t *testing.T) {
	// The append seam, normalizer and transport are restored per nonparallel case.
	for _, site := range []string{"unknown-source", "connector", "normalization", "result"} {
		for _, format := range []string{"text", "json"} {
			variants := []string{"append-failure", "sparse-append-failure", "success"}
			if site == "result" {
				variants = append(variants, "dry-run")
			}
			for _, variant := range variants {
				t.Run(site+"/"+format+"/"+variant, func(t *testing.T) {
					failAppend := strings.Contains(variant, "append-failure")
					queries, sourceNames := []string{"first", "pending"}, []string{"openalex"}
					if site == "unknown-source" {
						sourceNames = []string{"unknown-fixture-source", "openalex"}
					}
					out := t.TempDir()
					writePriorTopic(t, out, queries, sourceNames, []string{"10.0000/prior-a", "10.0000/prior-b"})
					prior, err := loadSearchBatchResults(filepath.Join(out, "results-deduped.jsonl"))
					if err != nil {
						t.Fatal(err)
					}
					state := searchBatchState{SchemaVersion: "1", CreatedAt: "2026-07-01T00:00:00Z", Queries: queries, Sources: sourceNames, Limit: 20}
					if variant != "sparse-append-failure" {
						if err := writeSearchBatchJSONL(filepath.Join(out, "results.jsonl"), prior); err != nil {
							t.Fatal(err)
						}
						for name, data := range map[string]string{"failures.jsonl": "", "results.md": "# Prior results\n"} {
							if err := os.WriteFile(filepath.Join(out, name), []byte(data), 0o640); err != nil {
								t.Fatal(err)
							}
						}
						for i, query := range queries {
							rawName := filepath.Join("raw", fmt.Sprintf("search-openalex-%03d-%s.txt", i+1, query))
							if err := os.MkdirAll(filepath.Join(out, "raw"), 0o750); err != nil {
								t.Fatal(err)
							}
							if err := writeSearchBatchRaw(filepath.Join(out, rawName), prior[i:i+1]); err != nil {
								t.Fatal(err)
							}
							if err := os.Chmod(filepath.Join(out, rawName), 0o640); err != nil {
								t.Fatal(err)
							}
							state.Completed = append(state.Completed, searchBatchStatePair{Source: "openalex", Query: query, RawFile: rawName, Records: 1})
						}
					}
					if err := writeSearchBatchState(out, state); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(out, "notes.txt"), []byte("unrelated notes\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					before := snapshotRefreshArtifacts(t, out)

					t.Setenv("RFORGE_OPENALEX_URL", "https://refresh.invalid")
					t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
					calls := 0
					transport := http.DefaultTransport
					t.Cleanup(func() { http.DefaultTransport = transport })
					http.DefaultTransport = rankFixtureTransport(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.URL.Host != "refresh.invalid" {
							t.Fatalf("unexpected connector request: %s", r.URL)
						}
						status := http.StatusOK
						body := `{"results":[{"id":"https://openalex.org/Wa","doi":"https://doi.org/10.0000/a","title":"A"},{"id":"https://openalex.org/Wb","doi":"https://doi.org/10.0000/b","title":"B"}]}`
						if site == "connector" && calls == 1 {
							status, body = http.StatusNotFound, "synthetic connector failure"
						}
						return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					})
					normalize := sources.PaperRecords
					t.Cleanup(func() { sources.PaperRecords = normalize })
					normalizations := 0
					sources.PaperRecords = func(response sources.SourceResponse) ([]library.PaperRecord, error) {
						normalizations++
						if site == "normalization" && normalizations == 1 {
							return nil, errors.New("synthetic normalization failure")
						}
						return normalize(response)
					}
					injected := errors.New("synthetic refresh append failure")
					appendRecord := appendSearchRefreshRecord
					t.Cleanup(func() { appendSearchRefreshRecord = appendRecord })
					appends := 0
					appendSearchRefreshRecord = func(w io.Writer, value any) error {
						appends++
						if appends == 1 {
							_, isFailure := value.(searchBatchFailure)
							if isFailure != (site != "result") {
								t.Fatalf("first append did not reach %s: %T", site, value)
							}
							if failAppend {
								return injected
							}
						}
						return appendRecord(w, value)
					}
					args := []string{"search", "refresh", "--dir", out}
					if format == "json" {
						args = append([]string{"--json"}, args...)
					}
					if variant == "dry-run" {
						args = append(args, "--dry-run")
					}
					stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
					code := Execute(args, stdout, stderr)
					if variant == "dry-run" {
						if code != 0 || calls != 0 || appends != 0 {
							t.Errorf("dry-run exit/calls/appends = %d/%d/%d", code, calls, appends)
						}
						if _, err := os.Stat(filepath.Join(out, "jobs")); !os.IsNotExist(err) {
							t.Errorf("dry-run created lease directory: %v", err)
						}
					} else if failAppend {
						wantCalls := 1
						if site == "unknown-source" {
							wantCalls = 0
						}
						if code != 1 || calls != wantCalls || appends != 1 {
							t.Errorf("exit/calls/appends = %d/%d/%d, want 1/%d/1; stdout=%s stderr=%s", code, calls, appends, wantCalls, stdout, stderr)
						}
						if format == "json" {
							var envelope struct {
								OK    bool                           `json:"ok"`
								Data  json.RawMessage                `json:"data"`
								Error struct{ Code, Message string } `json:"error"`
							}
							if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil || envelope.OK || envelope.Error.Code != "search_refresh_write_failed" || !strings.Contains(envelope.Error.Message, injected.Error()) || len(envelope.Data) != 0 {
								t.Errorf("lost write-error envelope or published delta: %s (%v)", stdout, err)
							}
							if stderr.Len() != 0 {
								t.Errorf("JSON error wrote stderr: %s", stderr)
							}
						} else if stdout.Len() != 0 || strings.TrimSpace(stderr.String()) != injected.Error() {
							t.Errorf("lost text persistence diagnostic or published summary: stdout=%s stderr=%s", stdout, stderr)
						}
					} else {
						wantResults, wantFailures, wantCause := 2, 1, "synthetic "+site+" failure"
						switch site {
						case "connector":
							wantCause = "source HTTP status 404"
						case "unknown-source":
							wantResults, wantFailures, wantCause = 4, 2, "unknown source"
						case "result":
							wantResults, wantFailures = 4, 0
						}
						if code != 0 || calls != 2 || appends != wantResults+wantFailures {
							t.Fatalf("successful append exit/calls/appends = %d/%d/%d; stdout=%s stderr=%s", code, calls, appends, stdout, stderr)
						}
						papers, err := loadSearchBatchResults(filepath.Join(out, "results.jsonl"))
						if err != nil || len(papers) != wantResults {
							t.Fatalf("read refreshed results: count=%d err=%v", len(papers), err)
						}
						for _, paper := range papers {
							if paper.Identifiers.DOI != "10.0000/a" && paper.Identifiers.DOI != "10.0000/b" {
								t.Errorf("unexpected refreshed paper: %+v", paper)
							}
						}
						ledger, err := os.ReadFile(filepath.Join(out, "failures.jsonl"))
						if err != nil {
							t.Fatal(err)
						}
						lines := bytes.Split(bytes.TrimSpace(ledger), []byte("\n"))
						if wantFailures == 0 {
							if len(ledger) != 0 {
								t.Errorf("unexpected failures: %s", ledger)
							}
						} else {
							if len(lines) != wantFailures {
								t.Errorf("failure ledger count=%d, want %d", len(lines), wantFailures)
							}
							for _, line := range lines {
								var failure searchBatchFailure
								if err := json.Unmarshal(line, &failure); err != nil || !strings.Contains(failure.Error, wantCause) {
									t.Errorf("lost ordinary failure: %s (%v)", line, err)
								}
							}
						}
						var manifest searchBatchManifest
						if err := readJSONFile(filepath.Join(out, "manifest.json"), &manifest); err != nil {
							t.Fatal(err)
						}
						if manifest.Results != wantResults || manifest.Failures != wantFailures || manifest.Deduped != 2 {
							t.Errorf("incorrect refreshed counts: %+v", manifest)
						}
						if _, err := os.Stat(filepath.Join(out, searchBatchStateFile)); !os.IsNotExist(err) {
							t.Errorf("successful refresh retained checkpoint: %v", err)
						}
						return
					}
					if after := snapshotRefreshArtifacts(t, out); !reflect.DeepEqual(after, before) {
						t.Error("refresh changed prior artifact bytes/modes or published staged outputs")
					}
				})
			}
		}
	}
}

func snapshotRefreshArtifacts(t *testing.T, root string) map[string]fileSnapshot {
	t.Helper()
	files := map[string]fileSnapshot{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Refresh may acquire a lease, but does not execute a durable batch job.
		if entry.IsDir() && entry.Name() == "jobs" {
			return filepath.SkipDir
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		var data []byte
		if !entry.IsDir() {
			data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		files[strings.TrimPrefix(path, root)] = fileSnapshot{data: data, mode: info.Mode()}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
