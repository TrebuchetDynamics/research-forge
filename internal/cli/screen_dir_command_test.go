package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeScreenDirTopicDir creates a topic dir with results.jsonl containing the given papers.
func makeScreenDirTopicDir(t *testing.T, papers []map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	var sb strings.Builder
	for _, p := range papers {
		line, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal paper: %v", err)
		}
		sb.Write(line)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write results.jsonl: %v", err)
	}
	return dir
}

// writeScreeningJSONL writes a screening.jsonl with pre-existing decisions.
func writeScreeningJSONL(t *testing.T, dir string, entries []map[string]any) {
	t.Helper()
	var sb strings.Builder
	for _, e := range entries {
		line, _ := json.Marshal(e)
		sb.Write(line)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "screening.jsonl"), []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write screening.jsonl: %v", err)
	}
}

// readCSV parses a CSV file and returns header + rows.
func readCSV(t *testing.T, path string) (header []string, rows [][]string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open CSV %s: %v", path, err)
	}
	defer f.Close()
	all, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse CSV: %v", err)
	}
	if len(all) == 0 {
		return nil, nil
	}
	return all[0], all[1:]
}

var samplePapers = []map[string]any{
	{
		"Title":       "Paper Alpha",
		"Identifiers": map[string]any{"DOI": "10.1/alpha"},
		"Authors":     []map[string]any{{"Name": "A. Smith"}},
		"Year":        2022,
		"Abstract":    "Alpha abstract text.",
		"SourceRefs":  []map[string]any{{"Source": "openalex"}},
	},
	{
		"Title":       "Paper Beta",
		"Identifiers": map[string]any{"DOI": "10.1/beta"},
		"Authors":     []map[string]any{{"Name": "B. Jones"}},
		"Year":        2023,
		"Abstract":    "Beta abstract text.",
		"SourceRefs":  []map[string]any{{"Source": "arxiv"}},
	},
}

func TestScreenDirQueueEmitsCSVWithAllColumns(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	out := filepath.Join(dir, "queue.csv")

	code := Execute([]string{"screen", "queue", "--dir", dir, "--out", out}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	header, rows := readCSV(t, out)

	wantCols := []string{"doi", "arxiv_id", "title", "authors", "year", "abstract", "source", "decision", "reason"}
	for _, col := range wantCols {
		found := false
		for _, h := range header {
			if h == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CSV missing column %q; header = %v", col, header)
		}
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 data rows, got %d", len(rows))
	}
}

func TestScreenDirQueueIncludesAbstractText(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	out := filepath.Join(dir, "queue.csv")

	Execute([]string{"screen", "queue", "--dir", dir, "--out", out}, new(bytes.Buffer), new(bytes.Buffer))
	header, rows := readCSV(t, out)

	abstractIdx := -1
	for i, h := range header {
		if h == "abstract" {
			abstractIdx = i
			break
		}
	}
	if abstractIdx < 0 {
		t.Fatal("no 'abstract' column")
	}
	found := false
	for _, row := range rows {
		if strings.Contains(row[abstractIdx], "Alpha abstract") || strings.Contains(row[abstractIdx], "Beta abstract") {
			found = true
		}
	}
	if !found {
		t.Error("abstract text not present in queue CSV rows")
	}
}

func TestScreenDirQueueSkipsAlreadyDecidedRecords(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	// Alpha is already decided — only Beta should appear in the queue
	writeScreeningJSONL(t, dir, []map[string]any{
		{"doi": "10.1/alpha", "arxiv_id": "", "decision": "include", "stage": "title_abstract", "reviewer": "Tester", "reason": "", "timestamp": "2026-06-30T00:00:00Z"},
	})
	out := filepath.Join(dir, "queue.csv")

	Execute([]string{"screen", "queue", "--dir", dir, "--out", out}, new(bytes.Buffer), new(bytes.Buffer))
	_, rows := readCSV(t, out)

	if len(rows) != 1 {
		t.Fatalf("expected 1 pending row (Beta only), got %d rows", len(rows))
	}
}

func TestScreenDirQueueDecisionAndReasonColumnsBlank(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	out := filepath.Join(dir, "queue.csv")

	Execute([]string{"screen", "queue", "--dir", dir, "--out", out}, new(bytes.Buffer), new(bytes.Buffer))
	header, rows := readCSV(t, out)

	decisionIdx := -1
	reasonIdx := -1
	for i, h := range header {
		if h == "decision" {
			decisionIdx = i
		}
		if h == "reason" {
			reasonIdx = i
		}
	}
	for _, row := range rows {
		if row[decisionIdx] != "" {
			t.Errorf("decision column should be blank for pending rows, got %q", row[decisionIdx])
		}
		if row[reasonIdx] != "" {
			t.Errorf("reason column should be blank for pending rows, got %q", row[reasonIdx])
		}
	}
}

func TestScreenDirImportWritesScreeningJSONL(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)

	// Write a reviewer-filled CSV
	csvPath := filepath.Join(dir, "queue.csv")
	f, _ := os.Create(csvPath)
	w := csv.NewWriter(f)
	_ = w.Write([]string{"doi", "arxiv_id", "title", "authors", "year", "abstract", "source", "decision", "reason"})
	_ = w.Write([]string{"10.1/alpha", "", "Paper Alpha", "A. Smith", "2022", "Abstract", "openalex", "include", ""})
	_ = w.Write([]string{"10.1/beta", "", "Paper Beta", "B. Jones", "2023", "Abstract", "arxiv", "exclude", "out of scope"})
	w.Flush()
	f.Close()

	stdout := new(bytes.Buffer)
	code := Execute([]string{"screen", "import", "--dir", dir, "--csv", csvPath, "--reviewer", "Tester"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d; stdout = %s", code, stdout.String())
	}

	data, err := os.ReadFile(filepath.Join(dir, "screening.jsonl"))
	if err != nil {
		t.Fatalf("screening.jsonl not written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in screening.jsonl, got %d\n%s", len(lines), string(data))
	}
	// Map iteration is non-deterministic; find alpha's entry by DOI rather than by line index.
	byDOI := map[string]map[string]any{}
	for _, line := range lines {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("line not JSON: %v\n%s", err, line)
		}
		if doi, ok := e["doi"].(string); ok && doi != "" {
			byDOI[doi] = e
		}
	}
	alpha := byDOI["10.1/alpha"]
	if alpha == nil {
		t.Fatalf("no entry for 10.1/alpha in screening.jsonl")
	}
	if alpha["decision"] != "include" {
		t.Errorf("alpha decision = %v, want include", alpha["decision"])
	}
	if alpha["reviewer"] != "Tester" {
		t.Errorf("alpha reviewer = %v, want Tester", alpha["reviewer"])
	}
}

