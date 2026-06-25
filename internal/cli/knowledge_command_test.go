package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteKnowledgeQueryPrefersGeneratedGraphArtifact(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFixture(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Raw", Identifiers: library.Identifiers{DOI: "10.1000/raw"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"tags": "raw"}}}}})
	writeJSONFixture(t, filepath.Join(project, "data", "knowledge-graph.json"), map[string]any{"schemaVersion": "1", "nodes": []map[string]string{{"id": "paper:generated", "kind": "paper", "label": "Generated artifact"}, {"id": "concept:artifact", "kind": "concept", "label": "artifact"}}, "edges": []map[string]string{{"source": "paper:generated", "target": "concept:artifact", "kind": "has_concept"}}})
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "knowledge", "query", "--project", project, "--term", "artifact"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("paper:generated")) || bytes.Contains(stdout.Bytes(), []byte("tag:raw")) {
		t.Fatalf("did not prefer generated graph artifact: %s", stdout.String())
	}
}

func TestExecuteKnowledgePathFindsShortestPath(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFixture(t, filepath.Join(project, "data", "knowledge-graph.json"), map[string]any{"schemaVersion": "1", "nodes": []map[string]string{{"id": "paper:a", "kind": "paper", "label": "A"}, {"id": "concept:bridge", "kind": "concept", "label": "Bridge"}, {"id": "paper:b", "kind": "paper", "label": "B"}}, "edges": []map[string]string{{"source": "paper:a", "target": "concept:bridge", "kind": "has_concept"}, {"source": "concept:bridge", "target": "paper:b", "kind": "has_concept"}}})
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "knowledge", "path", "--project", project, "--from", "paper:a", "--to", "paper:b"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{"paper:a", "concept:bridge", "paper:b"} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Fatalf("path output missing %q: %s", want, stdout.String())
		}
	}
}

func TestExecuteKnowledgeQueryMergesProjectArtifacts(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFixture(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"tags": "catalysts"}}}}})
	writeJSONFixture(t, filepath.Join(project, "data", "evidence.json"), []evidence.EvidenceItem{{PaperID: "10.1000/cat", SchemaName: "outcomes", Values: map[string]string{"outcome": "efficiency"}, Status: evidence.StatusAccepted}})
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "knowledge", "query", "--project", project, "--term", "catalysts"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Graph struct {
				Nodes []struct {
					ID   string `json:"id"`
					Kind string `json:"kind"`
				} `json:"nodes"`
				Edges []struct {
					Kind string `json:"kind"`
				} `json:"edges"`
			} `json:"graph"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	if !env.OK || len(env.Data.Graph.Nodes) < 2 {
		t.Fatalf("unexpected graph: %#v", env)
	}
	foundTag := false
	for _, node := range env.Data.Graph.Nodes {
		if node.ID == "tag:catalysts" {
			foundTag = true
		}
	}
	if !foundTag {
		t.Fatalf("missing catalysts tag in %s", stdout.String())
	}
}
