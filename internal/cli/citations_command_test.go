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

const crossrefCitationMock = `{
	"status": "ok",
	"message": {
		"type": "journal-article",
		"title": ["Mock Journal Paper"],
		"author": [{"given": "Alice", "family": "Smith"}, {"given": "Bob", "family": "Jones"}],
		"published-print": {"date-parts": [[2023]]},
		"container-title": ["Journal of Testing"],
		"volume": "10",
		"issue": "2",
		"page": "100-120"
	}
}`

const arxivCitationMock = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Mock ArXiv Paper</title>
    <published>2023-06-01T00:00:00Z</published>
    <author><name>Carol Zhang</name></author>
    <author><name>David Lee</name></author>
  </entry>
</feed>`

func makeCitationsResearchDir(t *testing.T, topics map[string][]map[string]any, pdfSlugs map[string][]string) string {
	t.Helper()
	dir := t.TempDir()
	for topic, records := range topics {
		topicDir := filepath.Join(dir, topic)
		if err := os.MkdirAll(topicDir, 0o755); err != nil {
			t.Fatalf("mkdir topic: %v", err)
		}
		writeBatchResults(t, topicDir, records)
		if slugs, ok := pdfSlugs[topic]; ok {
			pdfDir := filepath.Join(topicDir, "pdfs")
			if err := os.MkdirAll(pdfDir, 0o755); err != nil {
				t.Fatalf("mkdir pdfs: %v", err)
			}
			for _, slug := range slugs {
				if err := os.WriteFile(filepath.Join(pdfDir, slug+".pdf"), []byte("%PDF-1.4 fake"), 0o644); err != nil {
					t.Fatalf("write fake pdf: %v", err)
				}
			}
		}
	}
	return dir
}

func TestCitationsBuildRequiresResearchDir(t *testing.T) {
	code := Execute([]string{"citations", "build"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestCitationsBuildRejectsUnreadableResearchDirectory(t *testing.T) {
	researchDir := filepath.Join(t.TempDir(), "missing")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "citations", "build", "--research-dir", researchDir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"citation_discovery_failed"`) {
		t.Fatalf("missing discovery error: %s", stdout.String())
	}
}

