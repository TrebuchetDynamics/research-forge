package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportZoteroRDFPreservesMetadataAndRedactsAttachments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zotero.rdf")
	fixture := `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:prism="http://prismstandard.org/namespaces/1.2/basic/" xmlns:bib="http://purl.org/net/biblio#" xmlns:z="http://www.zotero.org/namespaces/export#" xmlns:better-bibtex="https://retorque.re/zotero-better-bibtex/export#">
  <bib:Article rdf:about="#item-1">
    <dc:title>Zotero RDF fixture</dc:title>
    <prism:doi>https://doi.org/10.1000/RDF</prism:doi>
    <dcterms:abstract>RDF abstract.</dcterms:abstract>
    <prism:publicationName>RDF Journal</prism:publicationName>
    <dc:date>2026-06-13</dc:date>
    <better-bibtex:citekey>smith2026rdf</better-bibtex:citekey>
    <dc:subject>zotero</dc:subject>
    <dc:subject>rdf</dc:subject>
    <z:collection>Systematic reviews</z:collection>
    <z:collection>Crypto ML</z:collection>
    <z:note>Important note.</z:note>
    <z:annotation>Highlighted passage.</z:annotation>
    <z:annotation>Margin comment.</z:annotation>
    <z:attachment>/Users/alice/Zotero/storage/ABC123/paper.pdf</z:attachment>
  </bib:Article>
</rdf:RDF>`
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, skipped, err := ImportZoteroRDF(path)
	if err != nil {
		t.Fatalf("ImportZoteroRDF returned error: %v", err)
	}
	if skipped != 0 || len(records) != 1 {
		t.Fatalf("records=%d skipped=%d", len(records), skipped)
	}
	record := records[0]
	if record.Title != "Zotero RDF fixture" || record.Identifiers.DOI != "10.1000/rdf" || record.Year != 2026 || record.Venue != "RDF Journal" {
		t.Fatalf("record = %#v", record)
	}
	metadata := record.SourceRefs[0].Metadata
	if metadata["citation_key"] != "smith2026rdf" || metadata["tags"] != "zotero; rdf" || metadata["collections"] != "Systematic reviews; Crypto ML" || metadata["note"] != "Important note." || metadata["annotations"] != "Highlighted passage.\nMargin comment." {
		t.Fatalf("metadata = %#v", metadata)
	}
	if metadata["attachment_files"] != "paper.pdf" || strings.Contains(metadata["attachment_files"], "/Users/alice") {
		t.Fatalf("attachment_files not redacted: %#v", metadata["attachment_files"])
	}
}

func TestExportZoteroRDFWritesInteroperableSubset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.rdf")
	records := []PaperRecord{{
		Title:       "Exported Zotero RDF fixture",
		Identifiers: Identifiers{DOI: "10.1000/rdf-export"},
		Year:        2026,
		Venue:       "RDF Journal",
		Abstract:    "Export abstract.",
		SourceRefs:  []SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"zotero_rdf_id": "#item-1", "citation_key": "doe2026rdf", "tags": "zotero; rdf", "collections": "Systematic reviews", "note": "Export note", "annotations": "Export highlight\nExport margin comment"}}},
	}}
	if err := ExportZoteroRDF(path, records); err != nil {
		t.Fatalf("ExportZoteroRDF returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	text := string(data)
	for _, want := range []string{"<bib:Article", "<dc:title>Exported Zotero RDF fixture</dc:title>", "<prism:doi>10.1000/rdf-export</prism:doi>", "<better-bibtex:citekey>doe2026rdf</better-bibtex:citekey>", "<dc:subject>zotero</dc:subject>", "<z:collection>Systematic reviews</z:collection>", "<z:note>Export note</z:note>", "<z:annotation>Export highlight</z:annotation>", "<z:annotation>Export margin comment</z:annotation>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("export missing %q:\n%s", want, text)
		}
	}
}
