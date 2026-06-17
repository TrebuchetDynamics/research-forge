package evidence

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type ExtractionGridInput struct {
	Items                    []EvidenceItem
	ParsedDocuments          []parsing.ParsedDocument
	AnalysisIncludedPaperIDs []string
	PDFBaseURL               string
}

type ExtractionGrid struct {
	SchemaVersion string              `json:"schemaVersion"`
	Rows          []ExtractionGridRow `json:"rows"`
}

type ExtractionGridRow struct {
	PaperID                    string             `json:"paperId"`
	SchemaName                 string             `json:"schemaName"`
	FieldName                  string             `json:"fieldName"`
	FieldValue                 string             `json:"fieldValue"`
	SupportKind                SupportKind        `json:"supportKind"`
	SupportRef                 string             `json:"supportRef"`
	ParserName                 string             `json:"parserName,omitempty"`
	ParserVersion              string             `json:"parserVersion,omitempty"`
	ParserOffset               parsing.TextOffset `json:"parserOffset"`
	PDFViewURL                 string             `json:"pdfViewUrl,omitempty"`
	ReviewerStatus             Status             `json:"reviewerStatus"`
	CorrectionHistory          []CorrectionEvent  `json:"correctionHistory,omitempty"`
	DownstreamAnalysisIncluded bool               `json:"downstreamAnalysisIncluded"`
}

type supportLookup struct {
	ParserName    string
	ParserVersion string
	Offset        parsing.TextOffset
}

func BuildExtractionGrid(input ExtractionGridInput) ExtractionGrid {
	included := map[string]bool{}
	for _, id := range input.AnalysisIncludedPaperIDs {
		if strings.TrimSpace(id) != "" {
			included[id] = true
		}
	}
	supports := buildSupportLookup(input.ParsedDocuments)
	grid := ExtractionGrid{SchemaVersion: "1"}
	for _, item := range input.Items {
		fieldNames := make([]string, 0, len(item.Values))
		for field := range item.Values {
			fieldNames = append(fieldNames, field)
		}
		sort.Strings(fieldNames)
		for _, field := range fieldNames {
			lookup := supports[item.PaperID+"\x00"+item.Support.Ref]
			row := ExtractionGridRow{PaperID: item.PaperID, SchemaName: item.SchemaName, FieldName: field, FieldValue: item.Values[field], SupportKind: item.Support.Kind, SupportRef: item.Support.Ref, ParserName: lookup.ParserName, ParserVersion: lookup.ParserVersion, ParserOffset: lookup.Offset, ReviewerStatus: item.Status, CorrectionHistory: append([]CorrectionEvent{}, item.History...), DownstreamAnalysisIncluded: included[item.PaperID]}
			if row.ParserOffset.End == 0 && row.ParserOffset.Start == 0 && lookup.ParserName == "" {
				row.ParserOffset = parsing.TextOffset{Start: -1, End: -1}
			}
			if base := strings.TrimRight(input.PDFBaseURL, "/"); base != "" {
				row.PDFViewURL = base + "/" + item.PaperID + "/pdf"
				if strings.TrimSpace(item.Support.Ref) != "" {
					row.PDFViewURL += "#" + item.Support.Ref
				}
			}
			grid.Rows = append(grid.Rows, row)
		}
	}
	return grid
}

func buildSupportLookup(docs []parsing.ParsedDocument) map[string]supportLookup {
	out := map[string]supportLookup{}
	for _, doc := range docs {
		for _, section := range doc.Sections {
			if section.ID != "" {
				out[doc.PaperID+"\x00"+section.ID] = supportLookup{ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, Offset: section.Offset}
			}
			for _, passage := range section.Passages {
				if passage.ID != "" {
					out[doc.PaperID+"\x00"+passage.ID] = supportLookup{ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, Offset: passage.Offset}
				}
			}
		}
		for _, annotation := range doc.LayeredAnnotations {
			if annotation.ID != "" {
				out[doc.PaperID+"\x00"+annotation.ID] = supportLookup{ParserName: firstNonEmptyEvidence(annotation.ParserName, doc.ParserName), ParserVersion: doc.ParserVersion, Offset: annotation.Offset}
			}
		}
	}
	return out
}

func firstNonEmptyEvidence(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
