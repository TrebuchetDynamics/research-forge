package documents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestLinkPMCIDPMIDBuildsBidirectionalIdentifierLinks(t *testing.T) {
	records := []library.PaperRecord{
		{Title: "PubMed", Identifiers: library.Identifiers{PMID: "123", PMCID: "PMC456", DOI: "10.1000/pmc"}},
		{Title: "PMC", Identifiers: library.Identifiers{PMCID: "456"}},
	}
	links := LinkPMCIDPMID(records)
	if len(links) != 2 {
		t.Fatalf("links = %#v", links)
	}
	if links[0].PMID != "123" || links[0].PMCID != "PMC456" || links[0].DOI != "10.1000/pmc" {
		t.Fatalf("first link = %#v", links[0])
	}
	if links[1].PMCID != "PMC456" {
		t.Fatalf("PMCID normalization missing: %#v", links[1])
	}
}

func TestImportStructuredBiomedicalFullTextExtractsSectionsAndSupplements(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pmc.xml")
	fixture := []byte(`<article><front><article-meta><article-id pub-id-type="pmid">123</article-id><article-id pub-id-type="pmc">PMC456</article-id><article-id pub-id-type="doi">10.1000/pmc</article-id><title-group><article-title>Biomedical fixture</article-title></title-group><abstract><p>Abstract text.</p></abstract></article-meta></front><body><sec><title>Methods</title><p>Method text.</p></sec><sec><title>Results</title><p>Result text.</p></sec></body><back><sec sec-type="supplementary-material"><title>Supplementary data</title><supplementary-material id="s1" xlink:href="supp1.xlsx" xmlns:xlink="http://www.w3.org/1999/xlink"><label>Table S1</label></supplementary-material></sec></back></article>`)
	if err := os.WriteFile(path, fixture, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	fullText, err := ImportStructuredBiomedicalFullText(path)
	if err != nil {
		t.Fatalf("ImportStructuredBiomedicalFullText: %v", err)
	}
	if fullText.PMID != "123" || fullText.PMCID != "PMC456" || fullText.DOI != "10.1000/pmc" || fullText.Title != "Biomedical fixture" {
		t.Fatalf("metadata = %#v", fullText)
	}
	if len(fullText.Sections) != 3 || fullText.Sections[1].Title != "Methods" || fullText.Sections[2].Text != "Result text." {
		t.Fatalf("sections = %#v", fullText.Sections)
	}
	if len(fullText.SupplementaryFiles) != 1 || fullText.SupplementaryFiles[0].Href != "supp1.xlsx" || fullText.SupplementaryFiles[0].Label != "Table S1" {
		t.Fatalf("supplements = %#v", fullText.SupplementaryFiles)
	}
}

func TestBiomedicalLiveDriftSmokeSnapshotPlansExpectedFields(t *testing.T) {
	snapshot := NewBiomedicalLiveDriftSmokeSnapshot()
	if len(snapshot.Connectors) != 2 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	for _, connector := range snapshot.Connectors {
		if connector.Source == "" || len(connector.ExpectedFields) == 0 || connector.OptInEnv != "RFORGE_RUN_LIVE_SOURCE_SMOKE=1" {
			t.Fatalf("bad connector drift smoke plan: %#v", connector)
		}
	}
}
