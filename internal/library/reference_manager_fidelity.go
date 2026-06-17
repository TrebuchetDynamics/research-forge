package library

import (
	"os"
	"path/filepath"
)

// ReferenceManagerFidelityReport summarizes whether imported reference-manager
// records retained review-relevant fields from Zotero/JabRef style sources.
type ReferenceManagerFidelitySummary struct {
	SchemaVersion string                          `json:"schemaVersion"`
	Records       []ReferenceManagerFidelityEntry `json:"records"`
}

type ReferenceManagerFidelityEntry struct {
	Title  string          `json:"title"`
	DOI    string          `json:"doi,omitempty"`
	Fields map[string]bool `json:"fields"`
}

func ReferenceManagerFidelityReport(records []PaperRecord) ReferenceManagerFidelitySummary {
	entries := make([]ReferenceManagerFidelityEntry, 0, len(records))
	for _, record := range records {
		entry := ReferenceManagerFidelityEntry{Title: record.Title, DOI: record.Identifiers.DOI, Fields: map[string]bool{
			"collections":                metadataAny(record, "collections"),
			"groups":                     metadataAny(record, "groups"),
			"tags":                       metadataAny(record, "tags"),
			"notes":                      metadataAny(record, "note"),
			"annotations":                metadataAny(record, "annotations"),
			"citation_keys":              metadataAny(record, "citation_key"),
			"bibtex_cleanup_diffs":       metadataAny(record, "cleanup_diff"),
			"linked_file_privacy_checks": metadataAny(record, "linked_file_privacy_check") || metadataAny(record, "attachment_files"),
		}}
		entries = append(entries, entry)
	}
	return ReferenceManagerFidelitySummary{SchemaVersion: "1", Records: entries}
}

const (
	FidelitySupported   = "supported"
	FidelityPartial     = "partial"
	FidelityRedacted    = "redacted"
	FidelityUnsupported = "unsupported"
)

type ReferenceManagerInterchangeMatrix struct {
	SchemaVersion string                           `json:"schemaVersion"`
	RecordCount   int                              `json:"recordCount"`
	FieldsPresent map[string]int                   `json:"fieldsPresent,omitempty"`
	Formats       []ReferenceManagerFormatFidelity `json:"formats"`
}

type ReferenceManagerFormatFidelity struct {
	Format string                   `json:"format"`
	Label  string                   `json:"label"`
	Fields map[string]FieldFidelity `json:"fields"`
}

type FieldFidelity struct {
	Status    string `json:"status"`
	Note      string `json:"note"`
	Preserved int    `json:"preserved,omitempty"`
	Lost      int    `json:"lost,omitempty"`
}

