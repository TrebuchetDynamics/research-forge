package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// A source error must not lock a capture, even if the retry runs out of budget.
func TestEngineRecordRefusesIncompleteCapture(t *testing.T) {
	for _, tc := range []struct {
		name, source, retryAfter string
		requestCap               int
		wallCap                  int64
		retryRedirect            bool
	}{
		{name: "openalex", source: "openalex", retryAfter: "28277"},
		{name: "semantic-scholar", source: "semantic-scholar", retryAfter: "28277"},
		{name: "rate limit at request budget", source: "semantic-scholar", retryAfter: "1", requestCap: 1},
		{name: "rate limit at wall budget", source: "semantic-scholar", retryAfter: "2", wallCap: 1000},
		{name: "rate limit then redirect at request budget", source: "semantic-scholar", retryAfter: "1", requestCap: 2, retryRedirect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY", "")
			dir := t.TempDir()
			for _, p := range []string{"set.json", "review.md"} {
				b, err := os.ReadFile(filepath.Join("testdata/fixture-tiny", p))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, p), b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			b, _ := os.ReadFile(filepath.Join(dir, "set.json"))
			var s Set
			if err := json.Unmarshal(b, &s); err != nil {
				t.Fatal(err)
			}
			if tc.requestCap > 0 {
				s.Questions[0].Budgets.MaxHTTPRequests = tc.requestCap
			}
			if tc.wallCap > 0 {
				s.Questions[0].Budgets.MaxWallClockMS = tc.wallCap
			}
			if tc.source == "semantic-scholar" {
				for i := range s.Questions[0].SearchPlan {
					call := &s.Questions[0].SearchPlan[i]
					call.Source = tc.source
					call.Filters = map[string]string{"publicationDateOrYear": ":2020-06-30"}
				}
			}
			s.State = "approved"
			s.SnapshotsSHA256 = nil
			s.SetChecksum = ""
			v, _ := CanonicalJSON(s)
			m, _ := parseJSON(v)
			delete(m.(map[string]any), "set_checksum")
			c, _ := canonical(m)
			s.SetChecksum = digest(c)
			approved, _ := CanonicalJSON(s)
			if err := os.WriteFile(filepath.Join(dir, "set.json"), approved, 0o600); err != nil {
				t.Fatal(err)
			}
			requests := 0
			o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test-revision", RecordTransport: fixtureTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Header.Get("x-api-key") != "" {
					t.Fatal("anonymous capture unexpectedly used a key")
				}
				if tc.retryRedirect && requests == 2 {
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://api.semanticscholar.org/redirect"}}, Body: io.NopCloser(bytes.NewBufferString("")), Request: req}, nil
				}
				body := `{"error":"Rate limit exceeded","message":"Insufficient budget"}`
				return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{tc.retryAfter}}, Body: io.NopCloser(bytes.NewBufferString(body)), Request: req}, nil
			})}
			got, err := RunEngine(context.Background(), o)
			if err == nil || !bytes.Contains(got.Report, []byte("capture_incomplete")) {
				t.Fatalf("rate-limited capture was accepted: err=%v report=%s", err, got.Report)
			}
			after, _ := os.ReadFile(filepath.Join(dir, "set.json"))
			if !bytes.Equal(after, approved) {
				t.Fatalf("approved manifest changed by an incomplete capture:\n%s", after)
			}
			if _, err := os.Stat(filepath.Join(dir, "snapshots")); !os.IsNotExist(err) {
				t.Fatalf("incomplete capture left a snapshot bundle: %v", err)
			}
		})
	}
}
