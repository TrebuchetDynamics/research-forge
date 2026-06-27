package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSemanticScholarMaxRetryAfterOverridesGlobalCap verifies that
// RFORGE_SEMANTIC_SCHOLAR_MAX_RETRY_AFTER takes precedence over the
// global RFORGE_SOURCE_MAX_RETRY_AFTER for Semantic Scholar requests.
//
// Without the fix, defaultSemanticScholarHTTPClient reads only
// RFORGE_SOURCE_MAX_RETRY_AFTER (capped at 1s here), so a Retry-After: 2
// response causes immediate failure. With the fix, the SS-specific cap of 5s
// allows waiting and retrying. The server returns success on the second
// request, producing a 0 exit code.
func TestSemanticScholarMaxRetryAfterOverridesGlobalCap(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			// Retry-After: 2 exceeds the global cap (1s) but fits the SS-specific cap (5s).
			w.Header().Set("Retry-After", "2")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"offset":0,"data":[]}`))
	}))
	defer server.Close()

	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	// Global cap too low to handle a 2-second Retry-After.
	t.Setenv("RFORGE_SOURCE_MAX_RETRY_AFTER", "1s")
	// SS-specific cap is high enough — the client must use this value.
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_MAX_RETRY_AFTER", "5s")

	code := Execute([]string{"search", "--source", "semantic-scholar", "--query", "fractal catalog"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0: RFORGE_SEMANTIC_SCHOLAR_MAX_RETRY_AFTER=5s should allow waiting 2s", code)
	}
	if requests < 2 {
		t.Errorf("requests = %d, want >= 2: the client should have retried", requests)
	}
}
