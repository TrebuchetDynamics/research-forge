package webui

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestResearchMapCockpitSupportsFilteringNeighborhoodProvenanceAndKeyboardAlternatives(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"concepts": "photocatalysis", "tags": "catalysts"}}}}, {Title: "Battery", Identifiers: library.Identifiers{DOI: "10.1000/bat"}}})
	writeJSON(t, filepath.Join(project, "data", "citation-graph.json"), map[string]any{"edges": []map[string]string{{"source": "10.1000/cat", "target": "10.1000/ref"}}})
	state, err := BuildResearchMapCockpitStateWithOptions(project, ResearchMapOptions{Filter: "photo", Neighborhood: "10.1000/cat", IncludeProvenance: true})
	if err != nil {
		t.Fatalf("BuildResearchMapCockpitStateWithOptions: %v", err)
	}
	if state.Filter != "photo" || state.Neighborhood != "10.1000/cat" || len(state.KeyboardAlternatives) == 0 || len(state.ProvenanceOverlays) == 0 {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewResearchMapHandler(state))
	for _, want := range []string{"Filter: photo", "Neighborhood: 10.1000/cat", "Provenance overlays", "Keyboard-accessible alternatives"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestResearchMapCockpitShowsLiveFeaturesAndSnapshot(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"concepts": "photocatalysis", "tags": "catalysts"}}}}})
	writeJSON(t, filepath.Join(project, "data", "citation-graph.json"), map[string]any{"edges": []map[string]string{{"source": "10.1000/cat", "target": "10.1000/ref"}}})
	writeJSON(t, filepath.Join(project, "data", "evidence.json"), []evidence.EvidenceItem{{PaperID: "10.1000/cat", SchemaName: "outcomes", Status: evidence.StatusAccepted}})
	writeJSON(t, filepath.Join(project, "data", "parser-quality.json"), map[string]any{"parserRuns": []map[string]any{{"parserName": "grobid", "qualityScore": 3.5}}, "conflicts": []map[string]any{{"field": "title", "status": "review-required"}}})
	writeJSON(t, filepath.Join(project, "data", "screening-priority.json"), []map[string]any{{"id": "paper-1", "label": "Catalyst review", "detail": "uncertainty 0.51"}})
	writeJSON(t, filepath.Join(project, "data", "screening-status.json"), []map[string]any{{"id": "paper-1", "label": "Catalyst review", "detail": "included"}})
	writeJSON(t, filepath.Join(project, "data", "retrieval-hits.json"), []map[string]any{{"id": "hit-1", "label": "opensearch hit", "detail": "BM25+vector"}})
	state, err := BuildResearchMapCockpitState(project)
	if err != nil {
		t.Fatalf("BuildResearchMapCockpitState: %v", err)
	}
	if len(state.ConceptMap) == 0 || len(state.CitationNeighborhoods) == 0 || len(state.RetrievalClusters) == 0 || len(state.RetrievalHits) == 0 || len(state.ScreeningPriority) == 0 || len(state.ScreeningStatus) == 0 || len(state.ParserQuality) == 0 || state.EvidenceCoverage.Accepted == 0 || state.SnapshotExportPath != "/map/snapshot.json" {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewResearchMapHandler(state))
	for _, want := range []string{"Research map cockpit", "citation graph", "OpenAlex concepts", "Zotero collections/tags", "Concept maps", "Citation neighborhoods", "Retrieval clusters", "Retrieval hits", "Screening priority", "screening status", "Parser quality", "Evidence coverage", "filters", "keyboard navigation", "no-JS tables", "Snapshot export", "photocatalysis"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestResearchMapCockpitDoesNotReadSymlinkedRetrievalItems(t *testing.T) {
	projectPath := t.TempDir()
	writeJSON(t, filepath.Join(projectPath, "data", "library.json"), []library.PaperRecord{})
	externalPath := filepath.Join(t.TempDir(), "retrieval-hits.json")
	writeJSON(t, externalPath, []ResearchMapItem{{ID: "external-private-hit", Label: "External private retrieval", Detail: "external-private-ranking"}})
	retrievalPath := filepath.Join(projectPath, "data", "retrieval-hits.json")
	if err := os.Symlink(externalPath, retrievalPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	state, err := BuildResearchMapCockpitState(projectPath)
	if err != nil {
		t.Fatalf("BuildResearchMapCockpitState: %v", err)
	}
	if len(state.RetrievalHits) != 0 {
		t.Fatalf("research map accepted symlinked retrieval items: %#v", state.RetrievalHits)
	}
	body := renderHandler(t, NewResearchMapHandler(state))
	for _, private := range []string{"external-private-hit", "External private retrieval", "external-private-ranking"} {
		if strings.Contains(body, private) {
			t.Fatalf("research map disclosed %q from symlinked retrieval items: %s", private, body)
		}
	}
	if info, err := os.Lstat(retrievalPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("retrieval items symlink changed: info=%v err=%v", info, err)
	}
}

func TestResearchMapCockpitDoesNotReadSymlinkedParserQuality(t *testing.T) {
	projectPath := t.TempDir()
	writeJSON(t, filepath.Join(projectPath, "data", "library.json"), []library.PaperRecord{})
	externalPath := filepath.Join(t.TempDir(), "parser-quality.json")
	writeJSON(t, externalPath, map[string]any{
		"parserRuns": []map[string]any{{"parserName": "external-private-parser", "qualityScore": 9.99}},
		"conflicts":  []map[string]any{{"field": "external-private-field", "status": "external-private-status"}},
	})
	qualityPath := filepath.Join(projectPath, "data", "parser-quality.json")
	if err := os.Symlink(externalPath, qualityPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	state, err := BuildResearchMapCockpitState(projectPath)
	if err != nil {
		t.Fatalf("BuildResearchMapCockpitState: %v", err)
	}
	if len(state.ParserQuality) != 0 {
		t.Fatalf("research map accepted symlinked parser quality: %#v", state.ParserQuality)
	}
	body := renderHandler(t, NewResearchMapHandler(state))
	for _, private := range []string{"external-private-parser", "external-private-field", "external-private-status"} {
		if strings.Contains(body, private) {
			t.Fatalf("research map disclosed %q from symlinked parser quality: %s", private, body)
		}
	}
	if info, err := os.Lstat(qualityPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("parser quality symlink changed: info=%v err=%v", info, err)
	}
}

func TestResearchMapPrefersGeneratedKnowledgeGraphArtifact(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Ignored rebuild concept", Identifiers: library.Identifiers{DOI: "10.1000/raw"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"concepts": "raw-concept"}}}}})
	writeJSON(t, filepath.Join(project, "data", "knowledge-graph.json"), map[string]any{"schemaVersion": "1", "nodes": []map[string]any{{"id": "paper:p1", "kind": "paper", "label": "Generated graph paper"}, {"id": "concept:generated-concept", "kind": "concept", "label": "generated-concept"}}, "edges": []map[string]string{{"id": "paper:p1 -has_concept-> concept:generated-concept", "source": "paper:p1", "target": "concept:generated-concept", "kind": "has_concept"}}})
	state, err := BuildResearchMapCockpitState(project)
	if err != nil {
		t.Fatalf("BuildResearchMapCockpitState: %v", err)
	}
	if len(state.ConceptMap) != 1 || state.ConceptMap[0].Label != "generated-concept" {
		t.Fatalf("did not prefer generated graph artifact: %#v", state.ConceptMap)
	}
}

func TestRouterServesResearchMapAndSnapshot(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"concepts": "photocatalysis"}}}}})
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: project}))
	defer ts.Close()
	body := httpGetBody(t, ts.URL+"/map")
	if !strings.Contains(body, "Research map cockpit") || !strings.Contains(body, "Snapshot export") {
		t.Fatalf("/map missing cockpit: %s", body)
	}
	body = httpGetBody(t, ts.URL+"/map/snapshot.json")
	var state ResearchMapCockpitState
	if err := json.Unmarshal([]byte(body), &state); err != nil {
		t.Fatalf("snapshot json: %v\n%s", err, body)
	}
	if len(state.ConceptMap) == 0 {
		t.Fatalf("snapshot missing concept map: %#v", state)
	}
}
