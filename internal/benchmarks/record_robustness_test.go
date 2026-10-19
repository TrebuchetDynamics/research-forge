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

func approvedRobustnessSet(t *testing.T, source, filter string, customize ...func(*Set)) string {
	t.Helper()
	dir := t.TempDir()
	b, err := os.ReadFile("testdata/fixture-tiny/set.json")
	if err != nil {
		t.Fatal(err)
	}
	var s Set
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	for i := range s.Questions[0].SearchPlan {
		s.Questions[0].SearchPlan[i].Source = source
		s.Questions[0].SearchPlan[i].Filters = map[string]string{"filter": filter}
	}
	for _, change := range customize {
		change(&s)
	}
	s.State, s.SnapshotsSHA256, s.SetChecksum = "approved", nil, ""
	b, _ = CanonicalJSON(s)
	v, _ := parseJSON(b)
	delete(v.(map[string]any), "set_checksum")
	b, _ = canonical(v)
	s.SetChecksum = digest(b)
	b, _ = CanonicalJSON(s)
	if err := os.WriteFile(filepath.Join(dir, "set.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile("testdata/fixture-tiny/review.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review.md"), b, 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Missing native/alias identities and failed PaperRecords normalization must
// exclude only the offending record, without dropping valid neighbours.
func TestEngineRecordExcludesOddRecords(t *testing.T) {
	for _, tc := range []struct {
		name, source, filter, body string
		reasons                    map[string]int
	}{
		{"untitled Crossref", "crossref", "until-pub-date:2020-06-30", `{"message":{"items":[{"DOI":"10.9999/odd","issued":{"date-parts":[[2019]]}},{"DOI":"10.9999/p1","title":["Good paper"],"issued":{"date-parts":[[2019]]}}]}}`, map[string]int{"normalization_failed": 1}},
		{"unidentified Europe PMC", "europepmc", "FIRST_PDATE:[* TO 2020-06-30]", `{"resultList":{"result":[{"title":"No identifiers","pubYear":"2019"},{"id":"PPR123","source":"PPR","title":"Preprint","pubYear":"2019"},{"id":"AGR123","source":"AGR","title":"Agricola","pubYear":"2019"},{"id":"123","source":"MED","title":"Good paper","pubYear":"2019"}]}}`, map[string]int{"missing_identity": 1, "normalization_failed": 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := approvedRobustnessSet(t, tc.source, tc.filter)
			o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test", RecordTransport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewBufferString(tc.body)), Request: r}, nil
			})}
			capture, err := RunEngine(context.Background(), o)
			if err != nil {
				t.Fatalf("odd record failed capture: %v %s", err, capture.Report)
			}
			b, err := os.ReadFile(filepath.Join(dir, "snapshots/index.json"))
			if err != nil {
				t.Fatal(err)
			}
			var idx struct {
				Questions []struct {
					Exclusions []DateExclusion `json:"record_exclusions"`
				}
			}
			if err := json.Unmarshal(b, &idx); err != nil {
				t.Fatal(err)
			}
			reasons := map[string]int{}
			for _, ex := range idx.Questions[0].Exclusions {
				reasons[ex.Reason]++
			}
			if structuredHash(reasons) != structuredHash(tc.reasons) {
				t.Fatalf("wrong exclusions: %s", b)
			}
			var report EngineReport
			if err := json.Unmarshal(capture.Report, &report); err != nil {
				t.Fatal(err)
			}
			if *report.Runs[0].Returned != 1 || *report.Runs[0].DateExcluded != 0 {
				t.Fatalf("bad counts: %s", capture.Report)
			}
			o.Record, o.RecordTransport = false, nil
			replay, err := RunEngine(context.Background(), o)
			report.Mode = "replay"
			want, _ := CanonicalJSON(report)
			if err != nil || !bytes.Equal(want, replay.Report) || !bytes.Equal(capture.Audit, replay.Audit) {
				t.Fatalf("replay differs: %v %s", err, replay.Report)
			}
		})
	}
}

