package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.FormatInt(maxSourceResponseBytes+1, 10))
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second})
	_, err := client.Get(context.Background(), "/works", nil)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("Get error = %v, want too large", err)
	}
}

func TestHTTPClientGetUsesMockServerUserAgentAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q, want /works", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "artificial photosynthesis" {
			t.Fatalf("search query = %q", r.URL.Query().Get("search"))
		}
		if got := r.Header.Get("User-Agent"); got != "ResearchForge/0.1 (mailto:test@example.org)" {
			t.Fatalf("User-Agent = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"W1"}]}`))
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseURL:   server.URL,
		UserAgent: "ResearchForge/0.1 (mailto:test@example.org)",
		Timeout:   time.Second,
	})
	body, err := client.Get(context.Background(), "/works", map[string]string{"search": "artificial photosynthesis"})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(body) != `{"results":[{"id":"W1"}]}` {
		t.Fatalf("body = %s", body)
	}
}

func TestHTTPClientRetriesTransientServerErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseURL:    server.URL,
		UserAgent:  "ResearchForge/test",
		Timeout:    time.Second,
		MaxRetries: 1,
	})
	body, err := client.Get(context.Background(), "/works", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s", body)
	}
}

func TestHTTPClientHonorsRetryAfterForRateLimitWithInjectedSleep(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "2")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var slept []time.Duration
	client := NewHTTPClient(HTTPClientOptions{
		BaseURL:    server.URL,
		UserAgent:  "ResearchForge/test",
		Timeout:    time.Second,
		MaxRetries: 1,
		Sleep: func(duration time.Duration) {
			slept = append(slept, duration)
		},
	})
	_, err := client.Get(context.Background(), "/works", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if len(slept) != 1 || slept[0] != 2*time.Second {
		t.Fatalf("slept = %#v, want [2s]", slept)
	}
}

func TestHTTPClientBacksOffExponentiallyWithoutRetryAfterOnRateLimit(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests <= 2 {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var slept []time.Duration
	client := NewHTTPClient(HTTPClientOptions{
		BaseURL:    server.URL,
		MaxRetries: 2,
		Sleep:      func(d time.Duration) { slept = append(slept, d) },
	})
	_, err := client.Get(context.Background(), "/works", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
	if len(slept) != 2 {
		t.Fatalf("sleep count = %d, want 2", len(slept))
	}
	if slept[0] != 1*time.Second || slept[1] != 2*time.Second {
		t.Fatalf("slept = %v, want [1s 2s]", slept)
	}
}

func TestHTTPClientRateLimitExhaustedReturnsActionableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseURL:    server.URL,
		MaxRetries: 1,
		Sleep:      func(d time.Duration) {},
	})
	_, err := client.Get(context.Background(), "/works", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("error %q should mention rate limited for 429 so users can act on it", err.Error())
	}
}

func TestHTTPClientDoesNotBackOffOnServerErrorWithoutRetryAfter(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var slept []time.Duration
	client := NewHTTPClient(HTTPClientOptions{
		BaseURL:    server.URL,
		MaxRetries: 1,
		Sleep:      func(d time.Duration) { slept = append(slept, d) },
	})
	_, err := client.Get(context.Background(), "/works", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if len(slept) != 0 {
		t.Fatalf("slept = %v, want no sleep on 500 retry", slept)
	}
}

func TestHTTPClientAppliesTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Millisecond})
	_, err := client.Get(context.Background(), "/works", nil)
	if err == nil {
		t.Fatalf("Get returned nil error, want timeout")
	}
}

func TestHTTPClientRejectsAbsoluteRequestPath(t *testing.T) {
	client := NewHTTPClient(HTTPClientOptions{BaseURL: "http://127.0.0.1", UserAgent: "ResearchForge/test", Timeout: time.Second})

	_, err := client.Get(context.Background(), "https://example.org/works", nil)
	if err == nil {
		t.Fatalf("Get returned nil error for absolute request path")
	}
}
