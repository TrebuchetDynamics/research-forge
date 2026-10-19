package search

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// Existing job IDs restore the exact saved request, never adapter defaults or
// fields supplied for a different new search.
func TestRunBatchJobResumesSavedInputs(t *testing.T) {
	dir := t.TempDir()
	input := BatchJobInput{Queries: []string{"alpha"}, Sources: []string{"fixture"}, Limit: 2}
	job, err := jobs.NewStore(dir).Create(BatchJobKind, input)
	if err != nil {
		t.Fatal(err)
	}
	_, err = RunBatchJob(context.Background(), dir, job.ID, BatchJobInput{Depth: "comprehensive", Project: "other", FetchPDFs: true}, true, func(_ context.Context, run *jobs.Run, saved BatchJobInput, resume bool) error {
		if !reflect.DeepEqual(saved, input) || resume {
			t.Fatalf("saved request overridden: %+v resume=%v", saved, resume)
		}
		return run.Checkpoint(jobs.Progress{Completed: 1, Total: 1}, "completed")
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A registration that never acquired the writer lease cannot supersede the
// interrupted owner of the committed checkpoint.
func TestRunBatchJobQueuedRegistrationDoesNotSupersedeWriter(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(dir)
	input := BatchJobInput{Queries: []string{"alpha", "beta"}, Sources: []string{"fixture"}, Limit: 2}
	first, err := store.Create(BatchJobKind, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(BatchJobKind, input)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err = store.Run(ctx, first.ID, func(_ context.Context, run *jobs.Run) error {
		if _, err := store.Run(ctx, second.ID, nil); !errors.Is(err, jobs.ErrBusy) {
			t.Fatalf("second writer: %v", err)
		}
		if err := WriteBatchState(dir, BatchState{JobID: first.ID}); err != nil {
			return err
		}
		cancel()
		return run.Checkpoint(jobs.Progress{Completed: 1, Total: 2}, "interrupted")
	})
	if err != nil {
		t.Fatal(err)
	}
	final, err := RunBatchJob(context.Background(), dir, first.ID, BatchJobInput{}, true, func(_ context.Context, run *jobs.Run, saved BatchJobInput, resume bool) error {
		if !resume || !reflect.DeepEqual(saved, input) || run.Job().Progress.Completed != 1 {
			t.Fatalf("lost checkpoint owner: %+v, %+v, resume=%v", run.Job(), saved, resume)
		}
		return run.Checkpoint(jobs.Progress{Completed: 2, Total: 2}, "completed")
	})
	if err != nil || final.Status != jobs.Completed {
		t.Fatalf("queued registration prevented resume: %+v, %v", final, err)
	}
}

func TestRunBatchDepthSourceCaps(t *testing.T) {
	for _, tc := range []struct {
		depth                  string
		queries, sources, want int
	}{
		{"quick", 2, 3, 4}, {"standard", 2, 6, 10}, {"comprehensive", 2, 21, 40},
	} {
		t.Run(tc.depth, func(t *testing.T) {
			budget, err := DepthBudget(tc.depth)
			if err != nil {
				t.Fatal(err)
			}
			state := BatchState{BatchProgress: BatchProgress{Budget: budget}}
			for i := 0; i < tc.queries; i++ {
				state.Queries = append(state.Queries, fmt.Sprint(i))
			}
			for i := 0; i < tc.sources; i++ {
				state.Sources = append(state.Sources, fmt.Sprint(i))
			}
			_, err = RunBatch(context.Background(), &state, nil, nil, true, BatchHooks{
				Search: func(_ context.Context, s, q string) ([]library.PaperRecord, error) {
					return []library.PaperRecord{{Title: s + q}}, nil
				},
				Checkpoint: func() error { return nil },
				Completed:  func(s, q string, _ int, p []library.PaperRecord) (string, error) { return "raw/result.txt", nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			if state.RequestsUsed != tc.want || state.StopReason != sources.StopBudgetExhausted {
				t.Fatalf("got requests=%d stop=%s", state.RequestsUsed, state.StopReason)
			}
		})
	}
}

func TestRunBatchInterruptedAttemptConsumesBudget(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	budget, _ := DepthBudget("standard")
	state := BatchState{Queries: []string{"q"}, Sources: []string{"s"}, BatchProgress: BatchProgress{Budget: budget}}
	dir := t.TempDir()
	store := jobs.NewStore(dir)
	job, err := store.Create("search.batch", nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err = store.Run(ctx, job.ID, func(ctx context.Context, run *jobs.Run) error {
		_, err := RunBatch(ctx, &state, nil, nil, true, BatchHooks{
			Job:        run,
			Search:     func(context.Context, string, string) ([]library.PaperRecord, error) { cancel(); return nil, ctx.Err() },
			Checkpoint: func() error { return WriteBatchState(dir, state) },
		})
		return err
	})
	if job.Status != jobs.Interrupted || job.Progress.Total != 1 || job.Progress.Completed != 0 {
		t.Fatalf("job did not persist interrupted batch progress: %+v", job)
	}
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := ReadBatchState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if saved.StopReason != sources.StopInterrupted || saved.RequestsUsed != 1 || len(saved.Completed) != 0 {
		t.Fatalf("interrupted checkpoint: %+v", saved)
	}
	saved.Resume([]string{"q"}, []string{"s"}, 25)
	if saved.RequestsUsed != 1 {
		t.Fatalf("resume lost aborted attempt: %+v", saved)
	}
}
