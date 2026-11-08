package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
)

func TestSearchDisconnectDuringRetryWait(t *testing.T) {
	// Initialize process-global signal plumbing outside the virtual-time bubble.
	_, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	stopSignals()
	var requests atomic.Int32
	original := http.DefaultTransport
	http.DefaultTransport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		if r.URL.Host != "retry-fixture.invalid" {
			return nil, fmt.Errorf("unexpected request: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"30"}}, Body: io.NopCloser(strings.NewReader("rate limited")), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	t.Setenv("RFORGE_OPENALEX_URL", "https://retry-fixture.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	t.Setenv("RFORGE_SOURCE_MAX_RETRY_AFTER", "30s")
	dir := filepath.Join(t.TempDir(), "research")
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		// Reuse the real CLI host and MCP transport, with no replacement wait.
		session, stopServer := startServer(t, ctx, "")
		defer func() {
			stopServer()
			// Also join a regressed, noncancelable 30s sleeper on failure.
			time.Sleep(30 * time.Second)
			synctest.Wait()
		}()
		data := call(t, ctx, session, "search", map[string]any{"dir": dir, "queries": []string{"retry"}, "sources": []string{"openalex"}})
		var response struct {
			Data struct {
				ID string `json:"job_id"`
			}
		}
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		// The worker must have reached its default retry timer, not merely
		// returned a response from the fake source.
		synctest.Wait()
		if requests.Load() != 1 {
			t.Fatalf("requests before disconnect = %d, want 1", requests.Load())
		}
		lease, err := jobs.NewStore(dir).Acquire()
		if !errors.Is(err, jobs.ErrBusy) {
			if lease != nil {
				lease.Close()
			}
			t.Fatalf("search did not hold its directory lease: %v", err)
		}
		start := time.Now()
		stopServer()
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Fatalf("disconnect took %s during a 30s retry wait", elapsed)
		}
		// Read the durable record directly: Status may recover a stale running
		// job and would hide a failure to checkpoint before server exit.
		var job jobs.Job
		if err := json.Unmarshal(read(t, filepath.Join(dir, "jobs", response.Data.ID+".json")), &job); err != nil {
			t.Fatal(err)
		}
		if job.Status != jobs.Interrupted || job.StopReason != "interrupted" || len(job.Input) == 0 {
			t.Fatalf("job was not saved as resumable: %+v", job)
		}
		_ = read(t, filepath.Join(dir, "batch-state.json"))
		lease, err = jobs.NewStore(dir).Acquire()
		if err != nil {
			t.Fatalf("directory lease not released: %v", err)
		}
		lease.Close()
		if requests.Load() != 1 {
			t.Fatalf("post-cancellation retry: %d requests", requests.Load())
		}
	})
}
