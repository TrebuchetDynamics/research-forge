package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// Missing budget checks or counting duplicate records as yield changes these outputs.
func TestSearchBatchBudgetStops(t *testing.T) {
	for _, tc := range []struct {
		name, reason              string
		queries, requests, unique int
		flags                     []string
		fail                      bool
		failFirst                 bool
	}{
		{name: "completed", reason: "completed", queries: 1, requests: 1, unique: 1},
		{name: "request cap", reason: "budget_exhausted", queries: 2, requests: 1, unique: 1, flags: []string{"--max-requests", "1"}},
		{name: "query cap", reason: "budget_exhausted", queries: 4, requests: 3, unique: 3, flags: []string{"--depth", "quick"}},
		{name: "duplicate saturation", reason: "coverage_saturated", queries: 6, requests: 3, unique: 1},
		{name: "error then saturation", reason: "coverage_saturated", queries: 6, requests: 4, unique: 1, flags: []string{"--continue-on-error"}, failFirst: true},
		{name: "error threshold", reason: "error_threshold", queries: 6, requests: 5, flags: []string{"--continue-on-error"}, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.fail || (tc.failFirst && r.URL.Query().Get("search") == "a") {
					http.Error(w, "missing", http.StatusNotFound)
					return
				}
				id := "shared"
				if tc.name == "query cap" {
					id = r.URL.Query().Get("search")
				}
				_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W1","doi":"https://doi.org/10.1000/` + id + `","title":"Shared"}]}`))
			}))
			defer server.Close()
			t.Setenv("RFORGE_OPENALEX_URL", server.URL)
			out := t.TempDir()
			args := []string{"--sources", "openalex", "--out", out}
			for i := 0; i < tc.queries; i++ {
				args = append(args, "--query", string(rune('a'+i)))
			}
			args = append(args, tc.flags...)
			code, _, stderr := runSearchBatch(t, args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			for _, file := range []string{"manifest.json", "batch-state.json"} {
				data, err := os.ReadFile(filepath.Join(out, file))
				if err != nil {
					t.Fatal(err)
				}
				var got struct {
					StopReason   string `json:"stop_reason"`
					RequestsUsed int    `json:"requests_used"`
					Coverage     struct {
						UniqueRecords int `json:"unique_records"`
					} `json:"coverage"`
				}
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				if got.StopReason != tc.reason || got.RequestsUsed != tc.requests || got.Coverage.UniqueRecords != tc.unique {
					t.Fatalf("%s: got %+v, want %s/%d/%d", file, got, tc.reason, tc.requests, tc.unique)
				}
			}
		})
	}
}

