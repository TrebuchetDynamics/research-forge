package jobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackgroundJobCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var ticks int32
	job := NewBackgroundJob(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				atomic.AddInt32(&ticks, 1)
				time.Sleep(time.Millisecond)
			}
		}
	})
	done := make(chan error, 1)
	go func() { done <- job.Run(ctx) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("job did not cancel")
	}
	if atomic.LoadInt32(&ticks) < 0 {
		t.Fatalf("invalid ticks")
	}
}
