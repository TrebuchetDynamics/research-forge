package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleCitationGraph = `{
  "nodes": [{"id": "10.1000/ap"}, {"id": "10.1000/cat"}],
  "edges": [{"source": "10.1000/ap", "target": "10.1000/cat"}]
}`

func writeCitationGraph(t *testing.T, projectPath, body string) {
	t.Helper()
	dir := filepath.Join(projectPath, "data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "citation-graph.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func TestBuildCitationGraphFromProjectData(t *testing.T) {
	dir := t.TempDir()
	writeCitationGraph(t, dir, sampleCitationGraph)

	state, err := BuildArtifactDashboardState(dir)
	if err != nil {
		t.Fatalf("BuildArtifactDashboardState: %v", err)
	}
	if len(state.CitationGraph.Nodes) != 2 || len(state.CitationGraph.Edges) != 1 {
		t.Fatalf("citation graph = %+v", state.CitationGraph)
	}
	if state.CitationGraph.Edges[0].Source != "10.1000/ap" || state.CitationGraph.Edges[0].Target != "10.1000/cat" {
		t.Fatalf("edge = %+v", state.CitationGraph.Edges[0])
	}
}

func TestBuildArtifactDashboardStateRejectsSymlinkedCitationGraph(t *testing.T) {
	projectPath := t.TempDir()
	dataDir := filepath.Join(projectPath, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir project data: %v", err)
	}

	externalProject := t.TempDir()
	writeCitationGraph(t, externalProject, `{
  "nodes": [{"id": "external-private-node"}],
  "edges": []
}`)
	graphPath := filepath.Join(dataDir, "citation-graph.json")
	if err := os.Symlink(filepath.Join(externalProject, "data", "citation-graph.json"), graphPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := BuildArtifactDashboardState(projectPath); err == nil {
		t.Fatal("BuildArtifactDashboardState accepted a symlinked citation graph")
	}
	if info, err := os.Lstat(graphPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("citation graph symlink changed: info=%v err=%v", info, err)
	}
}

func TestArtifactsPageRendersClickableCitationGraph(t *testing.T) {
	dir := t.TempDir()
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/artifacts")
	if status != http.StatusOK {
		t.Fatalf("GET /artifacts status = %d", status)
	}
	if !strings.Contains(body, "<svg") || !strings.Contains(body, "Citation graph visualization") {
		t.Fatalf("/artifacts missing citation graph SVG: %s", body)
	}
	// Nodes link to the corresponding paper reading page using the safe stem.
	if !strings.Contains(body, `href="/papers/10-1000-ap"`) {
		t.Fatalf("/artifacts citation graph node missing clickable link: %s", body)
	}
}

func TestGraphNodeStemMatchesParseStem(t *testing.T) {
	if got := graphNodeStem("10.1000/AP"); got != "10-1000-ap" {
		t.Fatalf("graphNodeStem = %q, want 10-1000-ap", got)
	}
}
