package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the real connector, date guard, projection and credential policy,
// then replay the resulting package without credentials or a live transport.
func TestSemanticScholarRecordReplay(t *testing.T) {
	for _, tc := range []struct{ cap, redirect string }{
		{cap: ":2020-06-30"}, {cap: ""}, {cap: ":2020-07-01"},
		{cap: ":2020-02-30"}, {cap: ":2020"}, {cap: ":2020-06-30:2021-01-01"},
		{cap: ":2020-06-30", redirect: "https://api.semanticscholar.org/redirect"},
		{cap: ":2020-06-30", redirect: "https://api.semanticscholar.org/redirect/private-semantic-scholar%2Fkey%2Bvalue"},
		{cap: ":2020-06-30", redirect: "https://unrelated.example/redirect"},
		{cap: ":2020-06-30", redirect: "http://api.semanticscholar.org/redirect"},
		{cap: ":2020-06-30", redirect: "https://api.semanticscholar.org:8443/redirect"},
	} {
		t.Run(tc.cap+tc.redirect, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"set.json", "review.md"} {
				b, err := os.ReadFile(filepath.Join("testdata/fixture-tiny", name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			b, _ := os.ReadFile(filepath.Join(dir, "set.json"))
			var set Set
			if err := json.Unmarshal(b, &set); err != nil {
				t.Fatal(err)
			}
			q := &set.Questions[0]
			q.Gold[1].Identifiers = []Identifier{{Scheme: "semantic_scholar", Value: "cccccccccccccccccccccccccccccccccccccccc"}}
			q.Gold[2].Identifiers = []Identifier{{Scheme: "arxiv", Value: "1901.00002"}}
			for i := range q.SearchPlan {
				call := &q.SearchPlan[i]
				call.Source = "semantic-scholar"
				call.Filters = map[string]string{"publicationDateOrYear": tc.cap}
				if i == 1 {
					call.PageCursor = "10"
				}
			}
			set.State, set.SnapshotsSHA256, set.SetChecksum = "approved", nil, ""
			b, _ = CanonicalJSON(set)
			v, _ := parseJSON(b)
			delete(v.(map[string]any), "set_checksum")
			b, _ = canonical(v)
			set.SetChecksum = digest(b)
			b, _ = CanonicalJSON(set)
			if err := os.WriteFile(filepath.Join(dir, "set.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			const key = "private-semantic-scholar/key+value"
			t.Setenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY", key)
			requests, wantRequests := 0, 2
			if tc.redirect != "" {
				wantRequests++
			}
			o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test-revision", RecordTransport: fixtureTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				params := req.URL.Query()
				offset, host, path, wantKey := "", "api.semanticscholar.org", "/graph/v1/paper/search", key
				if tc.redirect != "" && requests == 2 {
					target, _ := url.Parse(tc.redirect)
					host, path = target.Host, target.Path
					if target.Scheme != "https" || host != "api.semanticscholar.org" {
						wantKey = ""
					}
				}
				if requests == wantRequests {
					offset = "10"
				}
				if req.URL.Host != host || req.URL.Path != path || params.Get("query") != "synthetic energy" || params.Get("publicationDateOrYear") != ":2020-06-30" || params.Get("offset") != offset || params.Get("limit") != "10" || req.Header.Get("x-api-key") != wantKey {
					t.Fatalf("incorrect capped request or credential policy: %s keyPresent=%t", req.URL, req.Header.Get("x-api-key") != "")
				}
				if tc.redirect != "" && requests == 1 {
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{tc.redirect + "?" + req.URL.RawQuery}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
				}
				body := `{"next":10,"data":[
					{"paperId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","title":"Precise date paper","year":2021,"publicationDate":"2020-06-30","externalIds":{"DOI":"10.9999/p1"},"abstract":"must not redistribute","url":"https://example.org/paper?api_key=private-semantic-scholar%2Fkey%2Bvalue&x-api-key=private-semantic-scholar%2Fkey%2Bvalue&X-%61pi-Key=private-semantic-scholar%2Fkey%2Bvalue"},
					{"paperId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","title":"Year only paper","year":2019,"publicationDate":null,"externalIds":{"ArXiv":"1901.00002"}},
					{"paperId":"cccccccccccccccccccccccccccccccccccccccc","title":"Native ID paper","year":2021,"publicationDate":"2019-02-01","externalIds":{"PubMed":"333"}},
					{"paperId":"dddddddddddddddddddddddddddddddddddddddd","title":"After cap paper","year":2019,"publicationDate":"2020-07-01"},
					{"paperId":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","title":"Unknown date paper"},
					{"paperId":"ffffffffffffffffffffffffffffffffffffffff","title":"Coarse date paper","year":2020,"publicationDate":null}
				],"x-api-key":"private-semantic-scholar/key+value","message":"echoed private-semantic-scholar/key+value; query private-semantic-scholar%2Fkey%2Bvalue; path private-semantic-scholar%2Fkey+value"}`
				if requests == wantRequests {
					body = `{"data":[]}`
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Api-Key": []string{key}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}
			capture, err := RunEngine(context.Background(), o)
			if tc.cap != ":2020-06-30" {
				if err == nil || !bytes.Contains(capture.Report, []byte("date_cap_unenforceable")) || requests != 0 {
					t.Fatalf("unsafe cap accepted: %v %s requests=%d", err, capture.Report, requests)
				}
				return
			}
			if err != nil {
				t.Fatalf("record: %v %s", err, capture.Report)
			}
			t.Setenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY", "")
			o.Record, o.RecordTransport = false, nil
			replay, err := RunEngine(context.Background(), o)
			if err != nil {
				t.Fatalf("replay: %v %s", err, replay.Report)
			}
			var report EngineReport
			if err := json.Unmarshal(capture.Report, &report); err != nil {
				t.Fatal(err)
			}
			report.Mode = "replay"
			want, _ := CanonicalJSON(report)
			if !bytes.Equal(want, replay.Report) || !bytes.Equal(capture.Audit, replay.Audit) {
				t.Fatal("record/replay bytes differ")
			}
			run := report.Runs[0]
			if requests != wantRequests || *run.Returned != 3 || *run.DateExcluded != 3 || *run.Recall.Value != 1 {
				t.Fatalf("incorrect Semantic Scholar results: %s", replay.Report)
			}
			if err := filepath.WalkDir(filepath.Join(dir, "snapshots"), func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				b, err := os.ReadFile(path)
				for _, secret := range []string{key, "private-semantic-scholar%2Fkey%2Bvalue", "private-semantic-scholar%2Fkey+value", "must not redistribute"} {
					if bytes.Contains(b, []byte(secret)) {
						t.Errorf("credential or abstract retained in %s", path)
					}
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
