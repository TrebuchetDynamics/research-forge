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
