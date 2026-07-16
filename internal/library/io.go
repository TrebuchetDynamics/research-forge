package library

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

// ImportJSON reads PaperRecords from a deterministic JSON fixture/export file.
// It skips records that cannot be normalized into a storable PaperRecord (for
// example, records missing an identifier) and returns how many were skipped, so
// a single unstorable record cannot abort the whole import.
func ImportJSON(path string) ([]PaperRecord, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	var records []PaperRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, 0, err
	}
	out := make([]PaperRecord, 0, len(records))
	skipped := 0
	for _, record := range records {
		normalized, err := NewPaperRecord(PaperRecordInput{
			Title:         record.Title,
			Identifiers:   record.Identifiers,
			Authors:       record.Authors,
			Abstract:      record.Abstract,
			Year:          record.Year,
			Venue:         record.Venue,
			Publisher:     record.Publisher,
			URLs:          record.URLs,
			License:       record.License,
			OpenAccess:    record.OpenAccess,
			SourcePayload: record.SourcePayload,
			SourceRefs:    record.SourceRefs,
		})
		if err != nil {
			skipped++
			continue
		}
		out = append(out, normalized)
	}
	return out, skipped, nil
}

// ImportBibTeX reads a minimal deterministic BibTeX fixture/export file. It
// skips entries that cannot be normalized into a storable PaperRecord and
// returns how many were skipped.
func ImportBibTeX(path string) ([]PaperRecord, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	entries := strings.Split(string(data), "@")
	records := []PaperRecord{}
	skipped := 0
	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		fields := parseBibTeXFields(entry)
		year, _ := strconv.Atoi(fields["year"])
		metadata := jabRefMetadata(entry, fields)
		record, err := NewPaperRecord(PaperRecordInput{
			Title:       fields["title"],
			Identifiers: Identifiers{DOI: fields["doi"]},
			Year:        year,
			Venue:       fields["journal"],
			Publisher:   fields["publisher"],
			SourceRefs:  []SourceRef{{Source: "jabref-bibtex", Metadata: metadata}},
		})
		if err != nil {
			skipped++
			continue
		}
		records = append(records, record)
	}
	return records, skipped, nil
}

// ExportBibTeX writes a minimal deterministic BibTeX file.
func ExportBibTeX(path string, records []PaperRecord) error {
	var builder strings.Builder
	for i, record := range records {
		key := metadataValue(record, "citation_key")
		if key == "" {
			key = fmt.Sprintf("paper%d", i+1)
		}
		fmt.Fprintf(&builder, "@article{%s,\n", key)
		fmt.Fprintf(&builder, "  title = {%s},\n", record.Title)
		fmt.Fprintf(&builder, "  doi = {%s},\n", record.Identifiers.DOI)
		fmt.Fprintf(&builder, "  year = {%d}", record.Year)
		if record.Venue != "" {
			fmt.Fprintf(&builder, ",\n  journal = {%s}", record.Venue)
		}
		writeBibTeXMetadataField(&builder, record, "keywords", "tags")
		writeBibTeXMetadataField(&builder, record, "groups", "groups")
		writeBibTeXMetadataField(&builder, record, "note", "note")
		writeBibTeXMetadataField(&builder, record, "annote", "annotations")
		if file := metadataValue(record, "attachment_files"); file != "" {
			fmt.Fprintf(&builder, ",\n  file = {:%s:PDF}", file)
		}
		builder.WriteString("\n}\n")
	}
	return writeExport(path, []byte(builder.String()))
}

func writeBibTeXMetadataField(builder *strings.Builder, record PaperRecord, fieldName, metadataKey string) {
	if value := metadataValue(record, metadataKey); value != "" {
		fmt.Fprintf(builder, ",\n  %s = {%s}", fieldName, value)
	}
}

func parseBibTeXFields(entry string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(entry, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, ","))
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])
		value = strings.TrimPrefix(value, "{")
		value = strings.TrimSuffix(value, "}")
		fields[key] = strings.TrimSpace(value)
	}
	return fields
}

func jabRefMetadata(entry string, fields map[string]string) map[string]string {
	metadata := map[string]string{"citation_key": bibTeXCitationKey(entry)}
	copyBibTeXMetadata(metadata, "tags", fields["keywords"])
	copyBibTeXMetadata(metadata, "groups", fields["groups"])
	copyBibTeXMetadata(metadata, "note", fields["note"])
	copyBibTeXMetadata(metadata, "annotations", fields["annote"])
	if files := redactedBibTeXFiles(fields["file"]); len(files) > 0 {
		metadata["attachment_files"] = strings.Join(files, "; ")
		metadata["linked_file_privacy_check"] = "redacted-local-paths"
	}
	if diff := bibTeXCleanupDiffs(fields); diff != "" {
		metadata["cleanup_diff"] = diff
	}
	return metadata
}

func copyBibTeXMetadata(metadata map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		metadata[key] = value
	}
}

func bibTeXCitationKey(entry string) string {
	line := strings.TrimSpace(strings.SplitN(entry, "\n", 2)[0])
	if idx := strings.Index(line, "{"); idx >= 0 {
		line = line[idx+1:]
	}
	if idx := strings.Index(line, ","); idx >= 0 {
		return strings.TrimSpace(line[:idx])
	}
	return ""
}

