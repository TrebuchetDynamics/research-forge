package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixtureSearchDir writes a minimal search-batch dir used by brief/compare tests.
func writeFixtureSearchDir(t *testing.T, dir, topic string, papers []map[string]any) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schemaVersion": "1",
		"queries":       []string{topic + " query one", topic + " query two"},
		"sources":       []string{"openalex", "arxiv"},
		"results":       len(papers),
		"deduped":       len(papers),
		"failures":      0,
		"createdAt":     "2026-01-01T00:00:00Z",
	}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, p := range papers {
		row, _ := json.Marshal(p)
		sb.Write(row)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "results-deduped.jsonl"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteBriefWritesSummary(t *testing.T) {
	dir := t.TempDir() + "/topic-a"
	writeFixtureSearchDir(t, dir, "scaling laws", []map[string]any{
		{
			"title":       "Scaling Laws for Neural Language Models",
			"year":        2020,
			"venue":       "arXiv",
			"abstract":    "We study scaling laws.",
			"identifiers": map[string]any{"DOI": "10.48550/arxiv.2001.08361", "ArXivID": "2001.08361"},
		},
		{
			"title":       "A Second Paper on Scaling",
			"year":        2023,
			"openAccess":  true,
			"identifiers": map[string]any{"ArXivID": "2301.00001"},
		},
	})

	var out, errBuf bytes.Buffer
	code := executeBrief([]string{dir}, &out, &errBuf, globalOptions{})
	if code != 0 {
		t.Fatalf("executeBrief exit %d: %s", code, errBuf.String())
	}
	got := out.String()
	for _, want := range []string{"# Research Brief", "Scaling Laws for Neural Language Models", "A Second Paper on Scaling", "10.48550/arxiv.2001.08361", "https://arxiv.org/abs/2301.00001"} {
		if !strings.Contains(got, want) {
			t.Errorf("brief missing %q", want)
		}
	}
	if !strings.Contains(got, "## Gaps & Limitations") {
		t.Error("brief missing gaps section")
	}
}

func TestExecuteBriefJSON(t *testing.T) {
	dir := t.TempDir() + "/topic-b"
	writeFixtureSearchDir(t, dir, "meta analysis", []map[string]any{
		{"title": "Meta", "identifiers": map[string]any{"DOI": "10.1000/meta"}},
	})
	var out, errBuf bytes.Buffer
	code := executeBrief([]string{dir}, &out, &errBuf, globalOptions{JSON: true})
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errBuf.String())
	}
	var wrapper map[string]any
	if err := json.Unmarshal(out.Bytes(), &wrapper); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	data, ok := wrapper["data"].(map[string]any)
	if !ok {
		t.Fatalf("data key missing or wrong type: %+v", wrapper)
	}
	if data["totalRecords"] == nil {
		t.Fatal("missing totalRecords in data")
	}
	if data["topic"] == nil {
		t.Fatal("missing topic in data")
	}
	if data["briefSize"] == nil {
		t.Fatal("missing briefSize in data")
	}
}

func TestExecuteCompareOverlap(t *testing.T) {
	root := t.TempDir()
	t.Logf("root = %s", root)
	dirA := root + "/a"
	dirB := root + "/b"
	t.Logf("dirA = %s", dirA)
	shared := map[string]any{
		"title":       "Shared Paper",
		"identifiers": map[string]any{"DOI": "10.1000/shared"},
	}
	writeFixtureSearchDir(t, dirA, "topic a", []map[string]any{
		shared,
		{"title": "Only In A", "identifiers": map[string]any{"DOI": "10.1000/a"}},
	})
	writeFixtureSearchDir(t, dirB, "topic b", []map[string]any{
		shared,
		{"title": "Only In B", "identifiers": map[string]any{"DOI": "10.1000/b"}},
	})

	var out, errBuf bytes.Buffer
	code := executeCompare([]string{dirA, dirB}, &out, &errBuf, globalOptions{})
	if code != 0 {
		t.Fatalf("executeCompare exit %d: %s", code, errBuf.String())
	}
	got := out.String()
	if !strings.Contains(got, "Shared Paper") {
		t.Error("compare missing shared paper")
	}
	if !strings.Contains(got, "Only In A") || !strings.Contains(got, "Only In B") {
		t.Error("compare missing unique papers")
	}
	if !strings.Contains(got, "**Total unique papers across all dirs:** 3") {
		t.Errorf("compare summary missing union count:\n%s", got)
	}
	if !strings.Contains(got, "**In all directories:** 1") {
		t.Errorf("compare summary missing overlap count:\n%s", got)
	}
}

func TestExecuteCompareLabels(t *testing.T) {
	root := t.TempDir()
	dirA := root + "/a"
	dirB := root + "/b"
	writeFixtureSearchDir(t, dirA, "topic a", []map[string]any{
		{"title": "P1", "identifiers": map[string]any{"DOI": "10.1000/1"}},
	})
	writeFixtureSearchDir(t, dirB, "topic b", []map[string]any{
		{"title": "P1", "identifiers": map[string]any{"DOI": "10.1000/1"}},
	})
	var out, errBuf bytes.Buffer
	code := executeCompare([]string{dirA, dirB, "--labels", "alpha,beta"}, &out, &errBuf, globalOptions{})
	if code != 0 {
		t.Fatalf("executeCompare exit %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "alpha") || !strings.Contains(out.String(), "beta") {
		t.Error("compare labels not applied")
	}
}

// A flag where the search directory belongs must be a usage error, not a
// confusing attempt to open "<flag>/manifest.json".
func TestBriefRejectsFlagAsSearchDir(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"brief", "--out", "brief.md"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "usage: rforge brief <search-dir>") {
		t.Errorf("error does not show usage: %s", combined)
	}
	if strings.Contains(combined, "--out/manifest.json") {
		t.Errorf("treated the flag as a directory: %s", combined)
	}
}

// An empty search directory is a usage error, like every sibling command.
func TestBriefRejectsEmptySearchDir(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"brief", ""}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String()+stderr.String(), "usage: rforge brief <search-dir>") {
		t.Errorf("error does not show usage: %s", stdout.String()+stderr.String())
	}
}
