package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCitationsExpandStructuredStop(t *testing.T) {
	for _, tc := range []struct {
		name, source, direction, depth, reason string
		requests                               int
	}{
		{"completed", "semantic-scholar", "references", "1", "completed", 1},
		{"request budget", "semantic-scholar", "references", "2", "budget_exhausted", 1},
		{"both counts each request", "semantic-scholar", "both", "2", "budget_exhausted", 1},
		{"crossref", "crossref", "references", "1", "completed", 1},
		{"openalex", "openalex", "references", "1", "completed", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if tc.source == "crossref" {
					_, _ = w.Write([]byte(`{"message":{"DOI":"10.1000/seed","reference":[{"DOI":"10.1000/neighbor"}]}}`))
					return
				}
				if tc.source == "openalex" {
					_, _ = w.Write([]byte(`{"id":"https://openalex.org/W1","referenced_works":["https://openalex.org/W2"]}`))
					return
				}
				_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"S2","title":"Neighbor"}}]}`))
			}))
			defer server.Close()
			t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
			t.Setenv("RFORGE_CROSSREF_URL", server.URL)
			t.Setenv("RFORGE_OPENALEX_URL", server.URL)
			seed := "S1"
			if tc.source == "crossref" {
				seed = "10.1000/seed"
			}
			if tc.source == "openalex" {
				seed = "W1"
			}
			t.Setenv("RFORGE_SEMANTIC_SCHOLAR_REQUEST_DELAY", "0s")
			var stdout, stderr bytes.Buffer
			code := Execute([]string{"--json", "citations", "expand", "--source", tc.source, "--paper", seed, "--direction", tc.direction, "--depth", tc.depth, "--max-api-calls", "1", "--out", filepath.Join(t.TempDir(), "graph.json")}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr.String())
			}
			var got struct {
				Data struct {
					StopReason   string `json:"stop_reason"`
					RequestsUsed int    `json:"requests_used"`
					RawRef       string `json:"rawRef"`
				}
			}
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Data.StopReason != tc.reason || got.Data.RequestsUsed != tc.requests || int(calls.Load()) != tc.requests || strings.Contains(got.Data.RawRef, "budget_stopped") {
				t.Fatalf("structured result: %s", stdout.String())
			}
		})
	}
}
