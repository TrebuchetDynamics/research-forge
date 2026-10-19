package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteHistorySearch(t *testing.T) {
	root := t.TempDir()
	subA := filepath.Join(root, "scaling")
	subB := filepath.Join(root, "language")

	writeFixtureSearchDir(t, subA, "scaling laws", []map[string]any{
		{"title": "Scaling Laws for Neural Language Models", "year": 2020, "abstract": "We study scaling laws for language models.", "identifiers": map[string]any{"DOI": "10.1000/scaling"}},
		{"title": "Unrelated Topic Paper", "year": 2021, "abstract": "About biology.", "identifiers": map[string]any{"DOI": "10.1000/bio"}},
	})
	writeFixtureSearchDir(t, subB, "language models", []map[string]any{
		{"title": "Large Language Models: A Survey", "year": 2023, "abstract": "Survey of LLMs.", "identifiers": map[string]any{"DOI": "10.1000/llm"}},
		{"title": "Scaling Laws for Neural Language Models", "year": 2020, "abstract": "We study scaling laws for language models.", "identifiers": map[string]any{"DOI": "10.1000/scaling"}},
	})

	var out, errBuf bytes.Buffer
	code := executeHistory([]string{"search", "scaling", "--research-dir", root}, &out, &errBuf, globalOptions{})
	if code != 0 {
		t.Fatalf("history search exit %d: %s", code, errBuf.String())
	}
	got := out.String()
	if !strings.Contains(got, `"scaling"`) && !strings.Contains(got, "scaling laws") {
		t.Errorf("history search missing scaling match:\n%s", got)
	}
	// The fixture writes manifest + results-deduped.jsonl, search reads those
	if !strings.Contains(got, "Scaling Laws for Neural Language Models") {
		t.Errorf("history search missing scaling paper:\n%s", got)
	}
}

func TestExecuteHistorySearchNoResults(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "topic")
	writeFixtureSearchDir(t, sub, "test", []map[string]any{
		{"title": "Alpha Beta Gamma", "identifiers": map[string]any{"DOI": "10.1000/abg"}},
	})
	var out, errBuf bytes.Buffer
	code := executeHistory([]string{"search", "zzzz-nomatch", "--research-dir", root}, &out, &errBuf, globalOptions{})
	if code != 0 {
		t.Fatalf("history search no-results exit %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "no results") {
		t.Errorf("expected no results message, got:\n%s", out.String())
	}
}

func TestExecuteHistoryRequiresCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := executeHistory(nil, &out, &errBuf, globalOptions{})
	if code == 0 {
		t.Fatal("expected non-zero exit for empty history command")
	}
}

func TestReplicatePlanStructureOffline(t *testing.T) {
	workflowFixtureServer(t)
	path := filepath.Join(t.TempDir(), "plan.md")
	var out, errBuf bytes.Buffer
	if code := Execute([]string{"replicate", "10.1000/example", "--out", path}, &out, &errBuf); code != 0 {
		t.Fatalf("replicate exit %d: %s %s", code, &out, &errBuf)
	}
	plan, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Replication Plan", "Scaling Study", "10.1000/example", "github.com/example/scaling", "## Method Summary", "_Abstract not available from source._", "## Replication Checklist", "cc-by"} {
		if !bytes.Contains(plan, []byte(want)) {
			t.Errorf("plan missing %q: %s", want, plan)
		}
	}
	if !bytes.Equal(plan, out.Bytes()) {
		t.Error("saved plan differs from stdout")
	}
}
