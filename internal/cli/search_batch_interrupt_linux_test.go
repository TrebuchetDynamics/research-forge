//go:build linux

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// TestExecuteSearchBatchSIGTERMCheckpointsAndExits reproduces the operational
// failure from real research sessions: a long sweep killed by an external
// timeout guard (SIGTERM). The sweep must stop gracefully, keep its partial
// results in the topic dir, record the checkpoint, exit with 128+15=143, and
// print the exact resume command — never orphaning results in a temp dir.
func TestExecuteSearchBatchSIGTERMCheckpointsAndExits(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if strings.Contains(r.URL.Query().Get("search"), "beta") {
			// Simulate the timeout guard firing mid-sweep: the signal
			// cancels the in-flight request before it completes.
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
			time.Sleep(500 * time.Millisecond)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/W-I","doi":"https://doi.org/10.0000/interrupt","title":"Interrupt fixture","publication_year":2026}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	out := t.TempDir()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"search", "batch", "--query", "alpha topic", "--query", "beta topic", "--query", "gamma topic", "--sources", "openalex", "--out", out, "--continue-on-error"}, stdout, stderr)
	if code != 128+int(syscall.SIGTERM) {
		t.Fatalf("interrupted sweep exit code = %d, want %d (128+SIGTERM); stdout=%s stderr=%s", code, 128+int(syscall.SIGTERM), stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "interrupted") || !strings.Contains(stdout.String(), "--resume") {
		t.Fatalf("interrupted stdout must explain the pending pairs and resume command: %s", stdout)
	}
	state := readBatchState(t, out)
	if !state.Interrupted {
		t.Fatalf("checkpoint must record the interruption: %+v", state)
	}
	if len(state.Completed) != 1 || state.Completed[0].Query != "alpha topic" {
		t.Fatalf("interrupted checkpoint completed pairs = %+v, want exactly alpha topic", state.Completed)
	}
	var manifest searchBatchManifest
	manifestData, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatalf("interrupted sweep must still commit aggregates: %v", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if !manifest.Interrupted || manifest.Pending != 2 {
		t.Fatalf("manifest interrupted/pending = %v/%d, want true/2 (%+v)", manifest.Interrupted, manifest.Pending, manifest)
	}

	// The interrupted topic dir must resume to completion without repeating
	// the finished pair.
	resumeServer := openAlexBatchServer(&requests, "Resumed")
	defer resumeServer.Close()
	t.Setenv("RFORGE_OPENALEX_URL", resumeServer.URL)
	resumeStdout := new(bytes.Buffer)
	resumeStderr := new(bytes.Buffer)
	resumeCode := Execute([]string{"search", "batch", "--query", "alpha topic", "--query", "beta topic", "--query", "gamma topic", "--sources", "openalex", "--out", out, "--continue-on-error", "--resume"}, resumeStdout, resumeStderr)
	if resumeCode != 0 {
		t.Fatalf("resume exit code = %d; stdout=%s stderr=%s", resumeCode, resumeStdout, resumeStderr)
	}
	finalState := readBatchState(t, out)
	if len(finalState.Completed) != 3 || finalState.Interrupted {
		t.Fatalf("post-resume checkpoint completed=%d interrupted=%v, want 3/false", len(finalState.Completed), finalState.Interrupted)
	}
	if got := atomic.LoadInt32(&requests); got != 4 {
		t.Fatalf("total requests across interrupt+resume = %d, want 4 (alpha + aborted beta, then beta+gamma retried)", got)
	}
}

// TestSearchBatchSignalContextCancelWithoutSignal covers the getter's zero
// value when the context is canceled programmatically.
func TestSearchBatchSignalContextCancelWithoutSignal(t *testing.T) {
	ctx, interruptSignal, stop := newSearchBatchSignalContext()
	defer stop()
	stop()
	select {
	case <-ctx.Done():
	default:
		t.Fatalf("context must be canceled after stop")
	}
	if got := interruptSignal(); got != 0 {
		t.Fatalf("signal getter = %v, want 0", got)
	}
}
