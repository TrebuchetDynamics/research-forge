package citations

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestBuildDomainMapArtifactIncludesRepresentativesLabelsHistorySettingsAndGraphLinks(t *testing.T) {
	docs := []parsing.ParsedDocument{
		{PaperID: "paper-1", Title: "Solar catalyst study", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Solar catalyst improves water splitting."}}}}},
		{PaperID: "paper-2", Title: "Screening bias study", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p2", PaperID: "paper-2", SectionID: "s1", Text: "Screening bias affects review selection."}}}}},
	}
	graph := NewGraph()
	graph.AddCitation("paper-1", "paper-2")
	graphData, err := graph.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	artifact, err := BuildDomainMapArtifact(docs, graphData, DomainMapOptions{
		ReviewerLabels:    map[string]string{"solar": "Reviewer solar fuels"},
		MergeSplitHistory: []TopicHistoryEvent{{Action: "merge", TopicIDs: []string{"solar", "catalyst"}, ResultTopicID: "solar-catalyst", Reviewer: "reviewer-a", Reason: "same concept"}},
		ModelSettings:     DomainMapModelSettings{Model: "bertopic-fixture", EmbeddingProvider: "deterministic-hash", MinTopicSize: 1},
	})
	if err != nil {
		t.Fatalf("BuildDomainMapArtifact returned error: %v", err)
	}
	if artifact.SchemaVersion != "1" || artifact.ModelSettings.Model != "bertopic-fixture" || len(artifact.Topics) == 0 || len(artifact.MergeSplitHistory) != 1 {
		t.Fatalf("artifact = %#v", artifact)
	}
	var solar *DomainTopic
	for i := range artifact.Topics {
		if artifact.Topics[i].TopicID == "solar" {
			solar = &artifact.Topics[i]
		}
	}
	if solar == nil {
		t.Fatalf("missing solar topic: %#v", artifact.Topics)
	}
	if solar.Label != "Reviewer solar fuels" || len(solar.RepresentativePapers) == 0 || len(solar.RepresentativePassages) == 0 || len(solar.CitationGraphLinks) == 0 {
		t.Fatalf("solar topic = %#v", solar)
	}
	if !strings.Contains(artifact.QuerySetChecksum, "sha256:") {
		t.Fatalf("checksum = %q", artifact.QuerySetChecksum)
	}
}

func TestBuildDomainMapArtifactRequiresParsedDocuments(t *testing.T) {
	_, err := BuildDomainMapArtifact(nil, nil, DomainMapOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
