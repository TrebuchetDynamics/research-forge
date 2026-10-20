package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A missing identifier must be a usage error, not a resolution attempt on the
// literal flag: `replicate --out plan.md` used to spend a network round trip
// resolving "--out" and then report `could not resolve "--out"`.
func TestReplicateRejectsFlagAsPaperIdentifier(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.NotFound(w, r)
	}))
	defer server.Close()
	for _, env := range []string{
		"RFORGE_OPENALEX_URL", "RFORGE_CROSSREF_URL", "RFORGE_ARXIV_URL",
		"RFORGE_SEMANTIC_SCHOLAR_URL", "RFORGE_UNPAYWALL_URL",
	} {
		t.Setenv(env, server.URL)
	}

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"replicate", "--out", "plan.md"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "usage: rforge replicate <doi|arxiv-id>") {
		t.Errorf("error does not show usage: %s", combined)
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Errorf("resolution was attempted %d time(s) for a flag value", got)
	}
}
