package citations

import "testing"

func TestResearchLineageViewModelSummarizesBackwardAndForwardCitations(t *testing.T) {
	graph := NewGraph()
	graph.AddCitation("paper-a", "paper-b")
	graph.AddCitation("paper-c", "paper-a")
	vm := NewResearchLineageViewModel(graph, "paper-a")
	if vm.PaperID != "paper-a" || len(vm.References) != 1 || vm.References[0] != "paper-b" || len(vm.CitedBy) != 1 || vm.CitedBy[0] != "paper-c" {
		t.Fatalf("vm = %#v", vm)
	}
}
