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

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteGraphPapersFetchesPDFsBeforeBuildingGraph(t *testing.T) {
	setFakePDFTools(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("%PDF graph fixture"))
	}))
	defer server.Close()
	dir := t.TempDir()
	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	record, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Graph paper", Identifiers: library.Identifiers{DOI: "10.1000/graph"}, URLs: []string{server.URL + "/paper.pdf"}, License: "cc-by", OpenAccess: true})
	if err := store.Create(record); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "--project", dir, "graph", "papers"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "documents", "text", "10-1000-graph.txt")); err != nil {
		t.Fatalf("missing fetched text: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "knowledge-graph.html")); err != nil {
		t.Fatalf("missing graph html: %v", err)
	}
}

func TestExecuteGraphPapersBuildsHTMLFromExtractedText(t *testing.T) {
	dir := t.TempDir()
	textDir := filepath.Join(dir, "documents", "text")
	if err := os.MkdirAll(textDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "Self-Supervised Learning from Images with a Joint-Embedding Predictive Architecture\n\nThis paper predicts semantic image targets with latent embeddings, masking, context encoders, and target encoders."
	if err := os.WriteFile(filepath.Join(textDir, "2301-08243.txt"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "graph", "papers"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	for _, path := range []string{filepath.Join(dir, "data", "knowledge-graph.json"), filepath.Join(dir, "data", "knowledge-graph.html"), filepath.Join(dir, "data", "knowledge-graph-report.md")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	html, err := os.ReadFile(filepath.Join(dir, "data", "knowledge-graph.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<svg", "<script", "requestAnimationFrame(tick)", "id=\"filter\"", "assignCommunities()", "Colors indicate detected communities", "Self-Supervised Learning", "semantic"} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("html missing %q: %s", want, html)
		}
	}
	report, err := os.ReadFile(filepath.Join(dir, "data", "knowledge-graph-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Paper knowledge graph report", "## Central nodes", "## Communities", "## Shortest paths"} {
		if !strings.Contains(string(report), want) {
			t.Fatalf("report missing %q: %s", want, report)
		}
	}
}

func TestExecuteGraphPapersDoesNotPartiallyReplaceOutputs(t *testing.T) {
	dir := t.TempDir()
	textDir := filepath.Join(dir, "documents", "text")
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(textDir, 0o755); err != nil {
		t.Fatalf("create text directory: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(textDir, "paper.txt"), []byte("Paper title\n\nGraph concepts and evidence."), 0o644); err != nil {
		t.Fatalf("write extracted text: %v", err)
	}
	jsonPath := filepath.Join(dataDir, "knowledge-graph.json")
	htmlPath := filepath.Join(dataDir, "knowledge-graph.html")
	reportPath := filepath.Join(dataDir, "knowledge-graph-report.md")
	jsonBefore := []byte("{\"sentinel\":\"prior graph\"}\n")
	htmlBefore := []byte("prior graph html\n")
	if err := os.WriteFile(jsonPath, jsonBefore, 0o600); err != nil {
		t.Fatalf("write prior graph JSON: %v", err)
	}
	if err := os.WriteFile(htmlPath, htmlBefore, 0o600); err != nil {
		t.Fatalf("write prior graph HTML: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-report.md")
	outsideBefore := []byte("outside report\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside report: %v", err)
	}
	if err := os.Symlink(outsidePath, reportPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", dir, "graph", "papers"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "graph_write_failed") {
		t.Fatalf("code=%d stdout=%s stderr=%s, want graph write failure", code, stdout.String(), stderr.String())
	}
	for path, before := range map[string][]byte{jsonPath: jsonBefore, htmlPath: htmlBefore, outsidePath: outsideBefore} {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved output %s: %v", path, err)
		}
		if !bytes.Equal(after, before) {
			t.Errorf("output %s changed:\n got: %s\nwant: %s", path, after, before)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat preserved output %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("output %s mode = %o, want 600", path, got)
		}
	}
	info, err := os.Lstat(reportPath)
	if err != nil {
		t.Fatalf("lstat report symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("report output is no longer a symlink: %v", info.Mode())
	}
}