func TestCitationsBuildRejectsMalformedTopicResultsWithoutWritingOutput(t *testing.T) {
	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Paper", "Identifiers": map[string]any{"DOI": "10.1000/paper"}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_paper"}},
	)
	if err := os.WriteFile(filepath.Join(dir, "topic-a", "results.jsonl"), []byte(`{"Title":`), 0o644); err != nil {
		t.Fatalf("write malformed results: %v", err)
	}
	outPath := filepath.Join(dir, "CITATIONS.md")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "citations", "build", "--research-dir", dir, "--out", outPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"citation_discovery_failed"`) {
		t.Fatalf("missing discovery error: %s", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("citations output exists after discovery failure: %v", err)
	}
}

func TestCitationsBuildDoesNotWriteThroughSymlinkedOutput(t *testing.T) {
	researchDir := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside.md")
	outsideBefore := []byte("outside citations must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside citations: %v", err)
	}
	outPath := filepath.Join(researchDir, "CITATIONS.md")
	if err := os.Symlink(outsidePath, outPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"citations", "build", "--research-dir", researchDir, "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("citations build succeeded with symlinked output: stdout=%s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside citations: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("citations build wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(outPath)
	if err != nil {
		t.Fatalf("lstat citations output: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("citations build replaced output symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestCitationsBuildWritesCitationsFile(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(crossrefCitationMock)) //nolint:errcheck
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Test Paper", "Identifiers": map[string]any{"DOI": "10.1000/test", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_test"}},
	)
	outPath := filepath.Join(dir, "CITATIONS.md")
	if err := os.WriteFile(outPath, []byte("prior citations\n"), 0o600); err != nil {
		t.Fatalf("write prior citations: %v", err)
	}

	stdout := new(bytes.Buffer)
	code := Execute([]string{"citations", "build", "--research-dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("CITATIONS.md not written: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "[1]") {
		t.Errorf("CITATIONS.md missing [1]: %s", s)
	}
	if !strings.Contains(s, "Mock Journal Paper") {
		t.Errorf("CITATIONS.md missing paper title: %s", s)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat citations output: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("citations output mode = %o, want 600", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read citations directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("citations build left transaction debris: %s", entry.Name())
		}
	}
}

func TestCitationsBuildWritesMetadataWarningsForFallbackCitations(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusBadGateway)
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")
	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Local fallback title", "Identifiers": map[string]any{"DOI": "10.1000/fallback"}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_fallback"}},
	)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"citations", "build", "--research-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "CITATIONS.md"))
	if err != nil {
		t.Fatalf("read citations: %v", err)
	}
	for _, want := range []string{"Local fallback title", "## Metadata warnings", "crossref:10.1000/fallback", "verify"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("CITATIONS.md missing %q:\n%s", want, data)
		}
	}
	if !strings.Contains(stdout.String(), "1 metadata warning") {
		t.Fatalf("stdout did not report degraded metadata: %s", stdout.String())
	}
}

func TestCitationsBuildSkipsUndownloadedPapers(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(crossrefCitationMock)) //nolint:errcheck
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Downloaded Paper", "Identifiers": map[string]any{"DOI": "10.1000/down", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
				{"Title": "Not Downloaded", "Identifiers": map[string]any{"DOI": "10.1000/skip", "ArXivID": ""}, "OpenAccess": false, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_down"}},
	)

	Execute([]string{"citations", "build", "--research-dir", dir}, new(bytes.Buffer), new(bytes.Buffer))

	out, _ := os.ReadFile(filepath.Join(dir, "CITATIONS.md"))
	s := string(out)
	if strings.Contains(s, "Not Downloaded") {
		t.Errorf("CITATIONS.md should not include paper without PDF: %s", s)
	}
}

func TestCitationsBuildFormatsJournalCitation(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(crossrefCitationMock)) //nolint:errcheck
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Test Paper", "Identifiers": map[string]any{"DOI": "10.1000/test", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_test"}},
	)

	Execute([]string{"citations", "build", "--research-dir", dir}, new(bytes.Buffer), new(bytes.Buffer))

	out, _ := os.ReadFile(filepath.Join(dir, "CITATIONS.md"))
	s := string(out)
	if !strings.Contains(s, "A. Smith") {
		t.Errorf("expected author initials 'A. Smith', got: %s", s)
	}
	if !strings.Contains(s, "Journal of Testing") {
		t.Errorf("expected journal name, got: %s", s)
	}
	if !strings.Contains(s, "vol. 10") {
		t.Errorf("expected volume, got: %s", s)
	}
	if !strings.Contains(s, "10.1000/test") {
		t.Errorf("expected DOI link, got: %s", s)
	}
}

func TestCitationsBuildFormatsArXivCitation(t *testing.T) {
	arxiv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(arxivCitationMock)) //nolint:errcheck
	}))
	defer arxiv.Close()
	t.Setenv("RFORGE_ARXIV_ATOM_URL", arxiv.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "ArXiv Paper", "Identifiers": map[string]any{"DOI": "10.48550/arxiv.2301.99999", "ArXivID": "2301.99999"}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_48550_arxiv_2301_99999"}},
	)

	Execute([]string{"citations", "build", "--research-dir", dir}, new(bytes.Buffer), new(bytes.Buffer))

	out, _ := os.ReadFile(filepath.Join(dir, "CITATIONS.md"))
	s := string(out)
	if !strings.Contains(s, "arXiv") {
		t.Errorf("expected arXiv in venue: %s", s)
	}
	if !strings.Contains(s, "2301.99999") {
		t.Errorf("expected arXiv ID in citation: %s", s)
	}
	if !strings.Contains(s, "C. Zhang") {
		t.Errorf("expected author initial 'C. Zhang': %s", s)
	}
}

func TestCitationsBuildSortsByFirstAuthorLastName(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "zhao") {
			w.Write([]byte(`{"status":"ok","message":{"type":"journal-article","title":["Zhao Paper"],"author":[{"given":"X","family":"Zhao"}],"published-print":{"date-parts":[[2023]]},"container-title":["J"]}}`)) //nolint:errcheck
		} else {
			w.Write([]byte(`{"status":"ok","message":{"type":"journal-article","title":["Adams Paper"],"author":[{"given":"A","family":"Adams"}],"published-print":{"date-parts":[[2023]]},"container-title":["J"]}}`)) //nolint:errcheck
		}
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Zhao Paper", "Identifiers": map[string]any{"DOI": "10.1000/zhao", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
				{"Title": "Adams Paper", "Identifiers": map[string]any{"DOI": "10.1000/adams", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_zhao", "10_1000_adams"}},
	)

	Execute([]string{"citations", "build", "--research-dir", dir}, new(bytes.Buffer), new(bytes.Buffer))

	out, _ := os.ReadFile(filepath.Join(dir, "CITATIONS.md"))
	s := string(out)
	adamsPos := strings.Index(s, "Adams")
	zhaoPos := strings.Index(s, "Zhao")
	if adamsPos == -1 || zhaoPos == -1 {
		t.Fatalf("missing authors in output: %s", s)
	}
	if adamsPos > zhaoPos {
		t.Errorf("Adams should sort before Zhao (alphabetical by last name)")
	}
}

func TestCitationsBuildJSONOutput(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(crossrefCitationMock)) //nolint:errcheck
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Test", "Identifiers": map[string]any{"DOI": "10.1000/t", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_t"}},
	)

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "citations", "build", "--research-dir", dir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := env["data"].(map[string]any)
	if data["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", data["count"])
	}
	citations, _ := data["citations"].([]any)
	if len(citations) == 0 {
		t.Errorf("citations empty in JSON output")
	}
}

func TestCitationsBuildJSONReportsMetadataFetchFailure(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusBadGateway)
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")
	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "Local fallback title", "Identifiers": map[string]any{"DOI": "10.1000/fallback"}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_fallback"}},
	)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "citations", "build", "--research-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	var envelope struct {
		Data struct {
			Citations []struct {
				Title string `json:"title"`
			} `json:"citations"`
			Warnings []struct {
				Source     string `json:"source"`
				Identifier string `json:"identifier"`
				Message    string `json:"message"`
			} `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if len(envelope.Data.Citations) != 1 || envelope.Data.Citations[0].Title != "Local fallback title" {
		t.Fatalf("fallback citations = %#v", envelope.Data.Citations)
	}
	if len(envelope.Data.Warnings) != 1 {
		t.Fatalf("metadata warnings = %#v, want one", envelope.Data.Warnings)
	}
	warning := envelope.Data.Warnings[0]
	if warning.Source != "crossref" || warning.Identifier != "10.1000/fallback" || warning.Message == "" {
		t.Fatalf("metadata warning = %#v", warning)
	}
}

func TestCitationsBuildDefaultsOutToResearchDir(t *testing.T) {
	crossref := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(crossrefCitationMock)) //nolint:errcheck
	}))
	defer crossref.Close()
	t.Setenv("RFORGE_CROSSREF_API_URL", crossref.URL)
	t.Setenv("RFORGE_CITATIONS_FETCH_DELAY", "0")

	dir := makeCitationsResearchDir(t,
		map[string][]map[string]any{
			"topic-a": {
				{"Title": "P", "Identifiers": map[string]any{"DOI": "10.1000/p", "ArXivID": ""}, "OpenAccess": true, "URLs": []string{}},
			},
		},
		map[string][]string{"topic-a": {"10_1000_p"}},
	)

	Execute([]string{"citations", "build", "--research-dir", dir}, new(bytes.Buffer), new(bytes.Buffer))

	if _, err := os.Stat(filepath.Join(dir, "CITATIONS.md")); err != nil {
		t.Errorf("CITATIONS.md not written to research dir by default: %v", err)
	}
}
