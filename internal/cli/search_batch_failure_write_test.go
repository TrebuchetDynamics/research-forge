package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func TestSearchBatchFailureLedgerWrite(t *testing.T) {
	// These cases mutate an append seam and the default transport, never in parallel.
	for _, site := range []string{"configuration", "runtime"} {
		for _, resume := range []bool{false, true} {
			for _, format := range []string{"text", "json"} {
				name := site + "/fresh/" + format
				if resume {
					name = site + "/resume/" + format
				}
				t.Run(name, func(t *testing.T) {
					t.Setenv("RFORGE_LENS_TOKEN", "")
					t.Setenv("RFORGE_OPENALEX_URL", "https://ledger.invalid")
					t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
					calls := 0
					transport := http.DefaultTransport
					t.Cleanup(func() { http.DefaultTransport = transport })
					http.DefaultTransport = rankFixtureTransport(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.URL.Host != "ledger.invalid" {
							t.Fatalf("unexpected connector request: %s", r.URL)
						}
						status, body := http.StatusNotFound, "synthetic source failure"
						if r.URL.Query().Get("search") == "saved" {
							status = http.StatusOK
							body = `{"results":[{"id":"https://openalex.org/W1","doi":"https://doi.org/10.1000/saved","title":"Saved paper"}]}`
						}
						return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					})
					out, project := t.TempDir(), t.TempDir()
					var prior searchBatchState
					var priorManifest searchBatchManifest
					if resume {
						code := Execute([]string{"--project", project, "search", "batch", "--query", "saved", "--query", "failed", "--sources", "openalex", "--out", out, "--continue-on-error"}, new(bytes.Buffer), new(bytes.Buffer))
						if code != 0 {
							t.Fatalf("seed batch failed: %d", code)
						}
						prior = readBatchState(t, out)
						if len(prior.Completed) != 1 || len(prior.Failures) != 1 || prior.RequestsUsed != 2 {
							t.Fatalf("invalid saved batch: %+v", prior)
						}
						if err := readJSONFile(filepath.Join(out, "manifest.json"), &priorManifest); err != nil {
							t.Fatal(err)
						}
					}
					// Include raw, results, failures, receipts and the project library;
					// checkpoint/manifest accounting is checked separately below.
					snapshot := func(root string) map[string]string {
						t.Helper()
						files := map[string]string{}
						err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
							if err != nil {
								return err
							}
							if entry.IsDir() {
								if entry.Name() == "jobs" {
									return filepath.SkipDir
								}
								return nil
							}
							if entry.Name() == searchBatchStateFile || entry.Name() == "manifest.json" {
								return nil
							}
							data, err := os.ReadFile(path)
							files[strings.TrimPrefix(path, root)] = string(data)
							return err
						})
						if err != nil {
							t.Fatal(err)
						}
						return files
					}
					beforeOut, beforeProject := snapshot(out), snapshot(project)
					calls, appends := 0, 0
					injected := errors.New("synthetic failure-ledger append error")
					appendFailure := appendSearchBatchFailure
					t.Cleanup(func() { appendSearchBatchFailure = appendFailure })
					appendSearchBatchFailure = func(w io.Writer, value any) error {
						failure := value.(searchBatchFailure)
						if failure.Skipped != (site == "configuration") {
							return appendFailure(w, value)
						}
						appends++
						return injected
					}
					source, wantCalls, wantKind := "openalex", 1, "search_batch_failed"
					if site == "configuration" {
						source, wantCalls, wantKind = "lens,openalex", 0, "search_batch_write_failed"
					}
					args := []string{"--project", project}
					if format == "json" {
						args = append(args, "--json")
					}
					args = append(args, "search", "batch", "--query", "failed", "--query", "pending", "--sources", source, "--out", out, "--continue-on-error")
					if resume {
						args = append(args, "--query", "saved", "--resume")
					}
					stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
					code := Execute(args, stdout, stderr)
					if code != 1 || calls != wantCalls || appends != 1 {
						t.Errorf("exit/calls/appends = %d/%d/%d, want 1/%d/1; stdout=%s stderr=%s", code, calls, appends, wantCalls, stdout, stderr)
					}
					assertEnvelope := func(data []byte) {
						t.Helper()
						var envelope struct {
							OK    bool                           `json:"ok"`
							Error struct{ Code, Message string } `json:"error"`
						}
						if err := json.Unmarshal(data, &envelope); err != nil || envelope.OK || envelope.Error.Code != wantKind || !strings.Contains(envelope.Error.Message, injected.Error()) {
							t.Errorf("lost persistence cause/error envelope: %s (%v)", data, err)
						}
					}
					if format == "json" {
						assertEnvelope(stdout.Bytes())
						if stderr.Len() != 0 {
							t.Errorf("JSON error wrote stderr: %s", stderr)
						}
					} else if stdout.Len() != 0 || !strings.Contains(stderr.String(), injected.Error()) {
						t.Errorf("lost text persistence diagnostic: stdout=%s stderr=%s", stdout, stderr)
					}
					all, err := jobs.NewStore(out).List()
					if err != nil {
						t.Fatal(err)
					}
					failed := 0
					for _, job := range all {
						if job.Status == jobs.Failed {
							failed++
							if job.Progress.Completed != len(prior.Completed) || job.StopReason != "error_threshold" {
								t.Errorf("failed job reports rolled-back progress: %+v", job)
							}
							assertEnvelope(job.Result)
						}
					}
					if failed != 1 {
						t.Errorf("failed jobs = %d, want 1", failed)
					}
					if !reflect.DeepEqual(snapshot(out), beforeOut) || !reflect.DeepEqual(snapshot(project), beforeProject) {
						t.Error("failed append changed result/failure/raw/receipt/project artifacts")
					}
					if !resume {
						for _, name := range []string{searchBatchStateFile, "manifest.json"} {
							if _, err := os.Stat(filepath.Join(out, name)); !os.IsNotExist(err) {
								t.Errorf("fresh failure published %s: %v", name, err)
							}
						}
						return
					}
					after := readBatchState(t, out)
					expected := prior
					if wantCalls > 0 {
						expected.RequestsUsed += wantCalls
						expected.StopReason = sources.StopErrorThreshold
						expected.UpdatedAt = after.UpdatedAt
					}
					if !reflect.DeepEqual(after, expected) {
						t.Errorf("resume lost prior checkpoint or consumed usage: got %+v, want %+v", after, expected)
					}
					var manifest searchBatchManifest
					if err := readJSONFile(filepath.Join(out, "manifest.json"), &manifest); err != nil {
						t.Fatal(err)
					}
					priorManifest.BatchProgress = expected.BatchProgress
					if !reflect.DeepEqual(manifest, priorManifest) {
						t.Errorf("resume manifest changed beyond consumed usage: got %+v, want %+v", manifest, priorManifest)
					}
				})
			}
		}
	}
}
