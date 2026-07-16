package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeVaultResearchDir creates a research dir with two topic subdirs each
// containing a results.jsonl with a few papers. Paper "shared" appears in both.
func makeVaultResearchDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	papers := map[string][]map[string]any{
		"topic-alpha": {
			{
				"Title":       "Alpha Paper One",
				"Identifiers": map[string]any{"DOI": "10.1/alpha1"},
				"Authors":     []map[string]any{{"Given": "Alice", "Family": "Smith"}},
				"Year":        2022,
				"Abstract":    "Alpha one abstract.",
				"SourceRefs":  []map[string]any{{"Source": "openalex"}},
			},
			{
				"Title":       "Shared Cross-Topic Paper",
				"Identifiers": map[string]any{"DOI": "10.1/shared"},
				"Authors":     []map[string]any{{"Given": "Bob", "Family": "Jones"}},
				"Year":        2021,
				"Abstract":    "Shared abstract.",
				"SourceRefs":  []map[string]any{{"Source": "arxiv"}},
			},
		},
		"topic-beta": {
			{
				"Title":       "Beta Paper One",
				"Identifiers": map[string]any{"DOI": "10.1/beta1"},
				"Authors":     []map[string]any{{"Given": "Carol", "Family": "Lee"}},
				"Year":        2023,
				"Abstract":    "Beta one abstract.",
				"SourceRefs":  []map[string]any{{"Source": "semantic-scholar"}},
			},
			{
				"Title":       "Shared Cross-Topic Paper",
				"Identifiers": map[string]any{"DOI": "10.1/shared"},
				"Authors":     []map[string]any{{"Given": "Bob", "Family": "Jones"}},
				"Year":        2021,
				"Abstract":    "Shared abstract.",
				"SourceRefs":  []map[string]any{{"Source": "arxiv"}},
			},
		},
	}

	for topic, recs := range papers {
		topicDir := filepath.Join(dir, topic)
		if err := os.MkdirAll(topicDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", topicDir, err)
		}
		var sb strings.Builder
		for _, p := range recs {
			line, _ := json.Marshal(p)
			sb.Write(line)
			sb.WriteByte('\n')
		}
		if err := os.WriteFile(filepath.Join(topicDir, "results.jsonl"), []byte(sb.String()), 0o644); err != nil {
			t.Fatalf("write results.jsonl: %v", err)
		}
	}
	return dir
}

func runVaultBuild(t *testing.T, researchDir, outDir string) (int, string, string) {
	t.Helper()
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"vault", "build", "--research-dir", researchDir, "--out", outDir}, stdout, stderr)
	return code, stdout.String(), stderr.String()
}

// ── basic output structure ──────────────────────────────────────────────────

func TestVaultBuildCreatesOutputDir(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	code, _, stderr := runVaultBuild(t, researchDir, outDir)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr = %s", code, stderr)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Fatalf("vault output dir not created: %v", err)
	}
}

func TestVaultBuildCreatesPaperNotes(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	papersDir := filepath.Join(outDir, "papers")
	entries, err := os.ReadDir(papersDir)
	if err != nil {
		t.Fatalf("papers/ dir not created: %v", err)
	}
	// 3 unique papers (alpha1, beta1, shared)
	if len(entries) != 3 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected 3 paper notes, got %d: %v", len(entries), names)
	}
}

func TestVaultBuildPaperNoteHasYAMLFrontmatter(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	// Find the note for "Alpha Paper One"
	note := findPaperNote(t, outDir, "Alpha Paper One")
	if !strings.HasPrefix(note, "---\n") {
		t.Fatalf("paper note does not start with YAML frontmatter:\n%s", note[:minInt(200, len(note))])
	}
	if !strings.Contains(note, "title:") {
		t.Error("frontmatter missing 'title'")
	}
	if !strings.Contains(note, "year:") {
		t.Error("frontmatter missing 'year'")
	}
	if !strings.Contains(note, "doi:") {
		t.Error("frontmatter missing 'doi'")
	}
	if !strings.Contains(note, "topics:") {
		t.Error("frontmatter missing 'topics'")
	}
	if !strings.Contains(note, "authors:") {
		t.Error("frontmatter missing 'authors'")
	}
}

func TestVaultBuildPaperNoteHasTopicWikilinks(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	// Alpha Paper One is only in topic-alpha
	note := findPaperNote(t, outDir, "Alpha Paper One")
	if !strings.Contains(note, "[[topic-alpha]]") {
		t.Errorf("paper note missing [[topic-alpha]] wikilink:\n%s", note)
	}
}

func TestVaultBuildSharedPaperLinksAllTopics(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	note := findPaperNote(t, outDir, "Shared Cross-Topic Paper")
	if !strings.Contains(note, "[[topic-alpha]]") {
		t.Errorf("shared paper note missing [[topic-alpha]]:\n%s", note)
	}
	if !strings.Contains(note, "[[topic-beta]]") {
		t.Errorf("shared paper note missing [[topic-beta]]:\n%s", note)
	}
}

func TestVaultBuildPaperNoteIncludesAbstract(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	note := findPaperNote(t, outDir, "Alpha Paper One")
	if !strings.Contains(note, "Alpha one abstract.") {
		t.Errorf("paper note missing abstract text:\n%s", note)
	}
}

