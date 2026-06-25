package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCitationGraphJSONAddsStemAndHref(t *testing.T) {
	dir := t.TempDir()
	writeCitationGraph(t, dir, sampleCitationGraph)

	graph, err := BuildCitationGraphJSON(dir)
	if err != nil {
		t.Fatalf("BuildCitationGraphJSON: %v", err)
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 {
		t.Fatalf("graph json = %+v", graph)
	}
	var ap GraphJSONNode
	for _, n := range graph.Nodes {
		if n.ID == "10.1000/ap" {
			ap = n
		}
	}
	if ap.Stem != "10-1000-ap" || ap.Href != "/papers/10-1000-ap" {
		t.Fatalf("node = %+v, want stem/href derived from id", ap)
	}
	if graph.Edges[0].Source != "10.1000/ap" || graph.Edges[0].Target != "10.1000/cat" {
		t.Fatalf("edge = %+v", graph.Edges[0])
	}
}

func TestBuildCitationGraphJSONPrefersKnowledgeGraphArtifact(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "data", "knowledge-graph.json"), map[string]any{"schemaVersion": "1", "nodes": []map[string]string{{"id": "paper:10.1000/kg", "kind": "paper", "label": "KG paper"}, {"id": "concept:ignored", "kind": "concept", "label": "ignored"}}, "edges": []map[string]string{{"source": "paper:10.1000/kg", "target": "paper:10.1000/ref", "kind": "cites"}}})
	writeCitationGraph(t, dir, sampleCitationGraph)
	graph, err := BuildCitationGraphJSON(dir)
	if err != nil {
		t.Fatalf("BuildCitationGraphJSON: %v", err)
	}
	if len(graph.Nodes) != 2 || graph.Nodes[0].ID != "10.1000/kg" || graph.Edges[0].Target != "10.1000/ref" {
		t.Fatalf("did not prefer knowledge graph artifact: %+v", graph)
	}
}

func TestCitationGraphJSONEndpointServesJSON(t *testing.T) {
	dir := t.TempDir()
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	body, status, ctype := getURL(t, ts.URL+"/artifacts/graph.json")
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if !strings.Contains(ctype, "application/json") {
		t.Fatalf("content-type = %q", ctype)
	}
	var graph GraphJSON
	if err := json.Unmarshal([]byte(body), &graph); err != nil {
		t.Fatalf("unmarshal endpoint body: %v\n%s", err, body)
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 {
		t.Fatalf("endpoint graph = %+v", graph)
	}
}

func TestCitationGraphJSAssetServed(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()

	body, status, ctype := getURL(t, ts.URL+"/assets/citation-graph.js")
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if !strings.Contains(ctype, "javascript") {
		t.Fatalf("content-type = %q", ctype)
	}
	if !strings.Contains(body, "renderCitationGraph") {
		t.Fatalf("js asset missing renderer entry point:\n%s", body)
	}
}

func TestArtifactsPageMountsInteractiveGraph(t *testing.T) {
	dir := t.TempDir()
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/artifacts")
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	for _, want := range []string{
		`id="citation-graph"`,
		`data-src="/artifacts/graph.json"`,
		`src="/assets/citation-graph.js"`,
		// Server-rendered SVG remains as the no-JS fallback.
		"Citation graph visualization",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("/artifacts missing %q:\n%s", want, body)
		}
	}
}
