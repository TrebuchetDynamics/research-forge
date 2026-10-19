package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteLitWritesSynthesis(t *testing.T) {
	root := t.TempDir()
	subA := filepath.Join(root, "topic-a")
	subB := filepath.Join(root, "topic-b")

	writeFixtureSearchDir(t, subA, "scaling laws", []map[string]any{
		{"title": "Scaling Laws for Neural Language Models", "year": 2020, "abstract": "We study scaling laws for language models.", "identifiers": map[string]any{"DOI": "10.1000/scaling"}},
		{"title": "A Survey of Scaling Methods", "year": 2022, "abstract": "Survey of scaling methods in deep learning.", "identifiers": map[string]any{"ArXivID": "2201.00001"}},
	})
	writeFixtureSearchDir(t, subB, "language models", []map[string]any{
		{"title": "Scaling Laws for Neural Language Models", "year": 2020, "abstract": "We study scaling laws for language models.", "identifiers": map[string]any{"DOI": "10.1000/scaling"}},
		{"title": "Large Language Models: A Survey", "year": 2023, "abstract": "A survey of large language models.", "identifiers": map[string]any{"DOI": "10.1000/llm-survey"}},
	})

	var out, errBuf bytes.Buffer
	code := executeLit([]string{"--research-dir", root}, &out, &errBuf, globalOptions{})
	if code != 0 {
		t.Fatalf("executeLit exit %d: %s", code, errBuf.String())
	}

	assertEngineReceipt(t, root, "LIT.md")
	// Check output paths
	litPath := filepath.Join(root, "LIT.md")
	provPath := filepath.Join(root, "LIT.provenance.json")
	if _, err := os.Stat(litPath); err != nil {
		t.Fatalf("LIT.md not written: %v", err)
	}
	if _, err := os.Stat(provPath); err != nil {
		t.Fatalf("LIT.provenance.json not written: %v", err)
	}
	receipt, err := os.ReadFile(provPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(receipt, []byte(root)) {
		t.Fatalf("exportable receipt exposes absolute research path: %s", receipt)
	}

	lit, _ := os.ReadFile(litPath)
	got := string(lit)

	for _, want := range []string{
		"# Literature Synthesis",
		"Generated from **2 topics**",
		"**3 unique papers**",
		"topic-a",
		"topic-b",
		"## Consensus Themes",
		"## Cross-Topic Papers",
		"Scaling Laws for Neural Language Models",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("LIT.md missing %q", want)
		}
	}

	// Cross-topic: Shared DOI paper
	if !strings.Contains(got, "**2 topics**") {
		t.Errorf("expected cross-topic marker, got:\n%s", got)
	}
}

func TestExecuteLitEmptyDir(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := executeLit([]string{"--research-dir", "/tmp/nonexistent"}, &out, &errBuf, globalOptions{})
	if code == 0 {
		t.Fatal("expected non-zero exit for nonexistent dir")
	}
}

func TestExecuteLitJSONOutput(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "topic")
	writeFixtureSearchDir(t, sub, "test", []map[string]any{
		{"title": "Test Paper", "year": 2023, "abstract": "A test paper about scaling.", "identifiers": map[string]any{"DOI": "10.1000/test"}},
	})

	var out, errBuf bytes.Buffer
	code := executeLit([]string{"--research-dir", root, "--out", t.TempDir()}, &out, &errBuf, globalOptions{JSON: true})
	if code != 0 {
		t.Fatalf("executeLit exit %d: %s", code, errBuf.String())
	}
	var wrapper map[string]any
	if err := json.Unmarshal(out.Bytes(), &wrapper); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	data, ok := wrapper["data"].(map[string]any)
	if !ok {
		t.Fatalf("data key missing: %+v", wrapper)
	}
	if data["topics"] == nil {
		t.Fatal("topics missing")
	}
}

func TestExecuteNoteAddListClear(t *testing.T) {
	proj := t.TempDir()

	// Add a note
	var out, errBuf bytes.Buffer
	code := executeNote([]string{"add", "--paper", "10.1000/paper1", "--note", "Key result on scaling", "--actor", "bot"},
		&out, &errBuf, globalOptions{Project: proj})
	if code != 0 {
		t.Fatalf("note add exit %d: %s", code, errBuf.String())
	}

	// Add second note
	out.Reset()
	errBuf.Reset()
	code = executeNote([]string{"add", "--paper", "10.1000/paper2", "--note", "Contradicts earlier work"},
		&out, &errBuf, globalOptions{Project: proj})
	if code != 0 {
		t.Fatalf("note add exit %d: %s", code, errBuf.String())
	}

	// List all
	out.Reset()
	errBuf.Reset()
	code = executeNote([]string{"list"}, &out, &errBuf, globalOptions{Project: proj})
	if code != 0 {
		t.Fatalf("note list exit %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "10.1000/paper1") || !strings.Contains(out.String(), "10.1000/paper2") {
		t.Errorf("list missing papers:\n%s", out.String())
	}

	// List with filter
	out.Reset()
	errBuf.Reset()
	code = executeNote([]string{"list", "--paper", "10.1000/paper1"}, &out, &errBuf, globalOptions{Project: proj})
	if code != 0 {
		t.Fatalf("note list filtered exit %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "10.1000/paper1") {
		t.Errorf("filtered list missing paper1:\n%s", out.String())
	}
	if strings.Contains(out.String(), "10.1000/paper2") {
		t.Errorf("filtered list includes paper2:\n%s", out.String())
	}

	// Clear paper1
	out.Reset()
	errBuf.Reset()
	code = executeNote([]string{"clear", "--paper", "10.1000/paper1"}, &out, &errBuf, globalOptions{Project: proj})
	if code != 0 {
		t.Fatalf("note clear exit %d: %s", code, errBuf.String())
	}

	// Verify only paper2 remains
	out.Reset()
	errBuf.Reset()
	code = executeNote([]string{"list"}, &out, &errBuf, globalOptions{Project: proj})
	if code != 0 {
		t.Fatalf("note list after clear exit %d: %s", code, errBuf.String())
	}
	if strings.Contains(out.String(), "10.1000/paper1") {
		t.Errorf("paper1 not cleared:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "10.1000/paper2") {
		t.Errorf("paper2 missing after clear:\n%s", out.String())
	}
}

func TestExecuteNoteJSON(t *testing.T) {
	proj := t.TempDir()
	var out, errBuf bytes.Buffer
	code := executeNote([]string{"add", "--paper", "10.1000/x", "--note", "test note"},
		&out, &errBuf, globalOptions{Project: proj, JSON: true})
	if code != 0 {
		t.Fatalf("note add exit %d: %s", code, errBuf.String())
	}
	var wrapper map[string]any
	if err := json.Unmarshal(out.Bytes(), &wrapper); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	data, ok := wrapper["data"].(map[string]any)
	if !ok {
		t.Fatalf("data key missing: %+v", wrapper)
	}
	if data["action"] != "note.add" {
		t.Errorf("action = %v, want note.add", data["action"])
	}
}

func TestExecuteNoteRequiresProject(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := executeNote([]string{"add", "--paper", "10.1000/1", "--note", "test"}, &out, &errBuf, globalOptions{})
	if code == 0 {
		t.Fatal("expected non-zero exit for missing project")
	}
}
