package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Catch double rounding at the CLI seam: JSON rounds 0.125 to 0.13,
// while the existing terminal formatter displays the raw score as 0.12.
func TestRankTextPreservesScorePrecision(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = rankFixtureTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "rank.invalid" || r.URL.Query().Get("search") != "precision" {
			t.Fatalf("unexpected source request: %s", r.URL)
		}
		body := fmt.Sprintf(`{"results":[{"id":"W1","title":"Precision paper","publication_year":%d}]}`, time.Now().Year()-15)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	t.Setenv("RFORGE_OPENALEX_URL", "https://rank.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	var out, stderr bytes.Buffer
	code := Execute([]string{"--project", t.TempDir(), "rank", "precision", "--sources", "openalex"}, &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "[0.12] Precision paper") {
		t.Fatalf("rank=%d: %s %s", code, &out, &stderr)
	}
}

type rankFixtureTransport func(*http.Request) (*http.Response, error)

func (f rankFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