// Return over-cap bytes and a read error in the same Read: the hard cap
// takes precedence even when a subsequent retry would succeed.
func TestEngineRecordRejectsOverCapReadError(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	dir := approvedRobustnessSet(t, "openalex", "to_publication_date:2020-06-30", func(s *Set) { s.HTTPPolicy.MaxResponseBytes = 32 })
	before, _ := os.ReadFile(filepath.Join(dir, "set.json"))
	requests := 0
	got, err := RunEngine(context.Background(), EngineOptions{SetDir: dir, Record: true, RecordTransport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		body := io.ReadCloser(io.NopCloser(bytes.NewBufferString(`{"results":[]}`)))
		if requests == 1 {
			body = overCapCaptureBody{}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: body, Request: r}, nil
	})})
	if err == nil || !bytes.Contains(got.Report, []byte("protocol_violation")) || bytes.Contains(got.Report, []byte("snapshot_mismatch")) {
		t.Fatalf("cap bypassed: %v %s", err, got.Report)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "set.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("cap violation changed manifest")
	}
	b, err := os.ReadFile(filepath.Join(got.DiagnosticsDir, "failures.json"))
	if err != nil || !bytes.Contains(b, []byte("max_response_bytes")) {
		t.Fatalf("cap trace missing: %v %s", err, b)
	}
}

type overCapCaptureBody struct{}

func (overCapCaptureBody) Read(p []byte) (int, error) {
	return copy(p, "123456789012345678901234567890123"), io.ErrUnexpectedEOF
}
func (overCapCaptureBody) Close() error { return nil }

type brokenCaptureBody struct{}

func (brokenCaptureBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (brokenCaptureBody) Close() error             { return nil }

// A provider disconnect after a successful call, including retries/backoff,
// is a real source error, never a replay consistency failure.
func TestEngineRecordReadFailureKeepsDiagnostics(t *testing.T) {
	scratch := t.TempDir()
	t.Setenv("TMPDIR", scratch)
	dir := approvedRobustnessSet(t, "openalex", "to_publication_date:2020-06-30")
	before, _ := os.ReadFile(filepath.Join(dir, "set.json"))
	requests := 0
	got, err := RunEngine(context.Background(), EngineOptions{SetDir: dir, Record: true, RecordTransport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		body := io.ReadCloser(io.NopCloser(bytes.NewBufferString(`{"results":[]}`)))
		status := 200
		header := http.Header{}
		if requests == 2 {
			status = 503
			header.Set("Retry-After", "1")
		}
		if requests > 2 {
			body = brokenCaptureBody{}
		}
		return &http.Response{StatusCode: status, Header: header, Body: body, Request: r}, nil
	})})
	if err == nil || !bytes.Contains(got.Report, []byte("capture_incomplete")) || bytes.Contains(got.Report, []byte("snapshot_mismatch")) {
		t.Fatalf("wrong record failure: %v %s", err, got.Report)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "set.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed capture changed manifest")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("failed capture changed set: %v", entries)
	}
	dirs, _ := filepath.Glob(filepath.Join(scratch, "rforge-benchmark-diagnostics-*"))
	if len(dirs) != 1 || got.DiagnosticsDir != dirs[0] || got.DiagnosticsError != nil {
		t.Fatalf("diagnostics missing: %v result=%+v", dirs, got)
	}
	b, err := os.ReadFile(filepath.Join(dirs[0], "snapshots/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx SnapshotIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx.Questions) != 1 || len(idx.Questions[0].Exchanges) != 4 {
		t.Fatalf("attempts lost: %s", b)
	}
	b, err = os.ReadFile(filepath.Join(dirs[0], "failures.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"question_id":"q-one"`, `"plan_call":2`, `"provider":"openalex"`, `"http_status":503`, `"check":"source_error"`, "unexpected EOF"} {
		if !bytes.Contains(b, []byte(want)) {
			t.Fatalf("trace missing %s: %s", want, b)
		}
	}
}