// ── topic index notes ───────────────────────────────────────────────────────

func TestVaultBuildCreatesTopicIndexNotes(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	for _, topic := range []string{"topic-alpha", "topic-beta"} {
		path := filepath.Join(outDir, topic+".md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("topic index note %s not created: %v", topic+".md", err)
		}
	}
}

func TestVaultBuildTopicNoteLinksToAllPapers(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	alphaNote, err := os.ReadFile(filepath.Join(outDir, "topic-alpha.md"))
	if err != nil {
		t.Fatalf("topic-alpha.md not found: %v", err)
	}
	content := string(alphaNote)

	// Should link to both alpha1 and shared papers (via [[wikilink]])
	if !strings.Contains(content, "Alpha Paper One") {
		t.Errorf("topic-alpha.md missing link to Alpha Paper One:\n%s", content)
	}
	if !strings.Contains(content, "Shared Cross-Topic Paper") {
		t.Errorf("topic-alpha.md missing link to Shared Cross-Topic Paper:\n%s", content)
	}
}

// ── main index ──────────────────────────────────────────────────────────────

func TestVaultBuildCreatesMainIndex(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	indexPath := filepath.Join(outDir, "index.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index.md not created: %v", err)
	}
	content, _ := os.ReadFile(indexPath)
	if !strings.Contains(string(content), "topic-alpha") {
		t.Errorf("index.md missing topic-alpha:\n%s", content)
	}
	if !strings.Contains(string(content), "topic-beta") {
		t.Errorf("index.md missing topic-beta:\n%s", content)
	}
}

func TestVaultBuildMainIndexHighlightsCrossTopicPapers(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	content, _ := os.ReadFile(filepath.Join(outDir, "index.md"))
	// Shared paper appears in 2 topics — should be called out in the main index
	if !strings.Contains(string(content), "Shared Cross-Topic Paper") {
		t.Errorf("index.md missing cross-topic paper mention:\n%s", content)
	}
}

// ── deduplication ───────────────────────────────────────────────────────────

func TestVaultBuildDeduplicatesSharedPapers(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	runVaultBuild(t, researchDir, outDir)

	// shared paper appears in both topics but should produce exactly 1 note
	papersDir := filepath.Join(outDir, "papers")
	entries, _ := os.ReadDir(papersDir)
	sharedCount := 0
	for _, e := range entries {
		content, _ := os.ReadFile(filepath.Join(papersDir, e.Name()))
		if strings.Contains(string(content), "Shared Cross-Topic Paper") {
			sharedCount++
		}
	}
	if sharedCount != 1 {
		t.Errorf("expected exactly 1 note for shared paper, got %d", sharedCount)
	}
}

// ── error handling ──────────────────────────────────────────────────────────

func TestVaultBuildUsageError(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"vault", "build"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2 for missing args, got %d", code)
	}
}

func TestVaultBuildFailurePreservesExistingVault(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := t.TempDir()
	priorFiles := map[string][]byte{
		filepath.Join(outDir, "papers", "alpha-paper-one.md"): []byte("prior paper note\n"),
		filepath.Join(outDir, "topic-alpha.md"):               []byte("prior topic note\n"),
	}
	for path, content := range priorFiles {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create prior vault directory: %v", err)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("write prior vault file %s: %v", path, err)
		}
	}
	if err := os.Mkdir(filepath.Join(outDir, "index.md"), 0o755); err != nil {
		t.Fatalf("create failing index target: %v", err)
	}

	code, _, stderr := runVaultBuild(t, researchDir, outDir)
	if code != 1 {
		t.Fatalf("exit code = %d; stderr = %s, want write failure", code, stderr)
	}
	for path, want := range priorFiles {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved vault file %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("vault file %s changed after failed build:\n got: %s\nwant: %s", path, got, want)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat preserved vault file %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("vault file %s mode = %o, want 600", path, got)
		}
	}
}

// ── JSON output ─────────────────────────────────────────────────────────────

func TestVaultBuildJSONOutput(t *testing.T) {
	researchDir := makeVaultResearchDir(t)
	outDir := filepath.Join(t.TempDir(), "vault")

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"--json", "vault", "build", "--research-dir", researchDir, "--out", outDir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr = %s", code, stderr.String())
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := env["data"].(map[string]any)
	if data["papers"].(float64) == 0 {
		t.Errorf("JSON output missing papers count: %v", data)
	}
	if data["topics"].(float64) == 0 {
		t.Errorf("JSON output missing topics count: %v", data)
	}
	if data["vault"].(string) == "" {
		t.Errorf("JSON output missing vault path: %v", data)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// findPaperNote scans the papers/ subdir for a note containing titleSubstr.
func findPaperNote(t *testing.T, outDir, titleSubstr string) string {
	t.Helper()
	papersDir := filepath.Join(outDir, "papers")
	entries, err := os.ReadDir(papersDir)
	if err != nil {
		t.Fatalf("papers/ dir not found: %v", err)
	}
	for _, e := range entries {
		content, _ := os.ReadFile(filepath.Join(papersDir, e.Name()))
		if strings.Contains(string(content), titleSubstr) {
			return string(content)
		}
	}
	t.Fatalf("no paper note found containing %q", titleSubstr)
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
