package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportBibTeXPreservesJabRefFidelityAndPrivacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jabref.bib")
	fixture := `@article{Smith2026Catalysts,
  title = {Catalyst fidelity fixture},
  doi = {HTTPS://DOI.ORG/10.1000/JABREF},
  year = {2026},
  journal = {JabRef Journal},
  keywords = {hydrogen; catalysis},
  groups = {Screened/In; Included},
  note = {Private reviewer note},
  annote = {Highlighted passage},
  file = {:C\\Users\\alice\\Zotero\\storage\\ABC123\\paper.pdf:PDF}
}
`
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	records, skipped, err := ImportBibTeX(path)
	if err != nil {
		t.Fatalf("ImportBibTeX: %v", err)
	}
	if skipped != 0 || len(records) != 1 {
		t.Fatalf("records=%d skipped=%d", len(records), skipped)
	}
	metadata := records[0].SourceRefs[0].Metadata
	want := map[string]string{
		"citation_key":              "Smith2026Catalysts",
		"tags":                      "hydrogen; catalysis",
		"groups":                    "Screened/In; Included",
		"note":                      "Private reviewer note",
		"annotations":               "Highlighted passage",
		"attachment_files":          "paper.pdf",
		"linked_file_privacy_check": "redacted-local-paths",
	}
	for key, value := range want {
		if metadata[key] != value {
			t.Fatalf("metadata[%s] = %q, want %q; metadata=%#v", key, metadata[key], value, metadata)
		}
	}
	if !strings.Contains(metadata["cleanup_diff"], "doi: HTTPS://DOI.ORG/10.1000/JABREF -> 10.1000/jabref") {
		t.Fatalf("cleanup_diff missing DOI normalization: %#v", metadata["cleanup_diff"])
	}
	if strings.Contains(metadata["attachment_files"], "alice") || strings.Contains(metadata["attachment_files"], "C:") {
		t.Fatalf("attachment path leaked: %#v", metadata["attachment_files"])
	}
}

func TestExportBibTeXPreservesCitationKeyTagsAndGroups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.bib")
	record := PaperRecord{Title: "JabRef export fixture", Identifiers: Identifiers{DOI: "10.1000/export"}, Year: 2026, Venue: "Export Journal", SourceRefs: []SourceRef{{Source: "jabref-bibtex", Metadata: map[string]string{"citation_key": "doe2026export", "tags": "hydrogen; catalysis", "groups": "Included", "note": "Export note", "annotations": "Export annotation", "attachment_files": "paper.pdf"}}}}
	if err := ExportBibTeX(path, []PaperRecord{record}); err != nil {
		t.Fatalf("ExportBibTeX: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	text := string(data)
	for _, want := range []string{"@article{doe2026export,", "keywords = {hydrogen; catalysis}", "groups = {Included}", "note = {Export note}", "annote = {Export annotation}", "file = {:paper.pdf:PDF}"} {
		if !strings.Contains(text, want) {
			t.Fatalf("export missing %q:\n%s", want, text)
		}
	}
}

func TestReferenceManagerFidelityReportCoversZoteroAndJabRefFields(t *testing.T) {
	record := PaperRecord{Title: "Fidelity fixture", Identifiers: Identifiers{DOI: "10.1000/fidelity"}, SourceRefs: []SourceRef{{Source: "zotero-rdf", Metadata: map[string]string{"collections": "Reviews", "tags": "tag", "note": "note", "annotations": "annotation", "citation_key": "key", "attachment_files": "paper.pdf", "linked_file_privacy_check": "redacted-local-paths"}}, {Source: "jabref-bibtex", Metadata: map[string]string{"groups": "Included", "cleanup_diff": "doi normalized"}}}}
	report := ReferenceManagerFidelityReport([]PaperRecord{record})
	if len(report.Records) != 1 {
		t.Fatalf("report records = %#v", report.Records)
	}
	entry := report.Records[0]
	for _, field := range []string{"collections", "groups", "tags", "notes", "annotations", "citation_keys", "bibtex_cleanup_diffs", "linked_file_privacy_checks"} {
		if !entry.Fields[field] {
			t.Fatalf("field %s not covered: %#v", field, entry.Fields)
		}
	}
}
