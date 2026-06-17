package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

func TestExecuteRetrieveTuneHybridWritesCalibratedTuningFile(t *testing.T) {
	project := t.TempDir()
	queries := []retrieval.HybridTuningQuery{{ID: "q1", Query: "solar", RelevantPassageIDs: []string{"p1"}}}
	lexical := map[string][]retrieval.PassageResult{"q1": {{PassageID: "p2"}, {PassageID: "p1"}}}
	vector := map[string][]retrieval.PassageResult{"q1": {{PassageID: "p1"}}}
	queriesPath := filepath.Join(project, "queries.json")
	lexicalPath := filepath.Join(project, "lexical.json")
	vectorPath := filepath.Join(project, "vector.json")
	outPath := filepath.Join(project, "data", "hybrid-tuning.json")
	writeJSONFixture(t, queriesPath, queries)
	writeJSONFixture(t, lexicalPath, lexical)
	writeJSONFixture(t, vectorPath, vector)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "retrieve", "tune-hybrid", "--queries", queriesPath, "--lexical", lexicalPath, "--vector", vectorPath, "--out", outPath, "--k", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var file retrieval.HybridTuningFile
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("decode tuning: %v", err)
	}
	if file.QuerySetChecksum == "" || file.SelectedConfiguration.Name == "" || len(file.Evaluations) == 0 || file.Evaluations[0].Score == 0 {
		t.Fatalf("tuning file = %#v", file)
	}
}

func writeJSONFixture(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