func TestSearchBatchResumePreservesRequestBudget(t *testing.T) {
	server := openAlexBatchServer(nil, "Budget")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	args := []string{"--query", "alpha", "--query", "beta", "--sources", "openalex", "--out", out, "--max-requests", "1"}
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("initial: %d %s", code, stderr)
	}
	// Omitting the cap on resume must not silently replenish it.
	args = args[:len(args)-2]
	args = append(args, "--resume")
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("resume: %d %s", code, stderr)
	}
	state := readBatchState(t, out)
	if len(state.Completed) != 1 {
		t.Fatalf("budget reset on resume: %+v", state)
	}
	args = append(args, "--max-requests", "2")
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("extended resume: %d %s", code, stderr)
	}
	data, err := os.ReadFile(filepath.Join(out, "batch-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		RequestsUsed int    `json:"requests_used"`
		StopReason   string `json:"stop_reason"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.RequestsUsed != 2 || got.StopReason != "completed" {
		t.Fatalf("resume accounting: %s", data)
	}
}

func TestSearchBatchFailedResumeKeepsUsageAndFailures(t *testing.T) {
	for _, mode := range []string{"exhausted", "fail-fast", "legacy"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "missing", http.StatusNotFound)
			}))
			defer server.Close()
			t.Setenv("RFORGE_OPENALEX_URL", server.URL)
			out := t.TempDir()
			args := []string{"--query", "q", "--sources", "openalex", "--out", out, "--max-requests", "1", "--continue-on-error"}
			if code, _, stderr := runSearchBatch(t, args...); code != 0 {
				t.Fatalf("initial: %d %s", code, stderr)
			}
			if mode == "legacy" {
				if code := Execute([]string{"search", "resume", "--failures", filepath.Join(out, "failures.jsonl"), "--out", out}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
					t.Fatalf("legacy: %d", code)
				}
			} else {
				resume := []string{"--query", "q", "--sources", "openalex", "--out", out, "--resume"}
				if mode == "fail-fast" {
					resume = append(resume, "--max-requests", "2")
				}
				code, _, stderr := runSearchBatch(t, resume...)
				if mode == "fail-fast" && code == 0 {
					t.Fatal("expected fail-fast error")
				}
				if mode != "fail-fast" && code != 0 {
					t.Fatalf("resume: %d %s", code, stderr)
				}
			}
			state := readBatchState(t, out)
			want := 1
			if mode == "fail-fast" {
				want = 2
			}
			if state.RequestsUsed != want || int(calls.Load()) != want || len(state.Failures) != 1 {
				t.Fatalf("lost usage/failure: %+v", state)
			}
			failures := readSearchBatchFailures(filepath.Join(out, "failures.jsonl"))
			if len(failures) != 1 {
				t.Fatalf("lost unattempted failure: %+v", failures)
			}
		})
	}
}

func TestSearchBatchResumeIgnoresUncheckpointedResultTail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("search")
		fmt.Fprintf(w, `{"results":[{"id":"https://openalex.org/W1","doi":"https://doi.org/10.1000/%s","title":"%s"}]}`, q, q)
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	args := []string{"--query", "alpha", "--query", "beta", "--sources", "openalex", "--out", out, "--max-requests", "1"}
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("initial: %d %s", code, stderr)
	}
	// Simulate a kill after append but before the pair checkpoint. Include a torn final line.
	file, err := os.OpenFile(filepath.Join(out, "results.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	record := library.PaperRecord{Title: "beta", Identifiers: library.Identifiers{DOI: "10.1000/beta"}}
	if err := writeJSONLine(file, record); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"torn":`); err != nil {
		t.Fatal(err)
	}
	file.Close()
	args = append(args[:len(args)-2], "--resume", "--max-requests", "2")
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("resume: %d %s", code, stderr)
	}
	state := readBatchState(t, out)
	if len(state.Coverage.Queries) != 2 || state.Coverage.Queries[1].NewUnique != 1 {
		t.Fatalf("orphan tail corrupted marginal yield: %+v", state.Coverage)
	}
	papers, err := loadSearchBatchResults(filepath.Join(out, "results.jsonl"))
	if err != nil || len(papers) != 2 {
		t.Fatalf("results=%d err=%v", len(papers), err)
	}
}

func TestSearchBatchResumeCoverageHistoryGap(t *testing.T) {
	server := openAlexBatchServer(nil, "Legacy")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	args := []string{"--sources", "openalex", "--out", out}
	for _, q := range []string{"a", "b", "c", "d", "e", "f"} {
		args = append(args, "--query", q)
	}
	initial := append(append([]string{}, args...), "--max-requests", "1")
	if code, _, stderr := runSearchBatch(t, initial...); code != 0 {
		t.Fatalf("initial: %d %s", code, stderr)
	}
	state := readBatchState(t, out)
	// Legacy checkpoints have completed pairs but no per-query coverage history.
	state.Coverage.Queries = nil
	state.Coverage.ConsecutiveLowYield = 0
	if err := writeSearchBatchState(out, state); err != nil {
		t.Fatal(err)
	}
	args = append(args, "--resume", "--max-requests", "10")
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("resume: %d %s", code, stderr)
	}
	state = readBatchState(t, out)
	if state.StopReason != "coverage_saturated" || state.RequestsUsed != 4 {
		t.Fatalf("history gap disabled saturation: %+v", state)
	}
}

func TestSearchBatchResumeReconcilesChangedQueries(t *testing.T) {
	server := openAlexBatchServer(nil, "Changed")
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()
	args := []string{"--query", "a", "--query", "b", "--query", "c", "--query", "d", "--sources", "openalex", "--out", out}
	if code, _, stderr := runSearchBatch(t, args...); code != 0 {
		t.Fatalf("initial: %d %s", code, stderr)
	}
	// Prior three low-yield queries saturated; a different worklist must get a fresh coverage signal, not a new request budget.
	if code, _, stderr := runSearchBatch(t, "--query", "new", "--sources", "openalex", "--out", out, "--resume"); code != 0 {
		t.Fatalf("changed resume: %d %s", code, stderr)
	}
	state := readBatchState(t, out)
	if state.RequestsUsed != 4 || !state.HasCompleted("openalex", "new") {
		t.Fatalf("stale saturation blocked changed worklist: %+v", state)
	}
}