func TestScreenDirImportSkipsBlankDecisionRows(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	csvPath := filepath.Join(dir, "queue.csv")
	f, _ := os.Create(csvPath)
	w := csv.NewWriter(f)
	_ = w.Write([]string{"doi", "arxiv_id", "title", "authors", "year", "abstract", "source", "decision", "reason"})
	_ = w.Write([]string{"10.1/alpha", "", "Paper Alpha", "A. Smith", "2022", "Abstract", "openalex", "include", ""})
	_ = w.Write([]string{"10.1/beta", "", "Paper Beta", "B. Jones", "2023", "Abstract", "arxiv", "", ""}) // blank — not yet decided
	w.Flush()
	f.Close()

	Execute([]string{"screen", "import", "--dir", dir, "--csv", csvPath, "--reviewer", "Tester"}, new(bytes.Buffer), new(bytes.Buffer))

	data, _ := os.ReadFile(filepath.Join(dir, "screening.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Only alpha (decided); beta (blank) must be skipped
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (only alpha), got %d\n%s", len(lines), string(data))
	}
}

func TestScreenDirImportLastWriteWins(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	writeScreeningJSONL(t, dir, []map[string]any{
		{"doi": "10.1/alpha", "arxiv_id": "", "decision": "include", "stage": "title_abstract", "reviewer": "First", "reason": "", "timestamp": "2026-06-30T00:00:00Z"},
	})

	// Re-import alpha with a different decision
	csvPath := filepath.Join(dir, "queue.csv")
	f, _ := os.Create(csvPath)
	w := csv.NewWriter(f)
	_ = w.Write([]string{"doi", "arxiv_id", "title", "authors", "year", "abstract", "source", "decision", "reason"})
	_ = w.Write([]string{"10.1/alpha", "", "Paper Alpha", "A. Smith", "2022", "Abstract", "openalex", "exclude", "revised"})
	w.Flush()
	f.Close()

	Execute([]string{"screen", "import", "--dir", dir, "--csv", csvPath, "--reviewer", "Second"}, new(bytes.Buffer), new(bytes.Buffer))

	data, _ := os.ReadFile(filepath.Join(dir, "screening.jsonl"))
	// Last entry for alpha should be exclude
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var lastEntry map[string]any
	_ = json.Unmarshal([]byte(lines[len(lines)-1]), &lastEntry)
	if lastEntry["decision"] != "exclude" {
		t.Errorf("expected last decision=exclude after re-import, got %v", lastEntry["decision"])
	}
}

func TestScreenDirProgressShowsCounts(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	writeScreeningJSONL(t, dir, []map[string]any{
		{"doi": "10.1/alpha", "arxiv_id": "", "decision": "include", "stage": "title_abstract", "reviewer": "T", "reason": "", "timestamp": "2026-06-30T00:00:00Z"},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"screen", "progress", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "include") {
		t.Errorf("progress missing 'include' count: %s", out)
	}
	if !strings.Contains(out, "pending") {
		t.Errorf("progress missing 'pending' count: %s", out)
	}
	// 2 total papers, 1 decided → 1 pending
	if !strings.Contains(out, "1") {
		t.Errorf("progress should show count 1 somewhere: %s", out)
	}
}

func TestScreenDirProgressJSONOutput(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	writeScreeningJSONL(t, dir, []map[string]any{
		{"doi": "10.1/alpha", "decision": "exclude", "stage": "title_abstract", "reviewer": "T", "reason": "off-topic", "timestamp": "2026-06-30T00:00:00Z"},
		{"doi": "10.1/beta", "decision": "include", "stage": "title_abstract", "reviewer": "T", "reason": "", "timestamp": "2026-06-30T00:00:00Z"},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "screen", "progress", "--dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["included"].(float64) != 1 || data["excluded"].(float64) != 1 || data["pending"].(float64) != 0 {
		t.Errorf("unexpected counts: %v", data)
	}
}
