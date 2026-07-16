package library

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type zoteroRDF struct {
	Items []zoteroRDFItem `xml:",any"`
}

type zoteroRDFItem struct {
	XMLName     xml.Name `xml:""`
	About       string   `xml:"about,attr"`
	Title       string   `xml:"title"`
	DOI         string   `xml:"doi"`
	Identifier  []string `xml:"identifier"`
	Abstract    string   `xml:"abstract"`
	Date        string   `xml:"date"`
	Publisher   string   `xml:"publisher"`
	Venue       string   `xml:"publicationName"`
	CitationKey string   `xml:"citekey"`
	Subject     []string `xml:"subject"`
	Collection  []string `xml:"collection"`
	Note        []string `xml:"note"`
	Annotation  []string `xml:"annotation"`
	Attachment  []string `xml:"attachment"`
}

// ImportZoteroRDF reads a small, interoperable subset of Zotero RDF into PaperRecords.
func ImportZoteroRDF(path string) ([]PaperRecord, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	var rdf zoteroRDF
	if err := xml.Unmarshal(data, &rdf); err != nil {
		return nil, 0, err
	}
	records := make([]PaperRecord, 0, len(rdf.Items))
	skipped := 0
	for _, item := range rdf.Items {
		if strings.EqualFold(item.XMLName.Local, "RDF") || strings.EqualFold(item.XMLName.Local, "Seq") || strings.EqualFold(item.XMLName.Local, "Bag") {
			continue
		}
		metadata := map[string]string{"zotero_rdf_type": item.XMLName.Local, "zotero_rdf_id": item.About}
		if strings.TrimSpace(item.CitationKey) != "" {
			metadata["citation_key"] = strings.TrimSpace(item.CitationKey)
		}
		if tags := compactNonEmpty(item.Subject); len(tags) > 0 {
			metadata["tags"] = strings.Join(tags, "; ")
		}
		if collections := compactNonEmpty(item.Collection); len(collections) > 0 {
			metadata["collections"] = strings.Join(collections, "; ")
			if hierarchy := collectionHierarchy(collections); hierarchy != "" {
				metadata["collection_hierarchy"] = hierarchy
			}
		}
		if notes := compactNonEmpty(item.Note); len(notes) > 0 {
			metadata["note"] = strings.Join(notes, "\n")
		}
		if annotations := compactNonEmpty(item.Annotation); len(annotations) > 0 {
			metadata["annotations"] = strings.Join(annotations, "\n")
		}
		if files := redactedRDFAttachments(item.Attachment); len(files) > 0 {
			metadata["attachment_files"] = strings.Join(files, "; ")
			metadata["linked_file_privacy_check"] = "redacted-local-paths"
		}
		record, err := NewPaperRecord(PaperRecordInput{
			Title:       item.Title,
			Identifiers: Identifiers{DOI: firstNonEmptyRDF(item.DOI, doiFromIdentifiers(item.Identifier))},
			Abstract:    item.Abstract,
			Year:        yearFromRDFDate(item.Date),
			Venue:       item.Venue,
			Publisher:   item.Publisher,
			SourceRefs:  []SourceRef{{Source: "zotero-rdf", Metadata: metadata}},
		})
		if err != nil {
			skipped++
			continue
		}
		records = append(records, record)
	}
	return records, skipped, nil
}