func redactedBibTeXFiles(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := strings.Split(part, ":")
		candidate := part
		if len(pieces) >= 2 {
			candidate = pieces[len(pieces)-2]
		}
		candidate = strings.ReplaceAll(candidate, "\\", "/")
		file := filepath.Base(candidate)
		if file != "" && file != "." && file != string(filepath.Separator) {
			out = append(out, file)
		}
	}
	return out
}

func bibTeXCleanupDiffs(fields map[string]string) string {
	diffs := []string{}
	if raw := strings.TrimSpace(fields["doi"]); raw != "" {
		normalized := normalizeDOI(raw)
		if raw != normalized {
			diffs = append(diffs, "doi: "+raw+" -> "+normalized)
		}
	}
	return strings.Join(diffs, "; ")
}

// ImportRIS reads a minimal deterministic RIS fixture/export file. It skips
// entries that cannot be normalized into a storable PaperRecord and returns how
// many were skipped.
func ImportRIS(path string) ([]PaperRecord, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	records := []PaperRecord{}
	skipped := 0
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "ER  -") {
			year, _ := strconv.Atoi(fields["PY"])
			record, err := NewPaperRecord(PaperRecordInput{Title: fields["TI"], Identifiers: Identifiers{DOI: fields["DO"]}, Year: year, Venue: fields["JO"]})
			if err != nil {
				skipped++
				fields = map[string]string{}
				continue
			}
			records = append(records, record)
			fields = map[string]string{}
			continue
		}
		if len(line) >= 6 && line[2:6] == "  - " {
			fields[line[:2]] = strings.TrimSpace(line[6:])
		}
	}
	return records, skipped, nil
}

// ExportRIS writes a minimal deterministic RIS file.
func ExportRIS(path string, records []PaperRecord) error {
	var builder strings.Builder
	for _, record := range records {
		builder.WriteString("TY  - JOUR\n")
		fmt.Fprintf(&builder, "TI  - %s\n", record.Title)
		fmt.Fprintf(&builder, "DO  - %s\n", record.Identifiers.DOI)
		fmt.Fprintf(&builder, "PY  - %d\n", record.Year)
		if record.Venue != "" {
			fmt.Fprintf(&builder, "JO  - %s\n", record.Venue)
		}
		builder.WriteString("ER  - \n")
	}
	return writeExport(path, []byte(builder.String()))
}

// ImportCSV reads PaperRecords from a deterministic CSV fixture/export file. It
// skips rows that cannot be normalized into a storable PaperRecord and returns
// how many were skipped.
func ImportCSV(path string) ([]PaperRecord, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return nil, 0, fmt.Errorf("csv header is required")
	}
	header := map[string]int{}
	for i, name := range rows[0] {
		header[name] = i
	}
	records := []PaperRecord{}
	skipped := 0
	for _, row := range rows[1:] {
		year, _ := strconv.Atoi(csvValue(row, header, "year"))
		record, err := NewPaperRecord(PaperRecordInput{
			Title: csvValue(row, header, "title"),
			Identifiers: Identifiers{
				DOI:               csvValue(row, header, "doi"),
				ArXivID:           csvValue(row, header, "arxiv_id"),
				PMID:              csvValue(row, header, "pmid"),
				PMCID:             csvValue(row, header, "pmcid"),
				OpenAlexID:        csvValue(row, header, "openalex_id"),
				CrossrefID:        csvValue(row, header, "crossref_id"),
				SemanticScholarID: csvValue(row, header, "semantic_scholar_id"),
			},
			Year:      year,
			Abstract:  csvValue(row, header, "abstract"),
			Venue:     csvValue(row, header, "venue"),
			Publisher: csvValue(row, header, "publisher"),
			License:   csvValue(row, header, "license"),
		})
		if err != nil {
			skipped++
			continue
		}
		records = append(records, record)
	}
	return records, skipped, nil
}

// ExportCSV writes PaperRecords as stable CSV.
func ExportCSV(path string, records []PaperRecord) error {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"title", "doi", "arxiv_id", "pmid", "pmcid", "openalex_id", "crossref_id", "semantic_scholar_id", "year", "abstract", "venue", "publisher", "license", "open_access"}); err != nil {
		return err
	}
	for _, record := range records {
		if err := writer.Write([]string{record.Title, record.Identifiers.DOI, record.Identifiers.ArXivID, record.Identifiers.PMID, record.Identifiers.PMCID, record.Identifiers.OpenAlexID, record.Identifiers.CrossrefID, record.Identifiers.SemanticScholarID, strconv.Itoa(record.Year), record.Abstract, record.Venue, record.Publisher, record.License, strconv.FormatBool(record.OpenAccess)}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return writeExport(path, buffer.Bytes())
}

func csvValue(row []string, header map[string]int, name string) string {
	idx, ok := header[name]
	if !ok || idx >= len(row) {
		return ""
	}
	return row[idx]
}

// ExportJSON writes PaperRecords as stable, pretty JSON.
func ExportJSON(path string, records []PaperRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeExport(path, data)
}

func writeExport(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return filetxn.Replace(path, data, 0o644)
}
