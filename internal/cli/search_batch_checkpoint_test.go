package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

// openAlexBatchServer serves one record per request and counts requests so
// tests can prove which (query, source) pairs actually ran.
func openAlexBatchServer(counter *int32, title string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if counter != nil {
			atomic.AddInt32(counter, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-TXN","doi":"https://doi.org/10.0000/` + title + `","title":"` + title + ` ` + r.URL.Query().Get("search") + `","publication_year":2026}]}`))
	}))
}

func runSearchBatch(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute(append([]string{"search", "batch"}, args...), stdout, stderr)
	return code, stdout.String(), stderr.String()
}

func readBatchState(t *testing.T, dir string) searchBatchState {
	t.Helper()
	state, exists, err := readSearchBatchState(dir)
	if err != nil {
		t.Fatalf("read batch state: %v", err)
	}
	if !exists {
		t.Fatalf("expected %s to exist", filepath.Join(dir, searchBatchStateFile))
	}
	return state
}

// TestExecuteSearchBatchWritesCheckpoint proves a completed sweep leaves an
// incremental checkpoint describing every completed pair, so a later
// interruption or resume has a durable record to build on.
func TestExecuteSearchBatchWritesCheckpoint(t *testing.T) {
	server := openAlexBatchServer(nil, "Checkpoint")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	code, stdout, stderr := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}
	assertEngineReceipt(t, out, "results-deduped.jsonl")
	state := readBatchState(t, out)
	if len(state.Completed) != 2 {
		t.Fatalf("checkpoint completed pairs = %d, want 2 (%+v)", len(state.Completed), state.Completed)
	}
	for _, pair := range state.Completed {
		if pair.Records != 1 || pair.RawFile == "" {
			t.Fatalf("checkpoint pair missing records/rawFile: %+v", pair)
		}
		if _, err := os.Stat(filepath.Join(out, pair.RawFile)); err != nil {
			t.Fatalf("checkpoint raw file %s missing: %v", pair.RawFile, err)
		}
	}
	if state.Interrupted {
		t.Fatalf("completed sweep must not be marked interrupted: %+v", state)
	}
	var manifest searchBatchManifest
	manifestData, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Pending != 0 || manifest.Interrupted {
		t.Fatalf("manifest pending/interrupted = %d/%v, want 0/false", manifest.Pending, manifest.Interrupted)
	}
}

