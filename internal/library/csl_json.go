package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cslItem struct {
	ID             string          `json:"id,omitempty"`
	Type           string          `json:"type,omitempty"`
	Title          string          `json:"title,omitempty"`
	DOI            string          `json:"DOI,omitempty"`
	Abstract       string          `json:"abstract,omitempty"`
	ContainerTitle string          `json:"container-title,omitempty"`
	Publisher      string          `json:"publisher,omitempty"`
	URL            string          `json:"URL,omitempty"`
	CitationKey    string          `json:"citation-key,omitempty"`
	Note           string          `json:"note,omitempty"`
	Keyword        string          `json:"keyword,omitempty"`
	Attachments    []cslAttachment `json:"attachments,omitempty"`
	Issued         cslIssued       `json:"issued,omitempty"`
	Author         []cslName       `json:"author,omitempty"`
}

type cslAttachment struct {
	Title string `json:"title,omitempty"`
	Path  string `json:"path,omitempty"`
}

type cslIssued struct {
	DateParts [][]int `json:"date-parts,omitempty"`
}

type cslName struct {
	Given  string `json:"given,omitempty"`
	Family string `json:"family,omitempty"`
}

// ImportCSLJSON reads Zotero-compatible CSL-JSON items into normalized PaperRecords.
func ImportCSLJSON(path string) ([]PaperRecord, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	var items []cslItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, 0, err
	}
	records := make([]PaperRecord, 0, len(items))
	skipped := 0
	for _, item := range items {
		record, err := NewPaperRecord(PaperRecordInput{
			Title:       item.Title,
			Identifiers: Identifiers{DOI: item.DOI},
			Authors:     cslAuthorsToLibrary(item.Author),
			Abstract:    item.Abstract,
			Year:        cslIssuedYear(item.Issued),
			Venue:       item.ContainerTitle,
			Publisher:   item.Publisher,
			URLs:        normalizeStrings([]string{item.URL}),
			SourceRefs:  []SourceRef{{Source: "csl-json", Metadata: cslMetadata(item)}},
		})
		if err != nil {
			skipped++
			continue
		}
		records = append(records, record)
	}
	return records, skipped, nil
}

// ExportCSLJSON writes PaperRecords as Zotero-compatible CSL-JSON items.
func ExportCSLJSON(path string, records []PaperRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	items := make([]cslItem, 0, len(records))
	for i, record := range records {
		item := cslItem{
			ID:             cslID(record, i),
			Type:           cslType(record),
			Title:          record.Title,
			DOI:            record.Identifiers.DOI,
			Abstract:       record.Abstract,
			ContainerTitle: record.Venue,
			Publisher:      record.Publisher,
			CitationKey:    cslCitationKey(record),
			Note:           cslMetadataValue(record, "note"),
			Keyword:        cslMetadataValue(record, "tags"),
			Author:         libraryAuthorsToCSL(record.Authors),
		}
		if len(record.URLs) > 0 {
			item.URL = record.URLs[0]
		}
		if record.Year > 0 {
			item.Issued = cslIssued{DateParts: [][]int{{record.Year}}}
		}
		items = append(items, item)
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func cslMetadata(item cslItem) map[string]string {
	metadata := map[string]string{
		"csl_id":       item.ID,
		"csl_type":     item.Type,
		"citation_key": item.CitationKey,
	}
	if strings.TrimSpace(item.Note) != "" {
		metadata["note"] = strings.TrimSpace(item.Note)
	}
	if strings.TrimSpace(item.Keyword) != "" {
		metadata["tags"] = strings.Join(splitCSLKeywords(item.Keyword), "; ")
	}
	if files := redactedAttachmentFiles(item.Attachments); len(files) > 0 {
		metadata["attachment_files"] = strings.Join(files, "; ")
	}
	return metadata
}

func splitCSLKeywords(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' })
	out := []string{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func redactedAttachmentFiles(attachments []cslAttachment) []string {
	out := []string{}
	for _, attachment := range attachments {
		path := strings.TrimSpace(attachment.Path)
		if path == "" {
			continue
		}
		file := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
		if file != "." && file != string(filepath.Separator) && file != "" {
			out = append(out, file)
		}
	}
	return out
}

func cslAuthorsToLibrary(authors []cslName) []Author {
	out := make([]Author, 0, len(authors))
	for _, author := range authors {
		out = append(out, Author{Given: author.Given, Family: author.Family})
	}
	return out
}

func libraryAuthorsToCSL(authors []Author) []cslName {
	out := make([]cslName, 0, len(authors))
	for _, author := range authors {
		out = append(out, cslName{Given: author.Given, Family: author.Family})
	}
	return out
}

func cslIssuedYear(issued cslIssued) int {
	if len(issued.DateParts) == 0 || len(issued.DateParts[0]) == 0 {
		return 0
	}
	return issued.DateParts[0][0]
}

func cslID(record PaperRecord, index int) string {
	for _, ref := range record.SourceRefs {
		if ref.Source == "csl-json" && ref.Metadata != nil && strings.TrimSpace(ref.Metadata["csl_id"]) != "" {
			return strings.TrimSpace(ref.Metadata["csl_id"])
		}
	}
	if record.Identifiers.DOI != "" {
		return record.Identifiers.DOI
	}
	return "paper-" + strconv.Itoa(index+1)
}

func cslMetadataValue(record PaperRecord, key string) string {
	for _, ref := range record.SourceRefs {
		if (ref.Source == "csl-json" || ref.Source == "zotero" || ref.Source == "better-bibtex") && ref.Metadata != nil && strings.TrimSpace(ref.Metadata[key]) != "" {
			return strings.TrimSpace(ref.Metadata[key])
		}
	}
	return ""
}

func cslCitationKey(record PaperRecord) string {
	for _, ref := range record.SourceRefs {
		if (ref.Source == "csl-json" || ref.Source == "better-bibtex") && ref.Metadata != nil && strings.TrimSpace(ref.Metadata["citation_key"]) != "" {
			return strings.TrimSpace(ref.Metadata["citation_key"])
		}
	}
	return ""
}

func cslType(record PaperRecord) string {
	for _, ref := range record.SourceRefs {
		if ref.Source == "csl-json" && ref.Metadata != nil && strings.TrimSpace(ref.Metadata["csl_type"]) != "" {
			return strings.TrimSpace(ref.Metadata["csl_type"])
		}
	}
	return "article-journal"
}
