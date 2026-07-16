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

func TestExecuteCitationsImportBibliographyRejectsMalformedEvidence(t *testing.T) {
	project := t.TempDir()
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref", DOI: "10.1000/ref"}}})
	parsedPath := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsedPath, doc)
	if err := os.MkdirAll(filepath.Dir(evidenceItemsPath(project)), 0o755); err != nil {
		t.Fatalf("mkdir evidence directory: %v", err)
	}
	if err := os.WriteFile(evidenceItemsPath(project), []byte(`[{"PaperID":`), 0o644); err != nil {
		t.Fatalf("write malformed evidence: %v", err)
	}
	graphPath := filepath.Join(project, "data", "citation-graph.json")
	reportPath := filepath.Join(project, "data", "bibliography-import.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "import-bibliography", "--parsed", parsedPath, "--out", graphPath, "--report", reportPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"code":"citation_evidence_read_failed"`)) {
		t.Fatalf("missing evidence read error: %s", stdout.String())
	}
	for _, path := range []string{graphPath, reportPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("citation import wrote %s after evidence read failure: %v", path, err)
		}
	}
}

func TestExecuteCitationsImportBibliographyRejectsSharedOutputPathWithoutReplacingIt(t *testing.T) {
	project := t.TempDir()
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref", DOI: "10.1000/ref"}}})
	parsedPath := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsedPath, doc)
	sharedPath := filepath.Join(project, "data", "citation-output.json")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("mkdir output directory: %v", err)
	}
	prior := []byte("prior output\n")
	if err := os.WriteFile(sharedPath, prior, 0o644); err != nil {
		t.Fatalf("seed output: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "import-bibliography", "--parsed", parsedPath, "--out", sharedPath, "--report", sharedPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(sharedPath)
	if err != nil {
		t.Fatalf("read prior output: %v", err)
	}
	if !bytes.Equal(data, prior) {
		t.Fatalf("shared output changed after rejected import: got %q want %q", data, prior)
	}
}

func TestExecuteCitationsImportBibliographyRestoresBothOutputsAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", References: []parsing.Reference{{Title: "Ref", DOI: "10.1000/ref"}}})
	parsedPath := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsedPath, doc)
	graphPath := filepath.Join(project, "data", "citation-graph.json")
	reportPath := filepath.Join(project, "data", "bibliography-import.json")
	if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
		t.Fatalf("mkdir output directory: %v", err)
	}
	priorGraph := []byte("prior graph\n")
	priorReport := []byte("prior report\n")
	if err := os.WriteFile(graphPath, priorGraph, 0o600); err != nil {
		t.Fatalf("seed graph: %v", err)
	}
	if err := os.WriteFile(reportPath, priorReport, 0o640); err != nil {
		t.Fatalf("seed report: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "import-bibliography", "--parsed", parsedPath, "--out", graphPath, "--report", reportPath}, &stdout, &stderr)
	if code != 1 || !bytes.Contains(stdout.Bytes(), []byte(`"code":"citation_bibliography_provenance_failed"`)) {
		t.Fatalf("code=%d, want provenance failure; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	assertFileRestored(t, graphPath, priorGraph)
	assertFileRestored(t, reportPath, priorReport)
	for path, want := range map[string]os.FileMode{graphPath: 0o600, reportPath: 0o640} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat restored output %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Fatalf("restored output mode for %s = %o, want %o", path, got, want)
		}
	}
}