// ExportZoteroRDF writes PaperRecords as a conservative Zotero-compatible RDF/XML subset.
func ExportZoteroRDF(path string, records []PaperRecord) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:prism="http://prismstandard.org/namespaces/1.2/basic/" xmlns:bib="http://purl.org/net/biblio#" xmlns:z="http://www.zotero.org/namespaces/export#" xmlns:better-bibtex="https://retorque.re/zotero-better-bibtex/export#">` + "\n")
	for i, record := range records {
		about := metadataValue(record, "zotero_rdf_id")
		if strings.TrimSpace(about) == "" {
			about = "#paper-" + fmt.Sprint(i+1)
		}
		b.WriteString("  <bib:Article rdf:about=\"")
		writeEscapedAttr(&b, about)
		b.WriteString("\">\n")
		writeElement(&b, "dc:title", record.Title)
		writeElement(&b, "prism:doi", record.Identifiers.DOI)
		writeElement(&b, "dcterms:abstract", record.Abstract)
		writeElement(&b, "prism:publicationName", record.Venue)
		if record.Year > 0 {
			writeElement(&b, "dc:date", fmt.Sprint(record.Year))
		}
		writeElement(&b, "dc:publisher", record.Publisher)
		writeElement(&b, "better-bibtex:citekey", metadataValue(record, "citation_key"))
		for _, tag := range splitCSLKeywords(metadataValue(record, "tags")) {
			writeElement(&b, "dc:subject", tag)
		}
		for _, collection := range splitCSLKeywords(firstNonEmptyRDF(metadataValue(record, "collection_hierarchy"), metadataValue(record, "collections"))) {
			writeElement(&b, "z:collection", collection)
		}
		writeElement(&b, "z:note", metadataValue(record, "note"))
		for _, annotation := range splitRDFLines(metadataValue(record, "annotations")) {
			writeElement(&b, "z:annotation", annotation)
		}
		for _, attachment := range splitCSLKeywords(metadataValue(record, "attachment_files")) {
			writeElement(&b, "z:attachment", filepath.Base(attachment))
		}
		b.WriteString("  </bib:Article>\n")
	}
	b.WriteString("</rdf:RDF>\n")
	return writeExport(path, []byte(b.String()))
}

func doiFromIdentifiers(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.Contains(strings.ToLower(value), "doi") || strings.HasPrefix(strings.ToLower(value), "10.") {
			return value
		}
	}
	return ""
}

func yearFromRDFDate(value string) int {
	return cslIssuedYear(cslIssued{DateParts: [][]int{{atoiPrefix(value)}}})
}

func atoiPrefix(value string) int {
	year := 0
	for _, r := range strings.TrimSpace(value) {
		if r < '0' || r > '9' {
			break
		}
		year = year*10 + int(r-'0')
	}
	return year
}

func redactedRDFAttachments(values []string) []string {
	return redactedAttachmentFiles(rdfAttachments(values))
}

func rdfAttachments(values []string) []cslAttachment {
	out := []cslAttachment{}
	for _, value := range values {
		out = append(out, cslAttachment{Path: value})
	}
	return out
}

func collectionHierarchy(collections []string) string {
	parts := []string{}
	for _, collection := range collections {
		if strings.Contains(collection, "/") || strings.Contains(collection, ">") {
			parts = append(parts, collection)
		}
	}
	return strings.Join(parts, "; ")
}

func splitRDFLines(value string) []string {
	out := []string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func compactNonEmpty(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstNonEmptyRDF(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func metadataValue(record PaperRecord, key string) string {
	for _, ref := range record.SourceRefs {
		if ref.Metadata != nil && strings.TrimSpace(ref.Metadata[key]) != "" {
			return strings.TrimSpace(ref.Metadata[key])
		}
	}
	return ""
}

func writeElement(b *strings.Builder, name, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString("    <")
	b.WriteString(name)
	b.WriteString(">")
	writeEscapedText(b, value)
	b.WriteString("</")
	b.WriteString(name)
	b.WriteString(">\n")
}

func writeEscapedText(b *strings.Builder, value string) {
	_ = xml.EscapeText(builderWriter{b}, []byte(value))
}
func writeEscapedAttr(b *strings.Builder, value string) {
	_ = xml.EscapeText(builderWriter{b}, []byte(value))
}

type builderWriter struct{ b *strings.Builder }

func (w builderWriter) Write(p []byte) (int, error) { return w.b.Write(p) }
