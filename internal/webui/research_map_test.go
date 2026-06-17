package webui

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestResearchMapCockpitShowsLiveFeaturesAndSnapshot(t *testing.T) {
	project := t.TempDir()
	writeJSON(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/cat"}, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"concepts": "photocatalysis", "tags": "catalysts"}}}}})
	writeJSON(t, filepath.Join(project, "data", "citation-graph.json"), map[string]any{"edges": []map[string]string{{"source": "10.1000/cat", "target": "10.1000/ref"}}})
	writeJSON(t, filepath.Join(project, "data", "evidence.json"), []evidence.EvidenceItem{{PaperID: "10.1000/cat", SchemaName: "outcomes", Status: evidence.StatusAccepted}})
	state, err := BuildResearchMapCockpitState(project)
	if err != nil {
		t.Fatalf("BuildResearchMapCockpitState: %v", err)
	}
	if len(state.ConceptMap) == 0 || len(state.CitationNeighborhoods) == 0 || len(state.RetrievalClusters) == 0 || state.EvidenceCoverage.Accepted == 0 || state.SnapshotExportPath != "/map/snapshot.json" {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewResearchMapHandler(state))
	for _, want := range []string{"Research map cockpit", "Concept maps", "Citation neighborhoods", "Retrieval clusters", "Evidence coverage", "Snapshot export", "photocatalysis"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
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
