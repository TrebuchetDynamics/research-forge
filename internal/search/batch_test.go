package search

import (
	"context"
	"fmt"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

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
	_, err := RunBatch(ctx, &state, nil, nil, true, BatchHooks{
		Search:     func(context.Context, string, string) ([]library.PaperRecord, error) { cancel(); return nil, ctx.Err() },
		Checkpoint: func() error { return WriteBatchState(dir, state) },
	})
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
