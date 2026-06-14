package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportCSLJSONNormalizesZoteroExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zotero-csl.json")
	fixture := `[
  {
    "id": "smith2026crypto",
    "type": "article-journal",
    "title": "Leak-free LightGBM for crypto price data",
    "DOI": "https://doi.org/10.1000/CSL",
    "abstract": "Training guidance.",
    "container-title": "Journal of Financial ML",
    "publisher": "Research Press",
    "URL": "https://example.org/paper",
    "citation-key": "smith2026crypto",
    "note": "Reviewer note with annotation summary.",
    "keyword": "crypto; leakage; LightGBM",
    "attachments": [{"title":"PDF", "path":"/Users/alice/Zotero/storage/ABC123/paper.pdf"}],
    "issued": {"date-parts": [[2026, 6, 13]]},
    "author": [{"given": "Jane", "family": "Smith"}]
  },
  {"id": "missing-id", "title": "No identifier"}
]`
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	records, skipped, err := ImportCSLJSON(path)
	if err != nil {
		t.Fatalf("ImportCSLJSON returned error: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	record := records[0]
	if record.Title != "Leak-free LightGBM for crypto price data" || record.Identifiers.DOI != "10.1000/csl" || record.Year != 2026 {
		t.Fatalf("record normalization failed: %#v", record)
	}
	if len(record.Authors) != 1 || record.Authors[0].Family != "Smith" || record.Authors[0].Given != "Jane" {
		t.Fatalf("authors = %#v", record.Authors)
	}
	if len(record.SourceRefs) != 1 || record.SourceRefs[0].Source != "csl-json" || record.SourceRefs[0].Metadata["csl_id"] != "smith2026crypto" || record.SourceRefs[0].Metadata["citation_key"] != "smith2026crypto" {
		t.Fatalf("source refs = %#v", record.SourceRefs)
	}
	metadata := record.SourceRefs[0].Metadata
	if metadata["note"] != "Reviewer note with annotation summary." || metadata["tags"] != "crypto; leakage; LightGBM" {
		t.Fatalf("zotero note/tags metadata = %#v", metadata)
	}
	if metadata["attachment_files"] != "paper.pdf" || strings.Contains(metadata["attachment_files"], "/Users/alice") {
		t.Fatalf("attachment path not privacy-redacted: %#v", metadata["attachment_files"])
	}
}

func TestExportCSLJSONPreservesBetterBibTeXCitationKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "better-bibtex.csl.json")
	records := []PaperRecord{{
		Title:       "Better BibTeX citation key fixture",
		Identifiers: Identifiers{DOI: "10.1000/bbt"},
		SourceRefs:  []SourceRef{{Source: "better-bibtex", Metadata: map[string]string{"citation_key": "doe2026bbt"}}},
	}}
	if err := ExportCSLJSON(path, records); err != nil {
		t.Fatalf("ExportCSLJSON returned error: %v", err)
	}
	var exported []map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("export is not JSON: %v", err)
	}
	if exported[0]["citation-key"] != "doe2026bbt" {
		t.Fatalf("citation-key = %#v", exported[0]["citation-key"])
	}
}

func TestExportCSLJSONWritesZoteroCompatibleRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.csl.json")
	records := []PaperRecord{{
		Title:       "Leak-free LightGBM for crypto price data",
		Identifiers: Identifiers{DOI: "10.1000/csl"},
		Authors:     []Author{{Given: "Jane", Family: "Smith"}},
		Year:        2026,
		Venue:       "Journal of Financial ML",
		Publisher:   "Research Press",
		Abstract:    "Training guidance.",
		URLs:        []string{"https://example.org/paper"},
		SourceRefs:  []SourceRef{{Source: "csl-json", Metadata: map[string]string{"csl_id": "smith2026crypto", "citation_key": "smith2026crypto", "note": "Reviewer note", "tags": "crypto; leakage"}}},
	}}
	if err := ExportCSLJSON(path, records); err != nil {
		t.Fatalf("ExportCSLJSON returned error: %v", err)
	}
	var exported []map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("export is not JSON: %v\n%s", err, data)
	}
	if len(exported) != 1 {
		t.Fatalf("exported len = %d, want 1", len(exported))
	}
	if exported[0]["id"] != "smith2026crypto" || exported[0]["citation-key"] != "smith2026crypto" || exported[0]["note"] != "Reviewer note" || exported[0]["keyword"] != "crypto; leakage" || exported[0]["DOI"] != "10.1000/csl" || exported[0]["container-title"] != "Journal of Financial ML" {
		t.Fatalf("exported record = %#v", exported[0])
	}
	issued := exported[0]["issued"].(map[string]any)["date-parts"].([]any)[0].([]any)[0].(float64)
	if issued != 2026 {
		t.Fatalf("issued year = %v, want 2026", issued)
	}
}
