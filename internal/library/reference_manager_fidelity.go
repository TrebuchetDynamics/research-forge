package library

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

func metadataAny(record PaperRecord, key string) bool {
	return metadataValue(record, key) != ""
}