func DefaultReferenceManagerInterchangeMatrix() ReferenceManagerInterchangeMatrix {
	return ReferenceManagerInterchangeMatrix{SchemaVersion: "1", FieldsPresent: map[string]int{}, Formats: []ReferenceManagerFormatFidelity{
		{Format: "bibtex", Label: "BibTeX / JabRef", Fields: map[string]FieldFidelity{
			"core_metadata":              {Status: FidelitySupported, Note: "title, DOI, year, journal"},
			"better_bibtex_citation_key": {Status: FidelitySupported, Note: "entry key is preserved as citation_key"},
			"tags":                       {Status: FidelitySupported, Note: "keywords field"},
			"notes":                      {Status: FidelitySupported, Note: "note field"},
			"annotations":                {Status: FidelitySupported, Note: "annote field"},
			"collections":                {Status: FidelityPartial, Note: "JabRef groups preserved as collections/groups context"},
			"bibtex_cleanup_diffs":       {Status: FidelitySupported, Note: "normalization diffs recorded in source metadata"},
			"redacted_attachments":       {Status: FidelityRedacted, Note: "file paths are reduced to basenames with privacy check metadata"},
		}},
		{Format: "ris", Label: "RIS", Fields: map[string]FieldFidelity{
			"core_metadata":              {Status: FidelitySupported, Note: "title, DOI, year, journal"},
			"better_bibtex_citation_key": {Status: FidelityUnsupported, Note: "RIS has no Better BibTeX citation key convention in current adapter"},
			"tags":                       {Status: FidelityUnsupported, Note: "not preserved by current minimal RIS adapter"},
			"notes":                      {Status: FidelityUnsupported, Note: "not preserved by current minimal RIS adapter"},
			"annotations":                {Status: FidelityUnsupported, Note: "not preserved by current minimal RIS adapter"},
			"collections":                {Status: FidelityUnsupported, Note: "not preserved by current minimal RIS adapter"},
			"bibtex_cleanup_diffs":       {Status: FidelityUnsupported, Note: "BibTeX-specific"},
			"redacted_attachments":       {Status: FidelityUnsupported, Note: "not imported by current minimal RIS adapter"},
		}},
		{Format: "csl-json", Label: "CSL-JSON / Zotero export", Fields: map[string]FieldFidelity{
			"core_metadata":              {Status: FidelitySupported, Note: "title, DOI, authors, abstract, issued, venue, publisher, URL"},
			"better_bibtex_citation_key": {Status: FidelitySupported, Note: "citation-key field"},
			"tags":                       {Status: FidelitySupported, Note: "keyword field"},
			"notes":                      {Status: FidelitySupported, Note: "note field"},
			"annotations":                {Status: FidelityUnsupported, Note: "not represented in current CSL-JSON adapter"},
			"collections":                {Status: FidelityUnsupported, Note: "not represented in CSL-JSON item records"},
			"bibtex_cleanup_diffs":       {Status: FidelityUnsupported, Note: "BibTeX-specific"},
			"redacted_attachments":       {Status: FidelityRedacted, Note: "attachment paths are reduced to basenames"},
		}},
		{Format: "zotero-rdf", Label: "Zotero RDF", Fields: map[string]FieldFidelity{
			"core_metadata":              {Status: FidelitySupported, Note: "title, DOI, abstract, date, publication, publisher"},
			"better_bibtex_citation_key": {Status: FidelitySupported, Note: "better-bibtex:citekey"},
			"tags":                       {Status: FidelitySupported, Note: "dc:subject"},
			"notes":                      {Status: FidelitySupported, Note: "z:note"},
			"annotations":                {Status: FidelitySupported, Note: "z:annotation"},
			"collections":                {Status: FidelitySupported, Note: "z:collection"},
			"bibtex_cleanup_diffs":       {Status: FidelityUnsupported, Note: "BibTeX-specific"},
			"redacted_attachments":       {Status: FidelityRedacted, Note: "z:attachment paths are reduced to basenames with privacy check metadata"},
		}},
	}}
}

func BuildReferenceManagerInterchangeMatrix(records []PaperRecord) ReferenceManagerInterchangeMatrix {
	matrix := DefaultReferenceManagerInterchangeMatrix()
	matrix.RecordCount = len(records)
	matrix.FieldsPresent = map[string]int{
		"better_bibtex_citation_key": 0,
		"tags":                       0,
		"notes":                      0,
		"annotations":                0,
		"collections":                0,
		"groups":                     0,
		"bibtex_cleanup_diffs":       0,
		"redacted_attachments":       0,
	}
	for _, record := range records {
		if metadataAny(record, "citation_key") {
			matrix.FieldsPresent["better_bibtex_citation_key"]++
		}
		if metadataAny(record, "tags") {
			matrix.FieldsPresent["tags"]++
		}
		if metadataAny(record, "note") {
			matrix.FieldsPresent["notes"]++
		}
		if metadataAny(record, "annotations") {
			matrix.FieldsPresent["annotations"]++
		}
		if metadataAny(record, "collections") {
			matrix.FieldsPresent["collections"]++
		}
		if metadataAny(record, "groups") {
			matrix.FieldsPresent["groups"]++
		}
		if metadataAny(record, "cleanup_diff") {
			matrix.FieldsPresent["bibtex_cleanup_diffs"]++
		}
		if metadataAny(record, "attachment_files") || metadataAny(record, "linked_file_privacy_check") {
			matrix.FieldsPresent["redacted_attachments"]++
		}
	}
	return matrix
}

