package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/citations"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteCitationsDomainMapWritesArtifact(t *testing.T) {
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-1.json"), parsing.ParsedDocument{PaperID: "paper-1", Title: "Solar catalyst study", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Solar catalyst passage."}}}}})
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-2.json"), parsing.ParsedDocument{PaperID: "paper-2", Title: "Screening bias study", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p2", PaperID: "paper-2", SectionID: "s1", Text: "Screening bias passage."}}}}})
	graph := citations.NewGraph()
	graph.AddCitation("paper-1", "paper-2")
	graphData, _ := graph.ExportJSON()
	graphPath := filepath.Join(project, "data", "citation-graph.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	if err := os.WriteFile(graphPath, graphData, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
	out := filepath.Join(project, "data", "domain-map.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "domain-map", "--parsed-dir", parsedDir, "--graph", graphPath, "--out", out, "--label", "solar=Reviewer solar fuels", "--history", "merge:solar,catalyst:solar-catalyst:reviewer-a:same concept", "--model", "bertopic-fixture"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var artifact citations.DomainMapArtifact
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	if artifact.ModelSettings.Model != "bertopic-fixture" || len(artifact.Topics) == 0 || len(artifact.MergeSplitHistory) != 1 {
		t.Fatalf("artifact = %#v", artifact)
	}
	if artifact.Topics[0].RepresentativePapers == nil || artifact.Topics[0].RepresentativePassages == nil {
		t.Fatalf("missing representatives: %#v", artifact.Topics[0])
	}
}
