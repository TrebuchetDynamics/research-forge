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

func TestOpenAlexKeyCaptureRedactsEchoesAndReplaysWithoutKey(t *testing.T) {
	const key = "openalex-test-secret+/="
	t.Setenv("RFORGE_OPENALEX_API_KEY", key)
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
	b, err := os.ReadFile(filepath.Join(dir, "set.json"))
	if err != nil {
		t.Fatal(err)
	}
	var set Set
	if err := json.Unmarshal(b, &set); err != nil {
		t.Fatal(err)
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
	o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test-revision", RecordTransport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("OpenAlex request missing configured bearer key")
		}
		if requests == 1 {
			return &http.Response{StatusCode: 302, Header: http.Header{"Location": {"/redirect/" + url.PathEscape(key) + "?echo=" + url.QueryEscape(key)}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}
		if requests == 2 && (r.URL.Path != "/redirect/"+key || r.URL.Query().Get("echo") != key) {
			t.Error("redaction altered the live redirect request")
		}
		body := `{"results":[{"id":"https://openalex.org/W1","doi":"10.9999/p1","title":"Synthetic primary paper","publication_date":"2019-01-01"}],"error":"Bearer ` + key + `; encoded ` + url.QueryEscape(key) + `; https://api.openalex.org/works?api_key=` + url.QueryEscape(key) + `"}`
		headers := http.Header{"Link": {`<https://example.org/` + key + `>; rel="next"`}}
		return &http.Response{StatusCode: 200, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	capture, err := RunEngine(context.Background(), o)
	if err != nil {
		t.Fatalf("record: %v %s", err, capture.Report)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
	assertSafe := func(b []byte) {
		t.Helper()
		for _, secret := range []string{key, url.QueryEscape(key), url.PathEscape(key)} {
			if bytes.Contains(b, []byte(secret)) {
				t.Fatal("OpenAlex key retained in capture or output")
			}
		}
	}
	assertSafe(capture.Report)
	assertSafe(capture.Audit)
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err == nil {
			assertSafe(b)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RFORGE_OPENALEX_API_KEY", "")
	o.Record, o.RecordTransport = false, nil
	replay, err := RunEngine(context.Background(), o)
	if err != nil {
		t.Fatalf("keyless replay: %v %s", err, replay.Report)
	}
	if !bytes.Equal(capture.Audit, replay.Audit) {
		t.Fatal("keyless replay changed captured results")
	}
	assertSafe(replay.Report)
}
