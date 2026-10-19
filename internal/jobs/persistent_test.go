package jobs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestStoreListsJobsChronologically(t *testing.T) {
	store := NewStore(t.TempDir())
	var ids []string
	for _, stamp := range []string{"2026-01-01T00:00:00.11Z", "2026-01-01T00:00:00.1Z"} {
		job, err := store.Create("fixture", nil)
		if err != nil {
			t.Fatal(err)
		}
		job.CreatedAt, err = time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.write(job); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, job.ID)
	}
	all, err := store.List()
	if err != nil || len(all) != 2 || all[0].ID != ids[1] || all[1].ID != ids[0] {
		t.Fatalf("jobs not in creation order: %+v, %v", all, err)
	}
}

// Atomic checkpoints must remain readable to CLI/MCP status pollers, including
// while the worker is publishing its failed output-snapshot result.
func TestStoreStatusDuringCheckpoints(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	started, done := make(chan struct{}), make(chan error, 1)
	failure := errors.New("output snapshot failed")
	go func() {
		_, err := store.Run(context.Background(), job.ID, func(_ context.Context, run *Run) error {
			close(started)
			for i := 0; i < 1000; i++ {
				if err := run.Checkpoint(Progress{Completed: i, Total: 1000}, ""); err != nil {
					return err
				}
			}
			if err := run.SetResult([]byte(`{"ok":false,"error":{"code":"search_batch_output_snapshot_failed"}}`)); err != nil {
				return err
			}
			return failure
		})
		done <- err
	}()
	select {
	case <-started:
	case err := <-done:
		t.Fatalf("worker did not start: %v", err)
	}
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for range 1000 {
				status, err := store.Status(job.ID)
				if err != nil || status.ID != job.ID {
					t.Errorf("checkpoint hid the job: %+v; %v", status, err)
					return
				}
			}
		})
	}
	readers.Wait()
	if err := <-done; !errors.Is(err, failure) {
		t.Fatalf("worker error: %v", err)
	}
	status, err := store.Status(job.ID)
	if err != nil || status.Status != Failed || status.StopReason != "error_threshold" || len(status.Result) == 0 {
		t.Fatalf("failed job lost its status/result: %+v; %v", status, err)
	}
}

// Losing the lease or the persisted progress would repeat the first step after a kill.
func TestStoreKilledJobResumes(t *testing.T) {
	if dir := os.Getenv("RFORGE_JOB_TEST_DIR"); dir != "" {
		store := NewStore(dir)
		_, err := store.Run(context.Background(), os.Getenv("RFORGE_JOB_TEST_ID"), func(ctx context.Context, run *Run) error {
			if err := run.Checkpoint(Progress{Completed: 1, Total: 2}, ""); err != nil {
				return err
			}
			fmt.Println("ready")
			time.Sleep(time.Hour)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	store := NewStore(t.TempDir())
	job, err := store.Create("fixture", map[string]string{"query": "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStoreKilledJobResumes$")
	cmd.Env = append(os.Environ(), "RFORGE_JOB_TEST_DIR="+store.Dir, "RFORGE_JOB_TEST_ID="+job.ID)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	scanner := bufio.NewScanner(pipe)
	if !scanner.Scan() || scanner.Text() != "ready" {
		t.Fatal("worker never checkpointed")
	}
	active, err := store.Status(job.ID)
	if err != nil || active.Status != Running || active.Progress.Completed != 1 {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if _, err := store.Run(ctx, job.ID, nil); !errors.Is(err, ErrBusy) {
		t.Fatalf("concurrent run: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	interrupted, err := store.Status(job.ID)
	if err != nil || interrupted.Status != Interrupted || interrupted.StopReason != "interrupted" {
		t.Fatalf("killed=%+v err=%v", interrupted, err)
	}
	listed, err := store.List()
	if err != nil || len(listed) != 1 || listed[0].ID != job.ID {
		t.Fatalf("list=%+v err=%v", listed, err)
	}
	finish := func(ctx context.Context, run *Run) error {
		return run.Checkpoint(Progress{Completed: 2, Total: 2}, "completed")
	}
	final, err := store.Run(ctx, job.ID, finish)
	if err != nil || final.Status != Completed || final.StopReason != "completed" {
		t.Fatalf("resumed=%+v err=%v", final, err)
	}
	control, err := store.Create("fixture", map[string]string{"query": "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	control, err = store.Run(ctx, control.ID, finish)
	if err != nil || !reflect.DeepEqual(final.Progress, control.Progress) || string(final.Input) != string(control.Input) {
		t.Fatalf("resumed/control mismatch: %+v %+v %v", final, control, err)
	}
	if _, err := store.Status("../escape"); err == nil {
		t.Fatal("accepted path traversal as job ID")
	}
	if _, err := store.Run(ctx, job.ID, nil); !errors.Is(err, ErrNotResumable) {
		t.Fatalf("resumed completed job: %v", err)
	}
}
