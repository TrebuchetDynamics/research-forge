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
	if err := os.WriteFile(out, []byte("prior queue\n"), 0o600); err != nil {
		t.Fatalf("write prior queue: %v", err)
	}

	code := Execute([]string{"screen", "queue", "--dir", dir, "--out", out}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat queue output: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("queue output mode = %o, want 600", got)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read topic directory: %v", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), ".rforge-stage-") || strings.Contains(file.Name(), ".rforge-backup-") {
			t.Fatalf("screen queue left transaction debris: %s", file.Name())
		}
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

func TestScreenDirQueueDoesNotWriteThroughSymlinkedOutput(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	outsidePath := filepath.Join(t.TempDir(), "outside.csv")
	outsideBefore := []byte("outside queue must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside queue: %v", err)
	}
	outPath := filepath.Join(dir, "queue.csv")
	if err := os.Symlink(outsidePath, outPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "queue", "--dir", dir, "--out", outPath}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen queue succeeded with a symlinked output: stdout=%s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside queue: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("screen queue wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(outPath)
	if err != nil {
		t.Fatalf("lstat queue output: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("screen queue replaced output symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestScreenDirQueueReportsMalformedResultsWithoutReplacingOutput(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	resultsPath := filepath.Join(dir, "results.jsonl")
	results, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read results fixture: %v", err)
	}
	malformed := append(append([]byte{}, results...), []byte("{not valid JSON}\n")...)
	if err := os.WriteFile(resultsPath, malformed, 0o600); err != nil {
		t.Fatalf("write malformed results: %v", err)
	}
	outPath := filepath.Join(dir, "queue.csv")
	outBefore := []byte("existing queue must remain unchanged\n")
	if err := os.WriteFile(outPath, outBefore, 0o640); err != nil {
		t.Fatalf("write existing queue: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "queue", "--dir", dir, "--out", outPath}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen queue succeeded with malformed results: stdout=%s", stdout.String())
	}
	if output := stdout.String() + stderr.String(); !strings.Contains(output, "line 3") {
		t.Fatalf("screen queue error does not identify malformed line: %s", output)
	}
	outAfter, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read existing queue after command: %v", err)
	}
	if !bytes.Equal(outAfter, outBefore) {
		t.Fatalf("screen queue replaced output after malformed input: got %q, want %q", outAfter, outBefore)
	}
}

func TestScreenDirQueueDoesNotReadThroughSymlinkedResults(t *testing.T) {
	dir := makeScreenDirTopicDir(t, nil)
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	line, err := json.Marshal(samplePapers[0])
	if err != nil {
		t.Fatalf("marshal outside result: %v", err)
	}
	outsideBefore := append(line, '\n')
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside results: %v", err)
	}
	resultsPath := filepath.Join(dir, "results.jsonl")
	if err := os.Remove(resultsPath); err != nil {
		t.Fatalf("remove results fixture: %v", err)
	}
	if err := os.Symlink(outsidePath, resultsPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	outPath := filepath.Join(dir, "queue.csv")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "queue", "--dir", dir, "--out", outPath}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen queue succeeded through symlinked results: stdout=%s", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("screen queue created output after rejecting symlinked results: err=%v", err)
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside results: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("screen queue changed outside results: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(resultsPath)
	if err != nil {
		t.Fatalf("lstat results symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("screen queue replaced results symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestScreenDirImportWritesScreeningJSONL(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	screeningPath := filepath.Join(dir, "screening.jsonl")
	if err := os.WriteFile(screeningPath, nil, 0o600); err != nil {
		t.Fatalf("create screening log: %v", err)
	}

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

	data, err := os.ReadFile(screeningPath)
	if err != nil {
		t.Fatalf("screening.jsonl not written: %v", err)
	}
	info, err := os.Stat(screeningPath)
	if err != nil {
		t.Fatalf("stat screening log: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("screening log mode = %o, want 600", got)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read topic directory: %v", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), ".rforge-stage-") || strings.Contains(file.Name(), ".rforge-backup-") {
			t.Fatalf("screen import left transaction debris: %s", file.Name())
		}
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in screening.jsonl, got %d\n%s", len(lines), string(data))
	}
	// Index by DOI here so field assertions remain independent of ordering checks below.
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

func TestScreenDirImportDoesNotWriteThroughSymlinkedScreeningLog(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	outsideBefore := []byte("{\"doi\":\"10.1/outside\",\"decision\":\"include\",\"stage\":\"title_abstract\",\"reviewer\":\"Private\",\"reason\":\"\",\"timestamp\":\"2026-06-30T00:00:00Z\"}\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside screening log: %v", err)
	}
	screeningPath := filepath.Join(dir, "screening.jsonl")
	if err := os.Symlink(outsidePath, screeningPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	csvPath := filepath.Join(dir, "queue.csv")
	csvData := "doi,arxiv_id,title,authors,year,abstract,source,decision,reason\n10.1/alpha,,Paper Alpha,A. Smith,2022,Abstract,openalex,exclude,revised\n"
	if err := os.WriteFile(csvPath, []byte(csvData), 0o600); err != nil {
		t.Fatalf("write import CSV: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"screen", "import", "--dir", dir, "--csv", csvPath, "--reviewer", "Tester"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen import succeeded with a symlinked screening log: stdout=%s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside screening log: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("screen import wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(screeningPath)
	if err != nil {
		t.Fatalf("lstat screening log: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("screen import replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestScreenDirImportWritesDeterministicKeyOrder(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	csvPath := filepath.Join(dir, "queue.csv")
	var csvData strings.Builder
	csvData.WriteString("doi,arxiv_id,title,authors,year,abstract,source,decision,reason\n")
	for i := 19; i >= 0; i-- {
		csvData.WriteString("10.1/")
		csvData.WriteByte(byte('a' + i))
		csvData.WriteString(",,Paper,A. Smith,2022,Abstract,openalex,include,\n")
	}
	if err := os.WriteFile(csvPath, []byte(csvData.String()), 0o600); err != nil {
		t.Fatalf("write import CSV: %v", err)
	}

	code := Execute([]string{"screen", "import", "--dir", dir, "--csv", csvPath, "--reviewer", "Tester"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("screen import exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(dir, "screening.jsonl"))
	if err != nil {
		t.Fatalf("read screening log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	got := make([]string, 0, len(lines))
	for _, line := range lines {
		var entry screenDirEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decode screening entry: %v", err)
		}
		got = append(got, entry.DOI)
	}
	want := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		want = append(want, "10.1/"+string(rune('a'+i)))
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("screening log order = %v, want %v", got, want)
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

func TestScreenDirProgressReportsMalformedResults(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte("{not valid JSON}\n"), 0o600); err != nil {
		t.Fatalf("write malformed results: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "progress", "--dir", dir}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen progress succeeded with malformed results: stdout=%s", stdout.String())
	}
}

func TestScreenDirProgressReportsIncompleteResult(t *testing.T) {
	dir := makeScreenDirTopicDir(t, nil)
	if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write incomplete result: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "progress", "--dir", dir}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen progress silently discarded an incomplete result: stdout=%s", stdout.String())
	}
}

func TestScreenDirProgressReportsMalformedScreeningEntry(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	path := filepath.Join(dir, "screening.jsonl")
	malformed := []byte("{\"doi\":\"10.1/alpha\",\"decision\":\"include\"}\n{not valid JSON}\n")
	if err := os.WriteFile(path, malformed, 0o600); err != nil {
		t.Fatalf("write malformed screening log: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "progress", "--dir", dir}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen progress succeeded with a malformed screening entry: stdout=%s", stdout.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read malformed screening log after command: %v", err)
	}
	if !bytes.Equal(got, malformed) {
		t.Fatalf("screen progress changed malformed log: got %q, want %q", got, malformed)
	}
}

func TestScreenDirProgressReportsEntryWithoutIdentifier(t *testing.T) {
	dir := makeScreenDirTopicDir(t, samplePapers)
	if err := os.WriteFile(filepath.Join(dir, "screening.jsonl"), []byte("{\"decision\":\"include\"}\n"), 0o600); err != nil {
		t.Fatalf("write incomplete screening log: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--json", "screen", "progress", "--dir", dir}, stdout, stderr)
	if code == 0 {
		t.Fatalf("screen progress silently accepted an entry without an identifier: stdout=%s", stdout.String())
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
