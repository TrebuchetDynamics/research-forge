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

// writeBatchResults writes a results.jsonl file containing the given PaperRecord-like objects.
func writeBatchResults(t *testing.T, dir string, records []map[string]any) {
	t.Helper()
	var lines []byte
	for _, r := range records {
		line, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		lines = append(lines, line...)
		lines = append(lines, '\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), lines, 0o644); err != nil {
		t.Fatalf("write results.jsonl: %v", err)
	}
}

func TestOAFetchDownloadsPDFForArXivRecord(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 fake pdf content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".pdf") && !strings.Contains(r.URL.Path, "/pdf/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()

	t.Setenv("RFORGE_ARXIV_PDF_URL", server.URL)

	batchDir := t.TempDir()
	outDir := t.TempDir()
	writeBatchResults(t, batchDir, []map[string]any{
		{
			"Title":       "Attention Is All You Need",
			"Identifiers": map[string]any{"DOI": "10.48550/arxiv.1706.03762", "ArXivID": "1706.03762"},
			"OpenAccess":  true,
			"URLs":        []string{},
		},
	})

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"oa", "fetch", "--dir", batchDir, "--out", outDir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	pdfs, err := filepath.Glob(filepath.Join(outDir, "pdfs", "*.pdf"))
	if err != nil || len(pdfs) == 0 {
		t.Fatalf("no PDF written to %s/pdfs/: %v", outDir, err)
	}
	content, _ := os.ReadFile(pdfs[0])
	if !bytes.Equal(content, pdfBytes) {
		t.Errorf("PDF content = %q, want %q", content, pdfBytes)
	}
}

func TestOAFetchDownloadsPDFFromExplicitOAURL(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 oa pdf")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()

	batchDir := t.TempDir()
	outDir := t.TempDir()
	writeBatchResults(t, batchDir, []map[string]any{
		{
			"Title":       "LightGBM Survey",
			"Identifiers": map[string]any{"DOI": "10.1000/lightgbm"},
			"OpenAccess":  true,
			"URLs":        []string{server.URL + "/lightgbm.pdf"},
		},
	})

	code := Execute([]string{"oa", "fetch", "--dir", batchDir, "--out", outDir}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}

	pdfs, _ := filepath.Glob(filepath.Join(outDir, "pdfs", "*.pdf"))
	if len(pdfs) == 0 {
		t.Fatal("no PDF written for explicit OA URL record")
	}
}

func TestOAFetchSkipsClosedAccessRecords(t *testing.T) {
	batchDir := t.TempDir()
	outDir := t.TempDir()
	writeBatchResults(t, batchDir, []map[string]any{
		{
			"Title":       "Closed Access Paper",
			"Identifiers": map[string]any{"DOI": "10.1000/closed"},
			"OpenAccess":  false,
			"URLs":        []string{"https://example.com/paper"},
		},
	})

	code := Execute([]string{"oa", "fetch", "--dir", batchDir, "--out", outDir}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}

	pdfs, _ := filepath.Glob(filepath.Join(outDir, "pdfs", "*.pdf"))
	if len(pdfs) != 0 {
		t.Errorf("expected no PDFs for closed-access record, got %v", pdfs)
	}
}

func TestOAFetchWritesFetchReport(t *testing.T) {
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
		{"Title": "Paper A", "Identifiers": map[string]any{"DOI": "10.1/a", "ArXivID": "2001.00001"}, "OpenAccess": true, "URLs": []string{}},
		{"Title": "Paper B", "Identifiers": map[string]any{"DOI": "10.1/b"}, "OpenAccess": false, "URLs": []string{}},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"oa", "fetch", "--dir", batchDir, "--out", outDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}

	report, err := os.ReadFile(filepath.Join(outDir, "fetch-report.txt"))
	if err != nil {
		t.Fatalf("fetch-report.txt not written: %v", err)
	}
	if !strings.Contains(string(report), "1") {
		t.Errorf("fetch-report.txt = %q, want mention of 1 fetched", string(report))
	}
}

func TestOAFetchDefaultsOutToBatchDir(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 x")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()

	t.Setenv("RFORGE_ARXIV_PDF_URL", server.URL)

	batchDir := t.TempDir()
	writeBatchResults(t, batchDir, []map[string]any{
		{"Title": "Paper", "Identifiers": map[string]any{"DOI": "10.1/a", "ArXivID": "2001.00002"}, "OpenAccess": true, "URLs": []string{}},
	})

	code := Execute([]string{"oa", "fetch", "--dir", batchDir}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}

	pdfs, _ := filepath.Glob(filepath.Join(batchDir, "pdfs", "*.pdf"))
	if len(pdfs) == 0 {
		t.Fatal("expected PDF written to <dir>/pdfs/ when --out is omitted")
	}
}

func TestOAFetchRequiresDir(t *testing.T) {
	code := Execute([]string{"oa", "fetch"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestOAFetchJSONOutput(t *testing.T) {
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
		{"Title": "P", "Identifiers": map[string]any{"DOI": "10.1/p", "ArXivID": "2001.00003"}, "OpenAccess": true, "URLs": []string{}},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "oa", "fetch", "--dir", batchDir, "--out", outDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if _, ok := data["fetched"]; !ok {
		t.Errorf("JSON data missing 'fetched' key: %v", data)
	}
}