// TestExecuteSearchBatchResumeSkipsCompletedPairs is the core resume promise:
// re-running the same sweep with --resume must not repeat any network call
// for a pair the checkpoint already recorded as complete.
func TestExecuteSearchBatchResumeSkipsCompletedPairs(t *testing.T) {
	var requests int32
	server := openAlexBatchServer(&requests, "Resume")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	code, _, stderr := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error")
	if code != 0 {
		t.Fatalf("initial sweep failed: %d %s", code, stderr)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("initial sweep requests = %d, want 2", got)
	}
	resultsBefore, err := os.ReadFile(filepath.Join(out, "results.jsonl"))
	if err != nil {
		t.Fatalf("read results: %v", err)
	}

	code, stdout, stderr := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error", "--resume")
	if code != 0 {
		t.Fatalf("resume exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("resume repeated completed pairs: requests = %d, want 2", got)
	}
	if got := provenance.DirectoryLevel(out); got != provenance.EngineVerified {
		t.Fatalf("successful resume provenance=%s", got)
	}
	resultsAfter, err := os.ReadFile(filepath.Join(out, "results.jsonl"))
	if err != nil {
		t.Fatalf("read results after resume: %v", err)
	}
	if !bytes.Equal(resultsBefore, resultsAfter) {
		t.Fatalf("resume changed accumulated results:\nbefore=%s\nafter=%s", resultsBefore, resultsAfter)
	}
	if !strings.Contains(stdout, "resumed from checkpoint") {
		t.Fatalf("resume stdout missing checkpoint summary: %s", stdout)
	}
	var manifest searchBatchManifest
	manifestData, _ := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if !manifest.Resumed || manifest.ResumedPairs != 2 {
		t.Fatalf("manifest resumed/pairs = %v/%d, want true/2 (%+v)", manifest.Resumed, manifest.ResumedPairs, manifest)
	}
}

// TestExecuteSearchBatchResumeRetriesFailedPairs proves --resume re-runs pairs
// that previously failed (the "278 source failures" case) and clears their
// failure record once they succeed.
func TestExecuteSearchBatchResumeRetriesFailedPairs(t *testing.T) {
	var requests int32
	var failFirstQuery int32 = 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if atomic.LoadInt32(&failFirstQuery) == 1 && strings.Contains(r.URL.Query().Get("search"), "beta") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-R","doi":"https://doi.org/10.0000/retry","title":"Retry fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	code, _, _ := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error")
	if code != 0 {
		t.Fatalf("initial sweep failed: %d", code)
	}
	state := readBatchState(t, out)
	if len(state.Completed) != 1 || len(state.Failures) != 1 {
		t.Fatalf("after failed sweep: completed=%d failures=%d, want 1/1", len(state.Completed), len(state.Failures))
	}
	if state.Failures[0].Query != "beta topic" {
		t.Fatalf("failure recorded for wrong query: %+v", state.Failures[0])
	}

	atomic.StoreInt32(&failFirstQuery, 0)
	code, stdout, stderr := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error", "--resume")
	if code != 0 {
		t.Fatalf("resume exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}
	if got := atomic.LoadInt32(&requests); got != 3 {
		t.Fatalf("resume requests = %d, want 3 (2 initial + 1 retry)", got)
	}
	state = readBatchState(t, out)
	if len(state.Completed) != 2 || len(state.Failures) != 0 {
		t.Fatalf("after resume: completed=%d failures=%d, want 2/0", len(state.Completed), len(state.Failures))
	}
	failures, err := os.ReadFile(filepath.Join(out, "failures.jsonl"))
	if err != nil {
		t.Fatalf("read failures.jsonl: %v", err)
	}
	if strings.TrimSpace(string(failures)) != "" {
		t.Fatalf("failures.jsonl after successful retry = %q, want empty", failures)
	}
}

// TestExecuteSearchBatchResumeAfterCrash reconstructs the user's real failure
// mode: a sweep killed by an external timeout guard leaves a partial
// checkpoint in the topic dir (never a /tmp orphan), and --resume completes
// the missing pairs and regenerates aggregates from prior + new results.
func TestExecuteSearchBatchResumeAfterCrash(t *testing.T) {
	var requests int32
	server := openAlexBatchServer(&requests, "Crash")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	// Simulate a killed run: query 1 completed and checkpointed, query 2
	// never attempted, aggregates never written.
	if err := os.MkdirAll(filepath.Join(out, "raw"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "raw", "search-openalex-001-alpha-topic.txt"), []byte("10.0000/alpha\tAlpha Topic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "results.jsonl"), []byte("{\"Title\":\"Alpha Topic\",\"Identifiers\":{\"DOI\":\"10.0000/alpha\",\"OpenAlexID\":\"W-A\"},\"Year\":2026,\"SourceRefs\":[{\"Source\":\"openalex\"}]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	killedState := searchBatchState{
		SchemaVersion: "1",
		CreatedAt:     "2026-08-20T21:00:00Z",
		UpdatedAt:     "2026-08-20T21:20:00Z",
		Queries:       []string{"alpha topic", "beta topic"},
		Sources:       []string{"openalex"},
		Limit:         25,
		Completed:     []searchBatchStatePair{{Source: "openalex", Query: "alpha topic", RawFile: "raw/search-openalex-001-alpha-topic.txt", Records: 1}},
		Interrupted:   true,
	}
	if err := writeSearchBatchState(out, killedState); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error", "--resume")
	if code != 0 {
		t.Fatalf("crash resume exit code = %d, stderr = %s, stdout = %s", code, stderr, stdout)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("crash resume requests = %d, want 1 (only the missing pair)", got)
	}
	state := readBatchState(t, out)
	if len(state.Completed) != 2 || state.Interrupted {
		t.Fatalf("after crash resume: completed=%d interrupted=%v, want 2/false", len(state.Completed), state.Interrupted)
	}
	results, err := os.ReadFile(filepath.Join(out, "results.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(results), "10.0000/alpha") || !strings.Contains(string(results), "10.0000/crash") {
		t.Fatalf("results.jsonl after crash resume must hold prior + new records; got %s", results)
	}
	var manifest searchBatchManifest
	manifestData, _ := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Results != 2 || manifest.Pending != 0 {
		t.Fatalf("manifest results/pending = %d/%d, want 2/0", manifest.Results, manifest.Pending)
	}
}

// TestExecuteSearchBatchResumeWithoutStateRunsFresh proves --resume on a
// topic dir with no checkpoint is simply a fresh checkpointed sweep.
func TestExecuteSearchBatchResumeWithoutStateRunsFresh(t *testing.T) {
	var requests int32
	server := openAlexBatchServer(&requests, "Fresh")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	code, _, stderr := runSearchBatch(t, "--query", "alpha topic", "--sources", "openalex", "--out", out, "--continue-on-error", "--resume")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
	readBatchState(t, out)
}

// TestExecuteSearchBatchFailedRunLeavesNoCheckpoint preserves the transactional
// contract: without --continue-on-error a failing pair must leave the topic
// dir untouched, including the incremental checkpoint.
func TestExecuteSearchBatchFailedRunLeavesNoCheckpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "results.jsonl"), []byte("prior\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, _ := runSearchBatch(t, "--query", "alpha topic", "--sources", "openalex", "--out", out)
	if code == 0 {
		t.Fatalf("expected non-zero exit for failing pair")
	}
	if _, err := os.Stat(filepath.Join(out, searchBatchStateFile)); !os.IsNotExist(err) {
		t.Fatalf("failed run left a checkpoint behind: %v", err)
	}
	results, err := os.ReadFile(filepath.Join(out, "results.jsonl"))
	if err != nil || string(results) != "prior\n" {
		t.Fatalf("failed run must restore prior results.jsonl; got %q err=%v", results, err)
	}
	if _, err := os.Stat(filepath.Join(out, "raw")); !os.IsNotExist(err) {
		t.Fatalf("failed run left a raw dir: %v", err)
	}
}

// TestExecuteSearchStatsReportsCheckpoint proves the verification-gate command
// surfaces checkpoint progress and recommends the resume command.
func TestExecuteSearchStatsReportsCheckpoint(t *testing.T) {
	server := openAlexBatchServer(nil, "Stats")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	code, _, stderr := runSearchBatch(t, "--query", "alpha topic", "--query", "beta topic", "--sources", "openalex", "--out", out, "--continue-on-error")
	if code != 0 {
		t.Fatalf("sweep failed: %d %s", code, stderr)
	}

	// Simulate one never-run query by truncating the checkpoint's completed list.
	state := readBatchState(t, out)
	state.Completed = state.Completed[:1]
	if err := writeSearchBatchState(out, state); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	stderr2 := new(bytes.Buffer)
	if code := Execute([]string{"search", "stats", "--dir", out}, stdout, stderr2); code != 0 {
		t.Fatalf("stats exit code = %d: %s", code, stderr2.String())
	}
	if !strings.Contains(stdout.String(), "Batch checkpoint: 1 completed, 0 failed, 1 pending pair(s)") {
		t.Fatalf("stats stdout missing checkpoint summary: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "resume with: rforge search batch --resume") {
		t.Fatalf("stats stdout missing resume hint: %s", stdout.String())
	}

	jsonStdout := new(bytes.Buffer)
	if code := Execute([]string{"--json", "search", "stats", "--dir", out}, jsonStdout, new(bytes.Buffer)); code != 0 {
		t.Fatalf("json stats exit code = %d", code)
	}
	var payload struct {
		Data struct {
			BatchState *searchBatchCheckpointSummaryResult `json:"batchState"`
		} `json:"data"`
	}
	if err := json.Unmarshal(jsonStdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse json stats: %v", err)
	}
	if payload.Data.BatchState == nil || payload.Data.BatchState.Pending != 1 || !payload.Data.BatchState.ResumeStream {
		t.Fatalf("json stats batchState = %+v, want pending=1 resumeRecommended=true", payload.Data.BatchState)
	}
}
