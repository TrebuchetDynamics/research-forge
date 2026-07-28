package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// writeSearchRawFile writes a tab-separated DOI list into <dir>/raw/search-<source>-001-q.txt
func writeSearchRawFile(t *testing.T, dir, source string, dois []string) {
	t.Helper()
	rawDir := filepath.Join(dir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatalf("mkdir raw: %v", err)
	}
	var sb strings.Builder
	for _, doi := range dois {
		sb.WriteString(doi + "\ttitle placeholder\n")
	}
	name := filepath.Join(rawDir, "search-"+source+"-001-q.txt")
	if err := os.WriteFile(name, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write raw file: %v", err)
	}
}

// writeSearchStatsFile writes a search-stats.txt in <dir> root (as search batch does).
func writeSearchStatsFile(t *testing.T, dir string) {
	t.Helper()
	content := "Search batch stats\nQueries: 3\nSources: openalex,arxiv\nRecords: 265\nDeduped records: 198\nFailures: 4\n"
	if err := os.WriteFile(filepath.Join(dir, "search-stats.txt"), []byte(content), 0o644); err != nil {
		t.Fatalf("write search-stats.txt: %v", err)
	}
}

// writeResultsJSONLStats writes a minimal results.jsonl with N entries.
func writeResultsJSONLStats(t *testing.T, dir string, n int) {
	t.Helper()
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(`{"Title":"Paper","Identifiers":{"DOI":"10.1/p` + strings.Repeat("0", 3) + `"}}` + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write results.jsonl: %v", err)
	}
}

func TestSearchStatsReadsFromRawSubdir(t *testing.T) {
	dir := t.TempDir()
	writeSearchRawFile(t, dir, "openalex", []string{"10.1/a", "10.1/b", "10.1/c"})
	writeSearchRawFile(t, dir, "arxiv", []string{"10.48550/arxiv.2301.00001"})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d; stdout = %s", code, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "openalex") {
		t.Errorf("missing 'openalex' in output:\n%s", out)
	}
	if !strings.Contains(out, "arxiv") {
		t.Errorf("missing 'arxiv' in output:\n%s", out)
	}
	if !strings.Contains(out, "4") {
		t.Errorf("expected total 4 unique DOIs in output:\n%s", out)
	}
}

func TestSearchStatsCountsOnlyNormalizedDOIs(t *testing.T) {
	dir := t.TempDir()
	rawDir := filepath.Join(dir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "10.1000/ABC\tPaper A\n\tPaper without DOI\nnot-a-doi\tBad identifier\n10.1000/abc\tDuplicate DOI\ncontinuation from a multiline title\n"
	if err := os.WriteFile(filepath.Join(rawDir, "search-openalex-001-q.txt"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "Total unique DOIs: 1") {
		t.Fatalf("non-DOIs or case variants inflated DOI count:\n%s", stdout.String())
	}
}

func TestWriteSearchBatchRawKeepsOneRecordPerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.txt")
	papers := []library.PaperRecord{
		{Title: "A title\nwith\ttabs", Identifiers: library.Identifiers{DOI: "10.1000/a"}},
		{Title: "Another title", Identifiers: library.Identifiers{DOI: "10.1000/b"}},
	}
	if err := writeSearchBatchRaw(path, papers); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(papers) {
		t.Fatalf("raw lines = %d, want %d:\n%s", len(lines), len(papers), data)
	}
	for _, line := range lines {
		if strings.Count(line, "\t") != 1 {
			t.Fatalf("raw line is not two-column TSV: %q", line)
		}
	}
}

func TestSearchStatsIgnoresSearchStatsTxt(t *testing.T) {
	dir := t.TempDir()
	writeSearchStatsFile(t, dir)
	// also put real raw data so stats has something to show
	writeSearchRawFile(t, dir, "openalex", []string{"10.1/a", "10.1/b"})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	// "stats" must NOT appear as a source name
	if strings.Contains(out, `"stats"`) || strings.Contains(out, "  stats ") {
		t.Errorf("search-stats.txt was treated as a source; output:\n%s", out)
	}
	// search-stats.txt must not inflate the two unique DOIs in the raw file.
	if !strings.Contains(out, "Total unique DOIs: 2") {
		t.Errorf("unexpected unique DOI count; output:\n%s", out)
	}
}

func TestSearchStatsShowsResultsJSONLCount(t *testing.T) {
	dir := t.TempDir()
	writeResultsJSONLStats(t, dir, 42)

	stdout := new(bytes.Buffer)
	code := Execute([]string{"search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "42") {
		t.Errorf("results.jsonl count (42) not shown in output:\n%s", out)
	}
	if !strings.Contains(out, "results.jsonl") {
		t.Errorf("output does not mention results.jsonl:\n%s", out)
	}
}

func TestSearchStatsJSONIncludesResultsJSONLCount(t *testing.T) {
	dir := t.TempDir()
	writeResultsJSONLStats(t, dir, 17)
	writeSearchRawFile(t, dir, "openalex", []string{"10.1/x"})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "search", "stats", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	count, ok := data["libraryRecords"]
	if !ok {
		t.Fatalf("JSON missing 'libraryRecords'; data = %v", data)
	}
	if int(count.(float64)) != 17 {
		t.Errorf("libraryRecords = %v, want 17", count)
	}
}
