package library

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestImportJSONReadsPaperRecordsFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.json")
	fixture := `[
  {
    "Title": "Artificial photosynthesis import fixture",
    "Identifiers": {"DOI": "10.1000/import"},
    "Year": 2026
  }
]
`
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, _, err := ImportJSON(path)
	if err != nil {
		t.Fatalf("ImportJSON returned error: %v", err)
	}
	if len(records) != 1 || records[0].Title != "Artificial photosynthesis import fixture" || records[0].Identifiers.DOI != "10.1000/import" {
		t.Fatalf("records = %#v", records)
	}
}

func TestImportBibTeXReadsPaperRecordsFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.bib")
	fixture := `@article{ap2026,
  title = {Artificial photosynthesis BibTeX import},
  doi = {10.1000/bib-import},
  year = {2026},
  journal = {Journal of Fixtures},
  publisher = {Fixture Publisher}
}
`
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, _, err := ImportBibTeX(path)
	if err != nil {
		t.Fatalf("ImportBibTeX returned error: %v", err)
	}
	if len(records) != 1 || records[0].Title != "Artificial photosynthesis BibTeX import" || records[0].Identifiers.DOI != "10.1000/bib-import" || records[0].Venue != "Journal of Fixtures" {
		t.Fatalf("records = %#v", records)
	}
}

func TestExportBibTeXWritesGoldenPaperRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.bib")
	record, err := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis BibTeX export", Identifiers: Identifiers{DOI: "10.1000/bib-export"}, Year: 2026, Venue: "Journal of Fixtures"})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if err := ExportBibTeX(path, []PaperRecord{record}); err != nil {
		t.Fatalf("ExportBibTeX returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want := `@article{paper1,
  title = {Artificial photosynthesis BibTeX export},
  doi = {10.1000/bib-export},
  year = {2026},
  journal = {Journal of Fixtures}
}
`
	if string(data) != want {
		t.Fatalf("export mismatch:\n%s", string(data))
	}
}

func TestImportRISReadsPaperRecordsFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.ris")
	fixture := "TY  - JOUR\nTI  - Artificial photosynthesis RIS import\nDO  - 10.1000/ris-import\nPY  - 2026\nJO  - Journal of Fixtures\nER  - \n"
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	records, _, err := ImportRIS(path)
	if err != nil {
		t.Fatalf("ImportRIS returned error: %v", err)
	}
	if len(records) != 1 || records[0].Title != "Artificial photosynthesis RIS import" || records[0].Identifiers.DOI != "10.1000/ris-import" || records[0].Venue != "Journal of Fixtures" {
		t.Fatalf("records = %#v", records)
	}
}

func TestExportRISWritesGoldenPaperRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.ris")
	record, err := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis RIS export", Identifiers: Identifiers{DOI: "10.1000/ris-export"}, Year: 2026, Venue: "Journal of Fixtures"})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}
	if err := ExportRIS(path, []PaperRecord{record}); err != nil {
		t.Fatalf("ExportRIS returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want := "TY  - JOUR\nTI  - Artificial photosynthesis RIS export\nDO  - 10.1000/ris-export\nPY  - 2026\nJO  - Journal of Fixtures\nER  - \n"
	if string(data) != want {
		t.Fatalf("export mismatch:\n%s", string(data))
	}
}

func TestImportCSVReadsPaperRecordsFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.csv")
	fixture := "title,doi,year\nArtificial photosynthesis CSV import,10.1000/csv-import,2026\n"
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, _, err := ImportCSV(path)
	if err != nil {
		t.Fatalf("ImportCSV returned error: %v", err)
	}
	if len(records) != 1 || records[0].Title != "Artificial photosynthesis CSV import" || records[0].Identifiers.DOI != "10.1000/csv-import" || records[0].Year != 2026 {
		t.Fatalf("records = %#v", records)
	}
}

func TestExportCSVWritesGoldenPaperRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.csv")
	record, err := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis CSV export", Identifiers: Identifiers{DOI: "10.1000/csv-export"}, Year: 2026})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}

	if err := ExportCSV(path, []PaperRecord{record}); err != nil {
		t.Fatalf("ExportCSV returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want := "title,doi,arxiv_id,pmid,pmcid,openalex_id,crossref_id,semantic_scholar_id,year,abstract,venue,publisher,license,open_access\nArtificial photosynthesis CSV export,10.1000/csv-export,,,,,,,2026,,,,,false\n"
	if string(data) != want {
		t.Fatalf("export mismatch:\n%s", string(data))
	}
}

func TestLibraryExportsDoNotWriteThroughSymlinkedDestinations(t *testing.T) {
	exporters := []struct {
		name   string
		export func(string, []PaperRecord) error
	}{
		{name: "csv", export: ExportCSV},
		{name: "bibtex", export: ExportBibTeX},
		{name: "ris", export: ExportRIS},
		{name: "json", export: ExportJSON},
		{name: "csl-json", export: ExportCSLJSON},
		{name: "zotero-rdf", export: ExportZoteroRDF},
	}
	for _, tc := range exporters {
		t.Run(tc.name, func(t *testing.T) {
			outsidePath := filepath.Join(t.TempDir(), "outside-export")
			outsideBefore := []byte("outside export\n")
			if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
				t.Fatalf("write outside export: %v", err)
			}
			exportPath := filepath.Join(t.TempDir(), "export."+tc.name)
			if err := os.Symlink(outsidePath, exportPath); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}

			err := tc.export(exportPath, []PaperRecord{{Title: "Replacement export"}})
			if err == nil {
				t.Errorf("export succeeded through symlink, want error")
			}
			outsideAfter, readErr := os.ReadFile(outsidePath)
			if readErr != nil {
				t.Fatalf("read outside export: %v", readErr)
			}
			if !bytes.Equal(outsideAfter, outsideBefore) {
				t.Errorf("export wrote through symlink:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
			}
			info, statErr := os.Stat(outsidePath)
			if statErr != nil {
				t.Fatalf("stat outside export: %v", statErr)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Errorf("outside export mode = %o, want 600", got)
			}
		})
	}
}

func TestExportJSONWritesGoldenPaperRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "papers.json")
	record, err := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis export fixture", Identifiers: Identifiers{DOI: "10.1000/export"}, Year: 2026})
	if err != nil {
		t.Fatalf("NewPaperRecord returned error: %v", err)
	}

	if err := ExportJSON(path, []PaperRecord{record}); err != nil {
		t.Fatalf("ExportJSON returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want := `[
  {
    "Title": "Artificial photosynthesis export fixture",
    "Identifiers": {
      "DOI": "10.1000/export",
      "ArXivID": "",
      "PMID": "",
      "PMCID": "",
      "OpenAlexID": "",
      "CrossrefID": "",
      "SemanticScholarID": "",
      "ZoteroItemKey": "",
      "ADSBibcode": ""
    },
    "Authors": [],
    "Abstract": "",
    "Year": 2026,
    "Venue": "",
    "Publisher": "",
    "URLs": [],
    "License": "",
    "OpenAccess": false,
    "SourcePayload": "",
    "SourceRefs": []
  }
]
`
	if string(data) != want {
		t.Fatalf("export mismatch:\n%s", string(data))
	}
}
