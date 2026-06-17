package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/citations"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteCitationsImportBibliographyParsedDirImportsFullGraph(t *testing.T) {
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc1 := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref1", DOI: "10.1000/ref1"}}, Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Known [1]."}}}}})
	doc2 := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-2", References: []parsing.Reference{{Title: "Ref2", DOI: "10.1000/ref2"}}, Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p2", PaperID: "paper-2", SectionID: "s1", Text: "Known [1]."}}}}})
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-1.json"), doc1)
	writeParsedFixture(t, filepath.Join(parsedDir, "paper-2.json"), doc2)
	graphPath := filepath.Join(project, "data", "citation-graph.json")
	reportPath := filepath.Join(project, "data", "bibliography-import.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "import-bibliography", "--parsed-dir", parsedDir, "--out", graphPath, "--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report citations.BibliographyImportReport
	if err := readJSONFile(reportPath, &report); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if report.EdgeCount != 2 || len(report.DocumentReports) != 2 {
		t.Fatalf("report = %#v", report)
	}
}

func TestExecuteCitationsImportBibliographyLinksSpansAndEvidence(t *testing.T) {
	project := t.TempDir()
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref", DOI: "10.1000/ref"}}, Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Known [1]."}}}}})
	parsedPath := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsedPath, doc)
	items := []evidence.EvidenceItem{{PaperID: "paper-1", Support: evidence.Support{Kind: evidence.SupportPassage, Ref: "p1"}, Status: evidence.StatusAccepted}}
	if err := writeJSONFile(evidenceItemsPath(project), items); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	graphPath := filepath.Join(project, "data", "citation-graph.json")
	reportPath := filepath.Join(project, "data", "bibliography-import.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "import-bibliography", "--parsed", parsedPath, "--out", graphPath, "--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report citations.BibliographyImportReport
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.EdgeCount != 1 || len(report.CitationSpanLinks) != 1 || len(report.EvidenceLinks) != 1 {
		t.Fatalf("report = %#v", report)
	}
	graphData, err := os.ReadFile(graphPath)
	if err != nil || !bytes.Contains(graphData, []byte("doi:10.1000/ref")) {
		t.Fatalf("graph err=%v data=%s", err, string(graphData))
	}
}
