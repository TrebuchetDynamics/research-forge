package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestResearchParsePDFTextDoesNotWriteThroughSymlinkedOutput(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "paper.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 fixture"), 0o600); err != nil {
		t.Fatalf("write PDF fixture: %v", err)
	}
	fakePDFText := filepath.Join(t.TempDir(), "pdftotext")
	if err := os.WriteFile(fakePDFText, []byte("#!/bin/sh\nprintf 'Extracted research text with enough words to form a passage.\\n'\n"), 0o755); err != nil {
		t.Fatalf("write fake pdftotext: %v", err)
	}
	t.Setenv("RFORGE_PDFTOTEXT_CMD", fakePDFText)
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	outsideBefore := []byte("outside parsed document must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside parsed document: %v", err)
	}
	outPath := filepath.Join(dir, "parsed.json")
	if err := os.Symlink(outsidePath, outPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", dir, "research", "parse-pdftotext", "--paper", "paper-1", "--title", "Paper", "--pdf", pdfPath, "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("research parse-pdftotext succeeded with symlinked output: stdout=%s", stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside parsed document: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("parse-pdftotext wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(outPath)
	if err != nil {
		t.Fatalf("lstat parsed output: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("parse-pdftotext replaced output symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestResearchParsePDFTextPreservesModeAndCleansTransaction(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "paper.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 fixture"), 0o600); err != nil {
		t.Fatalf("write PDF fixture: %v", err)
	}
	fakePDFText := filepath.Join(t.TempDir(), "pdftotext")
	if err := os.WriteFile(fakePDFText, []byte("#!/bin/sh\nprintf 'Extracted research text with enough words to form a passage.\\n'\n"), 0o755); err != nil {
		t.Fatalf("write fake pdftotext: %v", err)
	}
	t.Setenv("RFORGE_PDFTOTEXT_CMD", fakePDFText)
	outPath := filepath.Join(dir, "parsed.json")
	if err := os.WriteFile(outPath, []byte("prior parsed output\n"), 0o600); err != nil {
		t.Fatalf("write prior parsed output: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", dir, "research", "parse-pdftotext", "--paper", "paper-1", "--title", "Paper", "--pdf", pdfPath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("parse-pdftotext exit code = %d, stderr=%s", code, stderr.String())
	}
	if content := readFileForCLITest(t, outPath); !strings.Contains(content, "pdftotext-sectionizer") {
		t.Fatalf("parsed output missing parser identity: %s", content)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat parsed output: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("parsed output mode = %o, want 600", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read parsed output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("parse-pdftotext left transaction debris: %s", entry.Name())
		}
	}
}

func TestResearchScreenQueueCommandWritesCSVAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	libraryPath := filepath.Join(dir, "library.json")
	writeJSONForCLITest(t, libraryPath, []library.PaperRecord{{
		Title:       "Cryptocurrency price prediction based on Xgboost, LightGBM and BNN",
		Abstract:    "Bitcoin cryptocurrency forecasting",
		Year:        2024,
		Identifiers: library.Identifiers{DOI: "10.example/lightgbm"},
		OpenAccess:  true,
	}})
	csvPath := filepath.Join(dir, "queue.csv")
	mdPath := filepath.Join(dir, "queue.md")
	if err := os.WriteFile(csvPath, []byte("prior CSV\n"), 0o600); err != nil {
		t.Fatalf("write prior CSV: %v", err)
	}
	if err := os.WriteFile(mdPath, []byte("prior Markdown\n"), 0o640); err != nil {
		t.Fatalf("write prior Markdown: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "research", "screen-queue", "--library", libraryPath, "--out", csvPath, "--markdown", mdPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(readFileForCLITest(t, csvPath), "direct_lightgbm_crypto") {
		t.Fatalf("csv missing screening group")
	}
	if !strings.Contains(readFileForCLITest(t, mdPath), "Structured screening queue") {
		t.Fatalf("markdown missing title")
	}
	for path, wantMode := range map[string]os.FileMode{csvPath: 0o600, mdPath: 0o640} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat output %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != wantMode {
			t.Fatalf("output %s mode = %o, want %o", path, got, wantMode)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("screening queue left transaction debris: %s", entry.Name())
		}
	}
}

func TestResearchScreenQueueRejectsSymlinkedMarkdownBeforeChangingCSV(t *testing.T) {
	dir := t.TempDir()
	libraryPath := filepath.Join(dir, "library.json")
	writeJSONForCLITest(t, libraryPath, []library.PaperRecord{{
		Title:       "Cryptocurrency price prediction based on Xgboost, LightGBM and BNN",
		Identifiers: library.Identifiers{DOI: "10.example/lightgbm"},
	}})
	csvPath := filepath.Join(dir, "queue.csv")
	csvBefore := []byte("existing screening queue must remain unchanged\n")
	if err := os.WriteFile(csvPath, csvBefore, 0o600); err != nil {
		t.Fatalf("write existing CSV: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.md")
	outsideBefore := []byte("outside markdown must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside markdown: %v", err)
	}
	markdownPath := filepath.Join(dir, "queue.md")
	if err := os.Symlink(outsidePath, markdownPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", dir, "research", "screen-queue", "--library", libraryPath, "--out", csvPath, "--markdown", markdownPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("research screen-queue succeeded with symlinked Markdown: stdout=%s", stdout.String())
	}
	csvAfter, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read existing CSV: %v", err)
	}
	if !bytes.Equal(csvAfter, csvBefore) {
		t.Fatalf("CSV changed before Markdown failure: got %q, want %q", csvAfter, csvBefore)
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside markdown: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("screen queue wrote through Markdown symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(markdownPath)
	if err != nil {
		t.Fatalf("lstat Markdown output: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("screen queue replaced Markdown symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestResearchLeakageAuditCommandAcceptsTextDir(t *testing.T) {
	dir := t.TempDir()
	textDir := filepath.Join(dir, "extracted-text")
	if err := os.Mkdir(textDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "Bitcoin order-flow validation\n\nWe forecast Bitcoin minute returns with order flow imbalance features. Training and testing use out-of-sample validation. Normalization is discussed."
	if err := os.WriteFile(filepath.Join(textDir, "paper-1.txt"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(dir, "audit.json")
	mdPath := filepath.Join(dir, "audit.md")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "research", "leakage-audit", "--text", textDir, "--out", jsonPath, "--markdown", mdPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(readFileForCLITest(t, jsonPath), "paper-1") || !strings.Contains(readFileForCLITest(t, mdPath), "Bitcoin order-flow validation") {
		t.Fatalf("text audit output missing paper evidence")
	}
}

func TestResearchLeakageAuditCommandWritesJSONAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.Mkdir(parsedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONForCLITest(t, filepath.Join(parsedDir, "paper.json"), parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		Title:         "Bitcoin order-flow validation",
		Sections:      []parsing.Section{{ID: "s1", Title: "Methods", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "We forecast Bitcoin minute returns with order flow imbalance features. Training and testing use out-of-sample validation. Normalization is discussed."}}}},
	})
	jsonPath := filepath.Join(dir, "audit.json")
	mdPath := filepath.Join(dir, "audit.md")
	if err := os.WriteFile(jsonPath, []byte("prior audit JSON\n"), 0o600); err != nil {
		t.Fatalf("write prior audit JSON: %v", err)
	}
	if err := os.WriteFile(mdPath, []byte("prior audit Markdown\n"), 0o640); err != nil {
		t.Fatalf("write prior audit Markdown: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "research", "leakage-audit", "--parsed", parsedDir, "--out", jsonPath, "--markdown", mdPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(readFileForCLITest(t, jsonPath), "global_preprocessing_possible") {
		t.Fatalf("audit json missing leakage flag")
	}
	if !strings.Contains(readFileForCLITest(t, mdPath), "Leakage-risk and feature-evidence triage") {
		t.Fatalf("audit markdown missing title")
	}
	for path, wantMode := range map[string]os.FileMode{jsonPath: 0o600, mdPath: 0o640} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat audit output %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != wantMode {
			t.Fatalf("audit output %s mode = %o, want %o", path, got, wantMode)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read audit output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".rforge-stage-") || strings.Contains(entry.Name(), ".rforge-backup-") {
			t.Fatalf("leakage audit left transaction debris: %s", entry.Name())
		}
	}
}

func TestResearchLeakageAuditRejectsSymlinkedMarkdownBeforeChangingJSON(t *testing.T) {
	dir := t.TempDir()
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.Mkdir(parsedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONForCLITest(t, filepath.Join(parsedDir, "paper.json"), parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		Title:         "Bitcoin order-flow validation",
		Sections:      []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Training and testing use out-of-sample validation."}}}},
	})
	jsonPath := filepath.Join(dir, "audit.json")
	jsonBefore := []byte("existing audit must remain unchanged\n")
	if err := os.WriteFile(jsonPath, jsonBefore, 0o600); err != nil {
		t.Fatalf("write existing audit: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.md")
	outsideBefore := []byte("outside audit markdown must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside Markdown: %v", err)
	}
	markdownPath := filepath.Join(dir, "audit.md")
	if err := os.Symlink(outsidePath, markdownPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", dir, "research", "leakage-audit", "--parsed", parsedDir, "--out", jsonPath, "--markdown", markdownPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("research leakage-audit succeeded with symlinked Markdown: stdout=%s", stdout.String())
	}
	jsonAfter, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read existing audit: %v", err)
	}
	if !bytes.Equal(jsonAfter, jsonBefore) {
		t.Fatalf("JSON changed before Markdown failure: got %q, want %q", jsonAfter, jsonBefore)
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside Markdown: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("leakage audit wrote through Markdown symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(markdownPath)
	if err != nil {
		t.Fatalf("lstat Markdown output: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("leakage audit replaced Markdown symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func writeJSONForCLITest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFileForCLITest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
