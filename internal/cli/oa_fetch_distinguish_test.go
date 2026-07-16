package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOAFetchJSONDistinguishesOAUnavailableFromFailures verifies that oa fetch
// reports records with no open-access copy as oa_unavailable (expected, info),
// not as failures. Real download failures stay in the failures count. This
// stops agents from recording "No copyrighted full text acquired" as an error
// when a paper simply has no legal OA copy.
func TestOAFetchJSONDistinguishesOAUnavailableFromFailures(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 x")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()
	t.Setenv("RFORGE_ARXIV_PDF_URL", server.URL)

	batchDir := t.TempDir()
	outDir := t.TempDir()
	writeBatchResults(t, batchDir, []map[string]any{
		{"Title": "ArXiv Paper", "Identifiers": map[string]any{"DOI": "10.1/a", "ArXivID": "2001.00001"}, "OpenAccess": true, "URLs": []string{}},
		{"Title": "Closed Paper", "Identifiers": map[string]any{"DOI": "10.1/b"}, "OpenAccess": false, "URLs": []string{}},
		{"Title": "No URL Paper", "Identifiers": map[string]any{"DOI": "10.1/c"}, "OpenAccess": true, "URLs": []string{"https://example.com/page"}},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "oa", "fetch", "--dir", batchDir, "--out", outDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	// oa_unavailable must report the 2 records with no OA PDF URL (closed + no-.pdf-url).
	if int(data["oa_unavailable"].(float64)) != 2 {
		t.Fatalf("oa_unavailable = %v, want 2 (closed access + no .pdf URL)", data["oa_unavailable"])
	}
	// failures must be 0 — no download was attempted and failed.
	if int(data["failures"].(float64)) != 0 {
		t.Fatalf("failures = %v, want 0 (no download failures, only missing OA copies)", data["failures"])
	}
	if int(data["fetched"].(float64)) != 1 {
		t.Fatalf("fetched = %v, want 1", data["fetched"])
	}
}

func TestOAFetchReportExplainsSkippedAsExpected(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 x")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()
	t.Setenv("RFORGE_ARXIV_PDF_URL", server.URL)

	batchDir := t.TempDir()
	outDir := t.TempDir()
	writeBatchResults(t, batchDir, []map[string]any{
		{"Title": "ArXiv Paper", "Identifiers": map[string]any{"DOI": "10.1/a", "ArXivID": "2001.00001"}, "OpenAccess": true, "URLs": []string{}},
		{"Title": "Closed Paper", "Identifiers": map[string]any{"DOI": "10.1/b"}, "OpenAccess": false, "URLs": []string{}},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"oa", "fetch", "--dir", batchDir, "--out", outDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	report, err := os.ReadFile(filepath.Join(outDir, "fetch-report.txt"))
	if err != nil {
		t.Fatalf("read fetch-report.txt: %v", err)
	}
	r := string(report)
	// The report must label skipped as "no open-access copy" so agents don't
	// record expected missing-OA as errors in provenance.
	if !strings.Contains(strings.ToLower(r), "no open-access") {
		t.Fatalf("fetch-report.txt must explain skipped as no open-access copy (expected): %q", r)
	}
	if strings.Contains(strings.ToLower(r), "error") {
		t.Fatalf("fetch-report.txt must not mention 'error' when there were only missing OA copies: %q", r)
	}
}
