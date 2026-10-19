package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/search"
)

func newJobsTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	port := os.Getenv("AUTOWORK_PORT_BASE")
	if port == "" {
		port = "0"
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func TestSearchWritersRespectJobLease(t *testing.T) {
	for _, command := range []string{"resume", "refresh"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			failures := filepath.Join(dir, "failures.jsonl")
			if err := os.WriteFile(failures, []byte(`{"source":"unknown","query":"alpha","error":"fixture"}`+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := writeJSONFile(filepath.Join(dir, "manifest.json"), searchBatchManifest{SchemaVersion: "1", Queries: []string{"alpha"}, Sources: []string{"unknown"}, Limit: 2}); err != nil {
				t.Fatal(err)
			}
			store := jobs.NewStore(dir)
			job, err := store.Create("fixture", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.Run(context.Background(), job.ID, func(_ context.Context, _ *jobs.Run) error {
				args := []string{"search", "resume", "--failures", failures, "--out", dir}
				if command == "refresh" {
					args = []string{"search", "refresh", "--dir", dir}
				}
				var out, stderr bytes.Buffer
				if code := Execute(args, &out, &stderr); code == 0 || !strings.Contains(stderr.String(), jobs.ErrBusy.Error()) {
					t.Fatalf("writer bypassed live job: code=%d out=%s err=%s", code, &out, &stderr)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// A SIGKILL must release the live lease and preserve both pair checkpoints and
// resolved inputs; jobs resume must not need the original queries file.
func TestJobsResumeKilledSearch(t *testing.T) {
	if dir := os.Getenv("RFORGE_SEARCH_JOB_WORKER"); dir != "" {
		code := Execute([]string{"search", "batch", "--queries", os.Getenv("RFORGE_SEARCH_JOB_QUERIES"), "--sources", "openalex", "--limit", "2", "--stats", "--out", dir}, os.Stdout, os.Stderr)
		if code != 0 {
			t.Fatalf("worker exit %d", code)
		}
		return
	}
	var block atomic.Bool
	block.Store(true)
	started := make(chan struct{}, 1)
	server := newJobsTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("search")
		if query == "beta" && block.CompareAndSwap(true, false) {
			started <- struct{}{}
			<-r.Context().Done()
			return
		}
		if r.URL.Query().Get("per-page") != "2" {
			t.Errorf("lost saved limit: %s", r.URL.RawQuery)
		}
		fmt.Fprintf(w, `{"results":[{"id":"https://openalex.org/W-%s","doi":"https://doi.org/10.0000/%s","title":"%s","publication_year":2026}]}`, query, query, query)
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	dir := t.TempDir()
	queries := filepath.Join(t.TempDir(), "queries.txt")
	if err := os.WriteFile(queries, []byte("alpha\nbeta\ngamma\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestJobsResumeKilledSearch$")
	cmd.Env = append(os.Environ(), "RFORGE_SEARCH_JOB_WORKER="+dir, "RFORGE_SEARCH_JOB_QUERIES="+queries)
	var workerOutput bytes.Buffer
	cmd.Stdout, cmd.Stderr = &workerOutput, &workerOutput
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatalf("worker never reached second pair: %v", ctx.Err())
	}
	runCLI := func(args ...string) []byte {
		t.Helper()
		var out, stderr bytes.Buffer
		if code := Execute(append([]string{"--json"}, args...), &out, &stderr); code != 0 {
			t.Fatalf("%v: code=%d out=%s err=%s", args, code, &out, &stderr)
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Data
	}
	var listing struct {
		Jobs []jobs.Job `json:"jobs"`
	}
	if err := json.Unmarshal(runCLI("jobs", "list", "--dir", dir), &listing); err != nil {
		t.Fatal(err)
	}
	if len(listing.Jobs) != 1 || listing.Jobs[0].Status != jobs.Running || listing.Jobs[0].Progress.Completed != 1 {
		t.Fatalf("live jobs: %+v", listing)
	}
	id := listing.Jobs[0].ID
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if err := os.Remove(queries); err != nil {
		t.Fatal(err)
	}
	var status struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(runCLI("jobs", "status", id, "--dir", dir), &status); err != nil {
		t.Fatal(err)
	}
	if status.Job.Status != jobs.Interrupted || status.Job.StopReason != "interrupted" {
		t.Fatalf("killed job: %+v", status.Job)
	}
	runCLI("jobs", "resume", id, "--dir", dir)
	if err := json.Unmarshal(runCLI("jobs", "status", id, "--dir", dir), &status); err != nil {
		t.Fatal(err)
	}
	if status.Job.Status != jobs.Completed || status.Job.Progress.Completed != 3 || status.Job.Progress.Total != 3 || status.Job.StopReason != "completed" {
		t.Fatalf("resumed job: %+v", status.Job)
	}
	if _, err := os.Stat(filepath.Join(dir, "search-stats.txt")); err != nil {
		t.Fatal("lost saved stats option:", err)
	}
	control := t.TempDir()
	runCLI("search", "batch", "--query", "alpha", "--query", "beta", "--query", "gamma", "--sources", "openalex", "--limit", "2", "--out", control)
	finalState, controlState := readBatchState(t, dir), readBatchState(t, control)
	if !reflect.DeepEqual(finalState.Completed, controlState.Completed) || !reflect.DeepEqual(finalState.Coverage, controlState.Coverage) || finalState.StopReason != controlState.StopReason {
		t.Fatalf("resume differs from uninterrupted search: %+v / %+v", finalState, controlState)
	}
	if finalState.RequestsUsed != 4 || controlState.RequestsUsed != 3 {
		t.Fatalf("interrupted request accounting: %d / %d", finalState.RequestsUsed, controlState.RequestsUsed)
	}
	finalResults, err := os.ReadFile(filepath.Join(dir, "results-deduped.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	controlResults, err := os.ReadFile(filepath.Join(control, "results-deduped.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(finalResults, controlResults) {
		t.Fatalf("resumed records differ: %s / %s", finalResults, controlResults)
	}
	// A host may exit after registering a fresh job but before it replaces an
	// older batch checkpoint. Resume must use the new inputs, not reject it as
	// superseded or accidentally retain the old worklist/results.
	queued, err := jobs.NewStore(dir).Create(search.BatchJobKind, search.BatchJobInput{Queries: []string{"delta"}, Sources: []string{"openalex"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	runCLI("jobs", "resume", queued.ID, "--dir", dir)
	fresh := readBatchState(t, dir)
	if fresh.JobID != queued.ID || len(fresh.Completed) != 1 || fresh.Completed[0].Query != "delta" || fresh.RequestsUsed != 1 {
		t.Fatalf("new job inherited the old checkpoint: %+v", fresh)
	}
}
