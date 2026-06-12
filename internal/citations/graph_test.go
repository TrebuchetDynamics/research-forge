package citations

import "testing"

func TestCitationGraphStoresBackwardForwardAndClusterScaffolds(t *testing.T) {
	graph := NewGraph()
	graph.AddCitation("paper-a", "paper-b")
	graph.AddCitation("paper-c", "paper-b")
	if len(graph.Backward("paper-a")) != 1 || graph.Backward("paper-a")[0] != "paper-b" {
		t.Fatalf("backward = %#v", graph.Backward("paper-a"))
	}
	forward := graph.Forward("paper-b")
	if len(forward) != 2 || forward[0] != "paper-a" || forward[1] != "paper-c" {
		t.Fatalf("forward = %#v", forward)
	}
	co := graph.CoCitationClusters()
	if len(co) != 1 || co[0].ReferenceID != "paper-b" || len(co[0].CitingPaperIDs) != 2 {
		t.Fatalf("co-citation = %#v", co)
	}
	coupling := graph.BibliographicCoupling()
	if len(coupling) != 1 || coupling[0].SharedReferenceID != "paper-b" {
		t.Fatalf("coupling = %#v", coupling)
	}
}

func TestCitationGraphExportInteroperableJSON(t *testing.T) {
	graph := NewGraph()
	graph.AddCitation("paper-a", "paper-b")
	exported, err := graph.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON returned error: %v", err)
	}
	want := "{\n  \"nodes\": [\n    {\n      \"id\": \"paper-a\"\n    },\n    {\n      \"id\": \"paper-b\"\n    }\n  ],\n  \"edges\": [\n    {\n      \"source\": \"paper-a\",\n      \"target\": \"paper-b\"\n    }\n  ]\n}\n"
	if string(exported) != want {
		t.Fatalf("export mismatch:\n%s", exported)
	}
}
