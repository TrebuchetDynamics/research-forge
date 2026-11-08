package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestHTTPClientDefaultWait(t *testing.T) {
	for _, tc := range []struct {
		name      string
		retry     bool
		cancel    bool
		deadline  bool
		preCancel bool
		wantErr   error
		wantCalls int
	}{
		{name: "cancel pacing", cancel: true, wantErr: context.Canceled},
		{name: "cancel retry", retry: true, cancel: true, wantErr: context.Canceled, wantCalls: 1},
		{name: "deadline", deadline: true, wantErr: context.DeadlineExceeded},
		{name: "already canceled", preCancel: true, wantErr: context.Canceled},
		{name: "pacing succeeds", wantCalls: 1},
		{name: "retry succeeds", retry: true, wantCalls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if tc.deadline {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(ctx, time.Second)
					defer stop()
				}
				if tc.preCancel {
					cancel()
				}
				calls := 0
				opts := HTTPClientOptions{
					BaseURL: "https://wait-fixture.invalid", RequestDelay: 10 * time.Second, MaxRetries: 1,
					Transport: waitRoundTripper(func(r *http.Request) (*http.Response, error) {
						calls++
						status := http.StatusOK
						if tc.retry && calls == 1 {
							status = http.StatusTooManyRequests
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": {"10"}}, Body: io.NopCloser(strings.NewReader("ok")), Request: r}, nil
					}),
				}
				if tc.retry {
					opts.RequestDelay = 0
				}
				client := NewHTTPClient(opts) // Exercise the default wait, not Sleep or Wait hooks.
				done := make(chan error, 1)
				start := time.Now()
				go func() {
					_, err := client.Get(ctx, "/works", nil)
					done <- err
				}()
				// All work is now blocked in the pacing/backoff timer (or has
				// already returned for pre-canceled input); no wall-clock sleeps.
				synctest.Wait()
				if !tc.preCancel {
					select {
					case err := <-done:
						t.Fatalf("returned before wait: %v", err)
					default:
					}
				}
				switch {
				case tc.cancel:
					cancel()
				case tc.deadline:
					time.Sleep(time.Second)
				case !tc.preCancel:
					time.Sleep(10 * time.Second)
				}
				synctest.Wait()
				select {
				case err := <-done:
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("Get error = %v, want %v", err, tc.wantErr)
					}
				default:
					t.Error("Get did not return promptly after cancellation/deadline")
					<-done // Join even a broken implementation before leaving the bubble.
				}
				if calls != tc.wantCalls {
					t.Errorf("requests = %d, want %d", calls, tc.wantCalls)
				}
				if tc.wantErr == nil && time.Since(start) != 10*time.Second {
					t.Errorf("wait = %s, want 10s", time.Since(start))
				}
			}) // synctest joins every goroutine before returning.
		})
	}
}

func TestHTTPClientWaitHookPrecedesSleep(t *testing.T) {
	ctx := context.Background()
	calls, waits := 0, 0
	client := NewHTTPClient(HTTPClientOptions{
		BaseURL: "https://wait-fixture.invalid", RequestDelay: time.Second, MaxRetries: 1,
		Sleep: func(time.Duration) { t.Error("Sleep called despite Wait override") },
		Wait: func(got context.Context, d time.Duration, retry bool) error {
			wantDelay := time.Second
			if waits == 1 {
				wantDelay = 2 * time.Second
			}
			if got != ctx || d != wantDelay || retry != (waits == 1) {
				t.Errorf("wait %d: ctx=%v delay=%s retry=%v", waits, got, d, retry)
			}
			waits++
			return nil
		},
		Transport: waitRoundTripper(func(r *http.Request) (*http.Response, error) {
			calls++
			status := http.StatusOK
			if calls == 1 {
				status = http.StatusTooManyRequests
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": {"2"}}, Body: io.NopCloser(strings.NewReader("ok")), Request: r}, nil
		}),
	})
	if _, err := client.Get(ctx, "/works", nil); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || waits != 3 {
		t.Fatalf("requests=%d waits=%d, want 2 and 3", calls, waits)
	}
}

type waitRoundTripper func(*http.Request) (*http.Response, error)

func (f waitRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
