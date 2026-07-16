package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

func TestExecuteIndexEmbeddingProvidersListsComplianceProfiles(t *testing.T) {
	stdout := new(strings.Builder)
	code := Execute([]string{"--json", "index", "embedding-providers"}, stdout, ioDiscard{})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stdout.String(), "deterministic-hash") || !strings.Contains(stdout.String(), "textEgress") || !strings.Contains(stdout.String(), "http-embedding") || !strings.Contains(stdout.String(), "licenseNotes") || !strings.Contains(stdout.String(), "vectorIndexInvalidation") || !strings.Contains(stdout.String(), "retrievalBenchmarkCompatible") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteIndexQdrantHTTPEmbeddingRequiresConsentAndModelLock(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "private qdrant text"}}}}})
	t.Setenv("RFORGE_QDRANT_URL", "http://127.0.0.1:1")
	t.Setenv("RFORGE_EMBEDDING_URL", "http://127.0.0.1:2/embed")
	stderr := new(strings.Builder)
	code := Execute([]string{"--project", project, "index", "rebuild", "--backend", "qdrant"}, ioDiscard{}, stderr)
	if code == 0 || !strings.Contains(stderr.String(), "requires explicit consent") {
		t.Fatalf("expected consent failure code=%d stderr=%s", code, stderr.String())
	}
	t.Setenv("RFORGE_EMBEDDING_CONSENT", "1")
	stderr.Reset()
	code = Execute([]string{"--project", project, "index", "rebuild", "--backend", "qdrant"}, ioDiscard{}, stderr)
	if code == 0 || !strings.Contains(stderr.String(), "RFORGE_EMBEDDING_MODEL") {
		t.Fatalf("expected model lock failure code=%d stderr=%s", code, stderr.String())
	}
}

func TestExecuteIndexQdrantWritesVectorLockPrivacyAndReport(t *testing.T) {
	server := newQdrantIndexTestServer(t)
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	t.Setenv("RFORGE_QDRANT_PAYLOAD_PRIVACY", retrieval.PayloadPrivacyRedacted)
	t.Setenv("RFORGE_QDRANT_INVALIDATE", "1")
	t.Setenv("RFORGE_EMBEDDING_DIMENSIONS", "10")
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "private qdrant text"}}}}})
	code := Execute([]string{"--project", project, "index", "rebuild", "--backend", "qdrant"}, ioDiscard{}, ioDiscard{})
	if code != 0 {
		t.Fatalf("index code = %d", code)
	}
	var report retrieval.QdrantRebuildReport
	data, err := os.ReadFile(filepath.Join(project, "data", "qdrant.index-report.json"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Dimension != 10 || report.PayloadPrivacy != retrieval.PayloadPrivacyRedacted || !report.InvalidatedBeforeUpsert {
		t.Fatalf("report = %#v", report)
	}
	lock, err := os.ReadFile(filepath.Join(project, "data", "qdrant.vector.lock.json"))
	if err != nil {
		t.Fatalf("read vector lock: %v", err)
	}
	if !strings.Contains(string(lock), "deterministic-hash") || !strings.Contains(string(lock), "redacted-checksum") {
		t.Fatalf("lock = %s", string(lock))
	}
}

func TestExecuteIndexQdrantPreservesMetadataWhenRetrievalLockCannotBeWritten(t *testing.T) {
	var requests atomic.Int32
	server := newQdrantIndexTestServer(t, func() { requests.Add(1) })
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	t.Setenv("RFORGE_QDRANT_PAYLOAD_PRIVACY", retrieval.PayloadPrivacyRedacted)
	t.Setenv("RFORGE_QDRANT_INVALIDATE", "1")
	t.Setenv("RFORGE_EMBEDDING_DIMENSIONS", "10")
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "private qdrant text"}}}}})
	dataDir := filepath.Join(project, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	prior := map[string][]byte{
		filepath.Join(dataDir, "qdrant.index-report.json"): []byte("prior qdrant report\n"),
		filepath.Join(dataDir, "qdrant.vector.lock.json"):  []byte("prior qdrant lock\n"),
	}
	for path, data := range prior {
		if err := os.WriteFile(path, data, 0o640); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dataDir, "retrieval.lock.json"), 0o755); err != nil {
		t.Fatalf("block retrieval lock: %v", err)
	}
	code := Execute([]string{"--project", project, "index", "rebuild", "--backend", "qdrant"}, ioDiscard{}, ioDiscard{})
	if code != 1 {
		t.Fatalf("index code = %d, want 1", code)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("Qdrant received %d requests before metadata preflight failure", got)
	}
	for path, want := range prior {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved %s: %v", path, err)
		}
		if string(got) != string(want) {
			t.Fatalf("metadata changed after lock failure: %s = %q, want %q", path, got, want)
		}
	}
}

func TestExecuteIndexQdrantRejectsSymlinkedMetadataDirectoryBeforeRemoteRequest(t *testing.T) {
	var requests atomic.Int32
	server := newQdrantIndexTestServer(t, func() { requests.Add(1) })
	defer server.Close()
	t.Setenv("RFORGE_QDRANT_URL", server.URL)
	t.Setenv("RFORGE_QDRANT_PAYLOAD_PRIVACY", retrieval.PayloadPrivacyRedacted)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "parsed"), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(project, "parsed", "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Qdrant text"}}}}})
	outsideDir := t.TempDir()
	dataDir := filepath.Join(project, "data")
	if err := os.Symlink(outsideDir, dataDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr strings.Builder
	code := Execute([]string{"--json", "--project", project, "index", "rebuild", "--backend", "qdrant"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("index code = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"index_metadata_preflight_failed"`) {
		t.Fatalf("missing metadata preflight error: %s", stdout.String())
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("Qdrant received %d requests before metadata parent preflight failure", got)
	}
	entries, err := os.ReadDir(outsideDir)
	if err != nil {
		t.Fatalf("read outside directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("metadata written through symlinked directory: %v", entries)
	}
}

func newQdrantIndexTestServer(t *testing.T, onRequest ...func()) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(onRequest) > 0 {
			onRequest[0]()
		}
		switch r.URL.Path {
		case "/collections/researchforge_passages":
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/collections/researchforge_passages/points/delete":
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		case "/collections/researchforge_passages/points":
			body := readBodyForCLI(t, r)
			if strings.Contains(body, `"Text":`) || !strings.Contains(body, "TextChecksum") {
				t.Fatalf("payload not redacted: %s", body)
			}
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
}

func readBodyForCLI(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