func BuildReferenceManagerRoundTripMatrix(records []PaperRecord) ReferenceManagerInterchangeMatrix {
	matrix := BuildReferenceManagerInterchangeMatrix(records)
	dir, err := os.MkdirTemp("", "rforge-refman-roundtrip-*")
	if err != nil {
		return matrix
	}
	defer os.RemoveAll(dir)
	for i := range matrix.Formats {
		format := matrix.Formats[i].Format
		path := filepath.Join(dir, "records."+format)
		var imported []PaperRecord
		if exportReferenceManager(format, path, records) == nil {
			imported, _, _ = importReferenceManager(format, path)
		}
		for field, fidelity := range matrix.Formats[i].Fields {
			present, preserved := 0, 0
			for _, record := range records {
				if !recordHasRoundTripField(record, field) {
					continue
				}
				present++
				if anyRecordHasRoundTripField(imported, field) {
					preserved++
				}
			}
			fidelity.Preserved = preserved
			if present > preserved {
				fidelity.Lost = present - preserved
			}
			if fidelity.Lost > 0 && fidelity.Status == FidelitySupported {
				fidelity.Status = FidelityPartial
			}
			matrix.Formats[i].Fields[field] = fidelity
		}
	}
	return matrix
}

func exportReferenceManager(format, path string, records []PaperRecord) error {
	switch format {
	case "bibtex":
		return ExportBibTeX(path, records)
	case "ris":
		return ExportRIS(path, records)
	case "csl-json":
		return ExportCSLJSON(path, records)
	case "zotero-rdf":
		return ExportZoteroRDF(path, records)
	default:
		return nil
	}
}
func importReferenceManager(format, path string) ([]PaperRecord, int, error) {
	switch format {
	case "bibtex":
		return ImportBibTeX(path)
	case "ris":
		return ImportRIS(path)
	case "csl-json":
		return ImportCSLJSON(path)
	case "zotero-rdf":
		return ImportZoteroRDF(path)
	default:
		return nil, 0, nil
	}
}
func recordHasRoundTripField(record PaperRecord, field string) bool {
	switch field {
	case "core_metadata":
		return record.Title != "" || record.Identifiers.DOI != ""
	case "better_bibtex_citation_key":
		return metadataAny(record, "citation_key")
	case "tags":
		return metadataAny(record, "tags")
	case "notes":
		return metadataAny(record, "note")
	case "annotations":
		return metadataAny(record, "annotations")
	case "collections":
		return metadataAny(record, "collections") || metadataAny(record, "groups") || metadataAny(record, "collection_hierarchy")
	case "bibtex_cleanup_diffs":
		return metadataAny(record, "cleanup_diff")
	case "redacted_attachments":
		return metadataAny(record, "attachment_files") || metadataAny(record, "linked_file_privacy_check")
	default:
		return false
	}
}
func anyRecordHasRoundTripField(records []PaperRecord, field string) bool {
	for _, record := range records {
		if recordHasRoundTripField(record, field) {
			return true
		}
	}
	return false
}

func (m ReferenceManagerInterchangeMatrix) Format(format string) (ReferenceManagerFormatFidelity, bool) {
	for _, row := range m.Formats {
		if row.Format == format {
			return row, true
		}
	}
	return ReferenceManagerFormatFidelity{}, false
}

func (m ReferenceManagerInterchangeMatrix) HasFormat(format string) bool {
	_, ok := m.Format(format)
	return ok
}

func metadataAny(record PaperRecord, key string) bool {
	return metadataValue(record, key) != ""
}
