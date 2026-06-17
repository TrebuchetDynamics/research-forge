package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecutePDFImportBiomedicalAndDriftPlan(t *testing.T) {
	project := t.TempDir()
	xmlPath := filepath.Join(project, "pmc.xml")
	if err := os.WriteFile(xmlPath, []byte(`<article><front><article-meta><article-id pub-id-type="pmid">123</article-id><article-id pub-id-type="pmc">PMC456</article-id><title-group><article-title>Biomedical CLI</article-title></title-group></article-meta></front><body><sec><title>Results</title><p>Result text.</p></sec></body></article>`), 0o644); err != nil {
		t.Fatalf("write xml: %v", err)
	}
	out := filepath.Join(project, "data", "biomedical-fulltext", "pmc456.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "pdf", "import-biomedical", "--xml", xmlPath, "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import-biomedical code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var fullText documents.BiomedicalFullText
	if err := json.Unmarshal(data, &fullText); err != nil {
		t.Fatalf("decode fulltext: %v", err)
	}
	if fullText.PMID != "123" || fullText.PMCID != "PMC456" || len(fullText.Sections) != 1 {
		t.Fatalf("fullText = %#v", fullText)
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "pdf", "biomedical-drift-smoke-plan"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("drift plan code=%d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("europepmc")) || !bytes.Contains(stdout.Bytes(), []byte("pubmed")) {
		t.Fatalf("drift stdout=%s", stdout.String())
	}
}

func TestExecuteLibraryPMCIDPMIDLinksJSON(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	record, err := library.NewPaperRecord(library.PaperRecordInput{Title: "Biomedical link", Identifiers: library.Identifiers{PMID: "123", PMCID: "456"}})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, err := store.ImportRecords([]library.PaperRecord{record}); err != nil {
		t.Fatalf("import: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "library", "pmcid-pmid-links"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("links code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("PMC456")) || !bytes.Contains(stdout.Bytes(), []byte("123")) {
		t.Fatalf("links stdout=%s", stdout.String())
	}
}
