package library

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ImportJSON reads PaperRecords from a deterministic JSON fixture/export file.
func ImportJSON(path string) ([]PaperRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []PaperRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	out := make([]PaperRecord, 0, len(records))
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
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

// ImportBibTeX reads a minimal deterministic BibTeX fixture/export file.
func ImportBibTeX(path string) ([]PaperRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	entries := strings.Split(string(data), "@")
	records := []PaperRecord{}
	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		fields := parseBibTeXFields(entry)
		year, _ := strconv.Atoi(fields["year"])
		record, err := NewPaperRecord(PaperRecordInput{
			Title:       fields["title"],
			Identifiers: Identifiers{DOI: fields["doi"]},
			Year:        year,
			Venue:       fields["journal"],
			Publisher:   fields["publisher"],
		})
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// ExportBibTeX writes a minimal deterministic BibTeX file.
func ExportBibTeX(path string, records []PaperRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var builder strings.Builder
	for i, record := range records {
		fmt.Fprintf(&builder, "@article{paper%d,\n", i+1)
		fmt.Fprintf(&builder, "  title = {%s},\n", record.Title)
		fmt.Fprintf(&builder, "  doi = {%s},\n", record.Identifiers.DOI)
		fmt.Fprintf(&builder, "  year = {%d}", record.Year)
		if record.Venue != "" {
			fmt.Fprintf(&builder, ",\n  journal = {%s}\n", record.Venue)
		} else {
			builder.WriteString("\n")
		}
		builder.WriteString("}\n")
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
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

// ImportRIS reads a minimal deterministic RIS fixture/export file.
func ImportRIS(path string) ([]PaperRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	records := []PaperRecord{}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "ER  -") {
			year, _ := strconv.Atoi(fields["PY"])
			record, err := NewPaperRecord(PaperRecordInput{Title: fields["TI"], Identifiers: Identifiers{DOI: fields["DO"]}, Year: year, Venue: fields["JO"]})
			if err != nil {
				return nil, err
			}
			records = append(records, record)
			fields = map[string]string{}
			continue
		}
		if len(line) >= 6 && line[2:6] == "  - " {
			fields[line[:2]] = strings.TrimSpace(line[6:])
		}
	}
	return records, nil
}

// ExportRIS writes a minimal deterministic RIS file.
func ExportRIS(path string, records []PaperRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
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
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

// ImportCSV reads PaperRecords from a deterministic CSV fixture/export file.
func ImportCSV(path string) ([]PaperRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("csv header is required")
	}
	header := map[string]int{}
	for i, name := range rows[0] {
		header[name] = i
	}
	records := []PaperRecord{}
	for _, row := range rows[1:] {
		year, _ := strconv.Atoi(csvValue(row, header, "year"))
		record, err := NewPaperRecord(PaperRecordInput{
			Title: csvValue(row, header, "title"),
			Identifiers: Identifiers{
				DOI:               csvValue(row, header, "doi"),
				ArXivID:           csvValue(row, header, "arxiv_id"),
				PMID:              csvValue(row, header, "pmid"),
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
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// ExportCSV writes PaperRecords as stable CSV.
func ExportCSV(path string, records []PaperRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"title", "doi", "arxiv_id", "pmid", "openalex_id", "crossref_id", "semantic_scholar_id", "year", "abstract", "venue", "publisher", "license", "open_access"}); err != nil {
		return err
	}
	for _, record := range records {
		if err := writer.Write([]string{record.Title, record.Identifiers.DOI, record.Identifiers.ArXivID, record.Identifiers.PMID, record.Identifiers.OpenAlexID, record.Identifiers.CrossrefID, record.Identifiers.SemanticScholarID, strconv.Itoa(record.Year), record.Abstract, record.Venue, record.Publisher, record.License, strconv.FormatBool(record.OpenAccess)}); err != nil {
			return err
		}
	}
	return writer.Error()
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
