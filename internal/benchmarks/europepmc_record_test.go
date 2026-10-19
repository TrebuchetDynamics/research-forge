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

func TestEuropePMCRecordReplay(t *testing.T) {
	for _, filter := range []string{
		"FIRST_PDATE:[* TO 2020-06-30]",
		"",
		"FIRST_PDATE:[* TO 2020-07-01]",
		"FIRST_PDATE:[* TO 2020-02-30]",
		"FIRST_PDATE:[* TO 2020-06-30] OR malaria",
		"FIRST_PDATE:[* TO 2020-06-30] FIRST_PDATE:[* TO 2021-01-01]",
	} {
		t.Run(filter, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"set.json", "review.md"} {
				b, err := os.ReadFile(filepath.Join("testdata/fixture-tiny", name))
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			b, _ := os.ReadFile(filepath.Join(dir, "set.json"))
			var set Set
			if err := json.Unmarshal(b, &set); err != nil {
				t.Fatal(err)
			}
			q := &set.Questions[0]
			// Exercise gold matching using each of Europe PMC's public identifiers.
			q.Gold[1].Identifiers = []Identifier{{Scheme: "pmid", Value: "333"}}
			q.Gold[2].Identifiers = []Identifier{{Scheme: "pmcid", Value: "PMC222"}}
			for i := range q.SearchPlan {
				q.SearchPlan[i].Source = "europepmc"
				q.SearchPlan[i].Terms = "synthetic energy OR malaria"
				q.SearchPlan[i].Filters = map[string]string{"filter": filter}
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
			requests := 0
			o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test-revision", RecordTransport: fixtureTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				params := req.URL.Query()
				cursor := "*"
				if requests == 2 {
					cursor = "page-two"
				}
				if req.URL.Host != "www.ebi.ac.uk" || req.URL.Path != "/europepmc/webservices/rest/search" || params.Get("query") != "(synthetic energy OR malaria) AND FIRST_PDATE:[* TO 2020-06-30]" || params.Get("cursorMark") != cursor || params.Get("resultType") != "core" || params.Get("pageSize") != "10" || params.Get("format") != "json" {
					t.Fatalf("uncapped or incorrect request: %s", req.URL)
				}
				body := `{"nextCursorMark":"page-two","resultList":{"result":[
					{"id":"111","pmid":"111","pmcid":"PMC111","doi":"10.9999/p1","title":"First date paper","pubYear":"2020","firstPublicationDate":"2020-06-30","abstractText":"must not redistribute"},
					{"id":"222","pmid":"222","pmcid":"PMC222","title":"Online date paper","pubYear":"2021","firstPublicationDate":"2021-01-01","electronicPublicationDate":"2019-01-01","journalInfo":{"printPublicationDate":"2021-02"}},
					{"id":"333","pmid":"333","title":"Print date paper","pubYear":"2021","firstPublicationDate":"2021-01-01","journalInfo":{"printPublicationDate":"2019-02"}},
					{"id":"444","pmid":"444","title":"After cap paper","pubYear":"2019","firstPublicationDate":"2020-07-01"},
					{"id":"555","pmid":"555","title":"Unknown date paper"}
				]}}`
				if requests == 2 {
					body = `{"resultList":{"result":[]}}`
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewBufferString(body)), Request: req}, nil
			})}
			capture, err := RunEngine(context.Background(), o)
			if filter != "FIRST_PDATE:[* TO 2020-06-30]" {
				if err == nil || !bytes.Contains(capture.Report, []byte("date_cap_unenforceable")) || requests != 0 {
					t.Fatalf("unsafe cap accepted: %v %s requests=%d", err, capture.Report, requests)
				}
				return
			}
			if err != nil {
				t.Fatalf("record: %v %s", err, capture.Report)
			}
			o.Record, o.RecordTransport = false, nil
			replay, err := RunEngine(context.Background(), o)
			if err != nil {
				t.Fatalf("replay: %v %s", err, replay.Report)
			}
			var report EngineReport
			if err := json.Unmarshal(capture.Report, &report); err != nil {
				t.Fatal(err)
			}
			// Record/replay reports differ only in their declared mode.
			report.Mode = "replay"
			want, _ := CanonicalJSON(report)
			if !bytes.Equal(want, replay.Report) || !bytes.Equal(capture.Audit, replay.Audit) {
				t.Fatal("record/replay bytes differ")
			}
			run := report.Runs[0]
			if requests != 2 || *run.Returned != 3 || *run.DateExcluded != 2 || *run.Recall.Value != 1 {
				t.Fatalf("incorrect Europe PMC results: %s", replay.Report)
			}
			var audit []QuestionAudit
			if err := json.Unmarshal(replay.Audit, &audit); err != nil {
				t.Fatal(err)
			}
			if audit[0].Raw[1].Metadata["publication_dates"] != "2021-01-01;2019-01-01;2021-02" {
				t.Fatalf("raw dates lost: %#v", audit[0].Raw[1])
			}
			b, _ = os.ReadFile(filepath.Join(dir, "snapshots/index.json"))
			var index SnapshotIndex
			if err := json.Unmarshal(b, &index); err != nil {
				t.Fatal(err)
			}
			reasons := map[string]int{}
			for _, exclusion := range index.Questions[0].DateExclusions {
				reasons[exclusion.Reason]++
			}
			if reasons["after_cap"] != 1 || reasons["unknown_date"] != 1 {
				t.Fatalf("date exclusions: %#v", reasons)
			}
			files, _ := filepath.Glob(filepath.Join(dir, "snapshots/bodies/*"))
			for _, name := range files {
				b, _ := os.ReadFile(name)
				if bytes.Contains(b, []byte("must not redistribute")) {
					t.Fatal("abstract retained in capture")
				}
			}
		})
	}
}
