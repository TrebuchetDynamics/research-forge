package citations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestImportParsedBibliographiesImportsFullGraphAcrossDocuments(t *testing.T) {
	doc1 := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref One", DOI: "10.1000/ref1"}}, Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Prior work [1]."}}}}})
	doc2 := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-2", References: []parsing.Reference{{Title: "Ref Two", DOI: "10.1000/ref2"}}, Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p2", PaperID: "paper-2", SectionID: "s1", Text: "Later work [1]."}}}}})
	items := []evidence.EvidenceItem{{PaperID: "paper-2", SchemaName: "outcome", Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "p2"}, Status: evidence.StatusAccepted}}
	report := ImportParsedBibliographies([]parsing.ParsedDocument{doc1, doc2}, items)
	if report.EdgeCount != 2 || len(report.DocumentReports) != 2 || len(report.CitationSpanLinks) != 2 || len(report.EvidenceLinks) != 1 {
		t.Fatalf("multi report = %#v", report)
	}
	data, err := report.Graph.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	if !strings.Contains(string(data), "paper-1") || !strings.Contains(string(data), "paper-2") || !strings.Contains(string(data), "doi:10.1000/ref2") {
		t.Fatalf("graph = %s", string(data))
	}
}

func TestImportParsedBibliographyAddsGraphEdgesAndLinksSpansToEvidence(t *testing.T) {
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref One", DOI: "10.1000/ref1"}}, Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Prior work [1] shows effects."}}}}})
	items := []evidence.EvidenceItem{{PaperID: "paper-1", SchemaName: "outcome", Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "p1"}, Status: evidence.StatusAccepted}}
	report := ImportParsedBibliography(doc, items)
	if report.PaperID != "paper-1" || report.EdgeCount != 1 || len(report.Edges) != 1 || report.Edges[0].TargetID != "doi:10.1000/ref1" {
		t.Fatalf("report edges = %#v", report)
	}
	if len(report.CitationSpanLinks) != 1 || report.CitationSpanLinks[0].PassageID != "p1" || report.CitationSpanLinks[0].TargetID != "doi:10.1000/ref1" {
		t.Fatalf("span links = %#v", report.CitationSpanLinks)
	}
	if len(report.EvidenceLinks) != 1 || report.EvidenceLinks[0].EvidenceSupportRef != "p1" || report.EvidenceLinks[0].CitationSpanID == "" {
		t.Fatalf("evidence links = %#v", report.EvidenceLinks)
	}
	data, err := report.Graph.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	var exported struct {
		Edges []struct{ Source, Target string }
	}
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("decode graph: %v", err)
	}
	if len(exported.Edges) != 1 || exported.Edges[0].Source != "paper-1" || exported.Edges[0].Target != "doi:10.1000/ref1" {
		t.Fatalf("exported graph = %s", string(data))
	}
}
