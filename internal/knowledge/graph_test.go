package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/report"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

func TestBuildProjectKnowledgeGraphMergesSpineArtifacts(t *testing.T) {
	input := ProjectGraphInput{
		LibraryRecords:   []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"collections": "Hydrogen", "tags": "HER; catalysts", "concepts": "photocatalysis"}}}}},
		CitationEdges:    []CitationEdge{{Source: "10.1000/cat", Target: "10.1000/ref"}},
		ParsedDocuments:  []parsing.ParsedDocument{{PaperID: "10.1000/cat", References: []parsing.Reference{{Title: "Reference paper", DOI: "10.1000/ref"}}}},
		EvidenceItems:    []evidence.EvidenceItem{{PaperID: "10.1000/cat", SchemaName: "outcomes", Values: map[string]string{"outcome": "efficiency"}, Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "p1"}, Status: evidence.StatusAccepted}},
		ScreeningEvents:  []screening.DecisionEvent{{PaperID: "10.1000/cat", Stage: screening.StageTitleAbstract, Decision: screening.DecisionInclude, Reviewer: "r1"}},
		AnalysisRuns:     []analysis.AnalysisRun{{ID: "run1", InputRows: []analysis.InputRow{{PaperID: "10.1000/cat", EffectSize: 0.5}}}},
		ReportTrace:      report.CitationEvidenceTraceView{Claims: []report.ClaimTraceView{{ClaimID: "claim1", PaperID: "10.1000/cat", ClaimText: "Catalysts improve efficiency"}}},
		ProvenanceEvents: []provenance.Event{{ID: "evt1", Action: "source.plan.approved", Target: "10.1000/cat"}},
	}
	graph := BuildProjectKnowledgeGraph(input)
	for _, id := range []string{"paper:10.1000/cat", "collection:Hydrogen", "tag:HER", "concept:photocatalysis", "citation:10.1000/cat->10.1000/ref", "reference:10.1000/cat:0", "evidence:10.1000/cat:0", "screening:10.1000/cat:title_abstract:0", "analysis:run1", "claim:claim1", "provenance:evt1"} {
		if !graph.HasNode(id) {
			t.Fatalf("missing node %s; nodes=%#v", id, graph.Nodes)
		}
	}
	if !graph.HasEdge("paper:10.1000/cat", "tag:HER", "tagged_with") || !graph.HasEdge("claim:claim1", "evidence:10.1000/cat:0", "supported_by") || !graph.HasEdge("analysis:run1", "paper:10.1000/cat", "analyzes") {
		t.Fatalf("missing merged edges: %#v", graph.Edges)
	}
	result := QueryProjectKnowledgeGraph(graph, "catalysts")
	if len(result.Nodes) == 0 || !result.HasNode("tag:catalysts") {
		t.Fatalf("query did not find catalysts tag: %#v", result)
	}
}

func TestBuildProjectKnowledgeGraphFromProjectReadsLocalArtifacts(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"tags": "catalysts"}}}}})
	writeJSON(t, filepath.Join(project, "data", "citation-graph.json"), map[string]any{"nodes": []map[string]string{{"id": "10.1000/cat"}, {"id": "10.1000/ref"}}, "edges": []map[string]string{{"source": "10.1000/cat", "target": "10.1000/ref"}}})
	writeJSON(t, filepath.Join(project, "parsed", "cat.json"), parsing.ParsedDocument{PaperID: "10.1000/cat", References: []parsing.Reference{{Title: "Reference paper", DOI: "10.1000/ref"}}})
	writeJSON(t, filepath.Join(project, "data", "evidence.items.json"), []evidence.EvidenceItem{{PaperID: "10.1000/cat", SchemaName: "outcomes", Status: evidence.StatusAccepted}})
	writeJSON(t, filepath.Join(project, "data", "screening.events.json"), []screening.DecisionEvent{{PaperID: "10.1000/cat", Stage: screening.StageTitleAbstract, Decision: screening.DecisionInclude}})
	writeJSON(t, filepath.Join(project, "analysis", "run1-run.json"), analysis.AnalysisRun{ID: "run1", InputRows: []analysis.InputRow{{PaperID: "10.1000/cat", EffectSize: 1}}})
	writeJSON(t, filepath.Join(project, "data", "claim-trace.json"), report.CitationEvidenceTraceView{Claims: []report.ClaimTraceView{{ClaimID: "claim1", PaperID: "10.1000/cat", ClaimText: "claim"}}})
	if err := provenance.Append(project, provenance.Event{SchemaVersion: "1", ID: "evt1", Timestamp: "2026-01-01T00:00:00Z", Actor: "tester", Action: "forge.state.transition", Target: "10.1000/cat", Inputs: map[string]any{}, Outputs: map[string]any{}, Warnings: []string{}}); err != nil {
		t.Fatalf("provenance: %v", err)
	}
	graph, err := BuildProjectKnowledgeGraphFromProject(project)
	if err != nil {
		t.Fatalf("BuildProjectKnowledgeGraphFromProject: %v", err)
	}
	for _, id := range []string{"paper:10.1000/cat", "tag:catalysts", "reference:10.1000/cat:0", "evidence:10.1000/cat:0", "screening:10.1000/cat:title_abstract:0", "analysis:run1", "claim:claim1", "provenance:evt1"} {
		if !graph.HasNode(id) {
			t.Fatalf("missing %s", id)
		}
	}
}

func TestLoadProjectKnowledgeGraphFromProjectPrefersGeneratedArtifact(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Raw", Identifiers: library.Identifiers{DOI: "10.1000/raw"}}})
	writeJSON(t, filepath.Join(project, "data", "knowledge-graph.json"), ProjectKnowledgeGraph{SchemaVersion: "1", Nodes: []KnowledgeNode{{ID: "paper:generated", Kind: "paper", Label: "Generated"}}})
	graph, err := LoadProjectKnowledgeGraphFromProject(project)
	if err != nil {
		t.Fatalf("LoadProjectKnowledgeGraphFromProject: %v", err)
	}
	if !graph.HasNode("paper:generated") || graph.HasNode("paper:10.1000/raw") {
		t.Fatalf("did not prefer generated artifact: %#v", graph.Nodes)
	}
}

func TestReadProjectKnowledgeGraphArtifactRejectsSymlink(t *testing.T) {
	project := t.TempDir()
	dataDir := filepath.Join(project, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-knowledge-graph.json")
	writeJSON(t, outsidePath, ProjectKnowledgeGraph{
		SchemaVersion: "1",
		Nodes:         []KnowledgeNode{{ID: "external-private", Kind: "paper", Label: "External private graph"}},
	})
	artifactPath := filepath.Join(dataDir, "knowledge-graph.json")
	if err := os.Symlink(outsidePath, artifactPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if graph, err := ReadProjectKnowledgeGraphArtifact(project); err == nil {
		t.Fatalf("ReadProjectKnowledgeGraphArtifact returned external graph through symlink: %#v", graph)
	}
	info, err := os.Lstat(artifactPath)
	if err != nil {
		t.Fatalf("lstat knowledge graph artifact: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("knowledge graph read replaced the symlink: mode=%v", info.Mode())
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
