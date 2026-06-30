package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeResearchDir creates a fake research dir with topic subdirs, each
// containing a results.jsonl built from the provided records.
func makeResearchDir(t *testing.T, topics map[string][]map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	for topic, records := range topics {
		topicDir := filepath.Join(dir, topic)
		if err := os.MkdirAll(topicDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", topicDir, err)
		}
		writeBatchResults(t, topicDir, records)
	}
	return dir
}

func TestMetaOverlapRanksByTopicCount(t *testing.T) {
	researchDir := makeResearchDir(t, map[string][]map[string]any{
		"topic-a": {
			{"Title": "Optuna Paper", "Identifiers": map[string]any{"DOI": "10.1145/3292500.3330701"}, "OpenAccess": false, "URLs": []string{}},
			{"Title": "Unique to A", "Identifiers": map[string]any{"DOI": "10.1000/unique-a"}, "OpenAccess": false, "URLs": []string{}},
		},
		"topic-b": {
			{"Title": "Optuna Paper", "Identifiers": map[string]any{"DOI": "10.1145/3292500.3330701"}, "OpenAccess": false, "URLs": []string{}},
			{"Title": "Unique to B", "Identifiers": map[string]any{"DOI": "10.1000/unique-b"}, "OpenAccess": false, "URLs": []string{}},
		},
		"topic-c": {
			{"Title": "Optuna Paper", "Identifiers": map[string]any{"DOI": "10.1145/3292500.3330701"}, "OpenAccess": false, "URLs": []string{}},
		},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"meta", "overlap", "--research-dir", researchDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Optuna") {
		t.Errorf("stdout missing Optuna paper: %s", out)
	}
	if !strings.Contains(out, "3") {
		t.Errorf("stdout missing topic count 3: %s", out)
	}
	// Optuna (3 topics) should appear before unique papers (1 topic)
	optPos := strings.Index(out, "Optuna")
	uniquePos := strings.Index(out, "Unique to")
	if uniquePos != -1 && optPos > uniquePos {
		t.Errorf("Optuna (3 topics) should rank above unique papers (1 topic)")
	}
}

func TestMetaOverlapJSONOutput(t *testing.T) {
	researchDir := makeResearchDir(t, map[string][]map[string]any{
		"topic-a": {
			{"Title": "Shared Paper", "Identifiers": map[string]any{"DOI": "10.1000/shared"}, "OpenAccess": false, "URLs": []string{}},
		},
		"topic-b": {
			{"Title": "Shared Paper", "Identifiers": map[string]any{"DOI": "10.1000/shared"}, "OpenAccess": false, "URLs": []string{}},
		},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "meta", "overlap", "--research-dir", researchDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	papers, _ := data["papers"].([]any)
	if len(papers) == 0 {
		t.Fatalf("JSON papers empty; data = %v", data)
	}
	first, _ := papers[0].(map[string]any)
	if first["doi"] != "10.1000/shared" {
		t.Errorf("first paper doi = %v, want 10.1000/shared", first["doi"])
	}
	if first["topicCount"] != float64(2) {
		t.Errorf("topicCount = %v, want 2", first["topicCount"])
	}
	topics, _ := first["topics"].([]any)
	if len(topics) != 2 {
		t.Errorf("topics len = %d, want 2", len(topics))
	}
}

func TestMetaOverlapMinTopicsFilter(t *testing.T) {
	researchDir := makeResearchDir(t, map[string][]map[string]any{
		"topic-a": {
			{"Title": "In Two", "Identifiers": map[string]any{"DOI": "10.1000/two"}, "OpenAccess": false, "URLs": []string{}},
			{"Title": "In One", "Identifiers": map[string]any{"DOI": "10.1000/one"}, "OpenAccess": false, "URLs": []string{}},
		},
		"topic-b": {
			{"Title": "In Two", "Identifiers": map[string]any{"DOI": "10.1000/two"}, "OpenAccess": false, "URLs": []string{}},
		},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"meta", "overlap", "--research-dir", researchDir, "--min-topics", "2"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "In Two") {
		t.Errorf("expected 'In Two' in output: %s", out)
	}
	if strings.Contains(out, "In One") {
		t.Errorf("'In One' should be filtered out by --min-topics 2: %s", out)
	}
}

func TestMetaOverlapRequiresResearchDir(t *testing.T) {
	code := Execute([]string{"meta", "overlap"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestMetaOverlapEmptyDirSucceeds(t *testing.T) {
	stdout := new(bytes.Buffer)
	code := Execute([]string{"meta", "overlap", "--research-dir", t.TempDir()}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
}

func TestMetaOverlapSummaryLine(t *testing.T) {
	researchDir := makeResearchDir(t, map[string][]map[string]any{
		"topic-a": {
			{"Title": "Paper X", "Identifiers": map[string]any{"DOI": "10.1000/x"}, "OpenAccess": false, "URLs": []string{}},
		},
		"topic-b": {
			{"Title": "Paper X", "Identifiers": map[string]any{"DOI": "10.1000/x"}, "OpenAccess": false, "URLs": []string{}},
			{"Title": "Paper Y", "Identifiers": map[string]any{"DOI": "10.1000/y"}, "OpenAccess": false, "URLs": []string{}},
		},
	})

	stdout := new(bytes.Buffer)
	code := Execute([]string{"meta", "overlap", "--research-dir", researchDir}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	out := stdout.String()
	// Should report total topics and total unique papers
	if !strings.Contains(out, "2") {
		t.Errorf("summary should mention 2 topics: %s", out)
	}
}
