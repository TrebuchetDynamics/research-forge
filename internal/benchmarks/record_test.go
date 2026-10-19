package benchmarks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// A capture must publish a replayable locked package, not normalized records,
// and must leave approval untouched when a forbidden review is encountered.
func TestEngineRecordPublication(t *testing.T) {
	for _, tc := range []struct {
		name                                                  string
		leak, redirect, budget, urlOnly, crossref, unsafeTemp bool
	}{{name: "capture"}, {name: "rollback leak", leak: true}, {name: "rollback leak with unsafe TMPDIR", leak: true, unsafeTemp: true}, {name: "redirect and retry", redirect: true}, {name: "request budget", budget: true}, {name: "URL-only identity", urlOnly: true}, {name: "Crossref precise online date and cursor", crossref: true}} {
		t.Run(tc.name, func(t *testing.T) {
			leak := tc.leak
			attempt := 0
			dir := t.TempDir()
			if tc.unsafeTemp {
				t.Setenv("TMPDIR", dir)
			}
			for _, p := range []string{"set.json", "review.md"} {
				b, e := os.ReadFile(filepath.Join("testdata/fixture-tiny", p))
				if e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(filepath.Join(dir, p), b, 0600); e != nil {
					t.Fatal(e)
				}
			}
			b, _ := os.ReadFile(filepath.Join(dir, "set.json"))
			var s Set
			if e := json.Unmarshal(b, &s); e != nil {
				t.Fatal(e)
			}
			if tc.budget {
				s.Questions[0].Budgets.MaxHTTPRequests = 1
			}
			if tc.urlOnly {
				s.Questions[0].Gold[0].Identifiers = []Identifier{{Scheme: "url", Value: "https://example.org/paper?b=2&a=1"}}
			}
			if tc.crossref {
				for i := range s.Questions[0].SearchPlan {
					call := &s.Questions[0].SearchPlan[i]
					call.Source = "crossref"
					call.Filters = map[string]string{"filter": "until-pub-date:2020-06-30"}
				}
			}
			s.State = "approved"
			s.SnapshotsSHA256 = nil
			s.SetChecksum = ""
			v, _ := CanonicalJSON(s)
			m, _ := parseJSON(v)
			delete(m.(map[string]any), "set_checksum")
			j, _ := canonical(m)
			s.SetChecksum = digest(j)
			approved, _ := CanonicalJSON(s)
			if e := os.WriteFile(filepath.Join(dir, "set.json"), approved, 0600); e != nil {
				t.Fatal(e)
			}
			doi := "10.9999/p1"
			if tc.urlOnly {
				doi = ""
			}
			if leak {
				doi = "10.9999/review"
			}
			o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test-revision", RecordTransport: fixtureTransport(func(req *http.Request) (*http.Response, error) {
				attempt++
				if tc.redirect && attempt == 1 {
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"/redirect?token=private-value"}, "Link": []string{`<https://example.org/page?api_key=private-value>; rel="next"`}}, Body: io.NopCloser(bytes.NewBufferString("")), Request: req}, nil
				}
				if tc.redirect && attempt == 2 {
					status := 503
					if req.URL.Query().Get("token") != "private-value" {
						status = 403
					}
					return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(bytes.NewBufferString("<html>temporarily unavailable</html>")), Request: req}, nil
				}
				body := `{"results":[{"id":"https://openalex.org/W1","doi":"` + doi + `","title":"Synthetic primary paper","publication_year":2019,"publication_date":"2019-01-01","primary_location":{"landing_page_url":"https://example.org/paper?b=2&a=1"},"abstract":"must not redistribute"}],"error":"request https://api.example.org/x?api_key=private-value denied"}`
				if req.URL.Query().Get("cursor") != "" {
					body = `{"results":[]}`
				}
				if tc.crossref {
					body = `{"message":{"items":[{"DOI":"10.9999/p1","title":["Synthetic primary paper"],"published-online":{"date-parts":[[2019,1,1]]},"published-print":{"date-parts":[[2021]]}}]}}`
					if req.URL.Query().Get("cursor") != "" {
						body = `{"message":{"items":[]}}`
					}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewBufferString(body)), Request: req}, nil
			})}
			got, e := RunEngine(context.Background(), o)
			if leak {
				if len(got.Audit) != 0 {
					t.Fatal("failed capture exposes audit for CLI to write outside diagnostics")
				}
				if tc.unsafeTemp && (got.DiagnosticsDir != "" || got.DiagnosticsError == nil) {
					t.Fatal("unsafe diagnostics TMPDIR accepted")
				}
				after, _ := os.ReadFile(filepath.Join(dir, "set.json"))
				if e == nil || !bytes.Contains(got.Report, []byte("reference_review_leak")) || !bytes.Equal(after, approved) {
					t.Fatalf("leak published: %v %s", e, got.Report)
				}
				if _, e = os.Stat(filepath.Join(dir, "snapshots")); !os.IsNotExist(e) {
					t.Fatal("failed capture left bundle")
				}
				return
			}
			if e != nil {
				t.Fatalf("record: %v %s", e, got.Report)
			}
			o.Record = false
			o.RecordTransport = nil
			replay, e := RunEngine(context.Background(), o)
			if e != nil {
				t.Fatalf("recorded bundle cannot replay: %v %s", e, replay.Report)
			}
			var r EngineReport
			if e = json.Unmarshal(replay.Report, &r); e != nil {
				t.Fatal(e)
			}
			spec, err := os.ReadFile("../../docs/benchmark/spec.md")
			if err != nil {
				t.Fatal(err)
			}
			if want := fmt.Sprintf("%x", sha256.Sum256(spec)); r.ProtocolSHA256 != want {
				t.Fatalf("report identifies the wrong protocol: got %s, spec %s", r.ProtocolSHA256, want)
			}
			requests := int64(2)
			if tc.redirect {
				requests = 4
			}
			if tc.budget {
				requests = 1
				if *r.Runs[0].SearchCalls != 1 || r.Runs[0].StopReason != "budget_exhausted" {
					t.Fatalf("request cap bypassed at call: %s", replay.Report)
				}
			}
			if r.Status != "complete" || *r.Runs[0].Recall.Value != 1.0/3 || *r.Runs[0].HTTPRequests != requests {
				t.Fatalf("bad replay: %s", replay.Report)
			}
			index, _ := os.ReadFile(filepath.Join(dir, "snapshots/index.json"))
			if bytes.Contains(index, []byte("private-value")) {
				t.Fatal("credential retained in header")
			}
			files, _ := filepath.Glob(filepath.Join(dir, "snapshots/bodies/*"))
			for _, p := range files {
				b, _ := os.ReadFile(p)
				if bytes.Contains(b, []byte("must not redistribute")) || bytes.Contains(b, []byte("private-value")) {
					t.Fatal("abstract or credential stored")
				}
			}
		})
	}
}
