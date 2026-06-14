package parsing

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// ReferenceMatch records how one parsed bibliography reference matched a source connector result.
type ReferenceMatch struct {
	Index        int               `json:"index"`
	Title        string            `json:"title"`
	DOI          string            `json:"doi,omitempty"`
	Matched      bool              `json:"matched"`
	Source       string            `json:"source,omitempty"`
	SourceID     string            `json:"sourceId,omitempty"`
	MatchedDOI   string            `json:"matchedDoi,omitempty"`
	MatchedTitle string            `json:"matchedTitle,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ReferenceNormalizationReport summarizes connector normalization for parsed references.
type ReferenceNormalizationReport struct {
	PaperID    string           `json:"paperId"`
	Connector  string           `json:"connector"`
	References int              `json:"references"`
	Matched    int              `json:"matched"`
	Matches    []ReferenceMatch `json:"matches"`
}

// NormalizeParsedReferences queries a source connector for each parsed reference and records the top match.
func NormalizeParsedReferences(ctx context.Context, connector sources.SourceConnector, doc ParsedDocument) (ReferenceNormalizationReport, error) {
	if connector == nil {
		return ReferenceNormalizationReport{}, fmt.Errorf("source connector is required")
	}
	report := ReferenceNormalizationReport{PaperID: doc.PaperID, Connector: connector.Name(), References: len(doc.References)}
	for i, ref := range doc.References {
		query := referenceQuery(ref)
		match := ReferenceMatch{Index: i, Title: ref.Title, DOI: strings.TrimSpace(ref.DOI)}
		if query == "" {
			report.Matches = append(report.Matches, match)
			continue
		}
		response, err := connector.Search(ctx, sources.SourceQuery{Terms: query, Limit: 1})
		if err != nil {
			return ReferenceNormalizationReport{}, err
		}
		if len(response.Records) > 0 {
			record := response.Records[0]
			match.Matched = true
			match.Source = record.Source
			match.SourceID = record.SourceID
			match.MatchedDOI = record.Identifiers.DOI
			match.MatchedTitle = record.Title
			match.Metadata = record.Metadata
			report.Matched++
		}
		report.Matches = append(report.Matches, match)
	}
	return report, nil
}

func referenceQuery(ref Reference) string {
	if doi := strings.TrimSpace(ref.DOI); doi != "" {
		return doi
	}
	return strings.TrimSpace(ref.Title)
}
