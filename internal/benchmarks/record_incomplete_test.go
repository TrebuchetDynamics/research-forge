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

// A capture in which any question ends in a source error (here HTTP 429 from
// an exhausted provider budget) is not a lockable baseline: the approved
// manifest stays byte-identical and no snapshot bundle is published.
func TestEngineRecordRefusesIncompleteCapture(t *testing.T) {
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
	o := EngineOptions{SetDir: dir, Record: true, Version: "test", Revision: "test-revision", RecordTransport: fixtureTransport(func(req *http.Request) (*http.Response, error) {
		body := `{"error":"Rate limit exceeded","message":"Insufficient budget"}`
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"28277"}}, Body: io.NopCloser(bytes.NewBufferString(body)), Request: req}, nil
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
}
