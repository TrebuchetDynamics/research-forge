package parsing

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// ReferenceMatch records how one parsed bibliography reference matched a source connector result.
type ReferenceMatch struct {
	Index            int                 `json:"index"`
	Title            string              `json:"title"`
	DOI              string              `json:"doi,omitempty"`
	Raw              string              `json:"raw,omitempty"`
	ParserConfidence float64             `json:"parserConfidence,omitempty"`
	Matched          bool                `json:"matched"`
	Source           string              `json:"source,omitempty"`
	SourceID         string              `json:"sourceId,omitempty"`
	MatchedDOI       string              `json:"matchedDoi,omitempty"`
	MatchedTitle     string              `json:"matchedTitle,omitempty"`
	MatchConfidence  float64             `json:"matchConfidence,omitempty"`
	CandidateCount   int                 `json:"candidateCount"`
	Ambiguous        bool                `json:"ambiguous"`
	AmbiguityReason  string              `json:"ambiguityReason,omitempty"`
	Request          sources.SourceQuery `json:"request"`
	ResponseRawRef   string              `json:"responseRawRef,omitempty"`
	Metadata         map[string]string   `json:"metadata,omitempty"`
}

// ReferenceNormalizationReport summarizes connector normalization for parsed references.
type ReferenceNormalizationReport struct {
	PaperID        string                `json:"paperId"`
	Connector      string                `json:"connector"`
	References     int                   `json:"references"`
	Matched        int                   `json:"matched"`
	Ambiguous      int                   `json:"ambiguous"`
	Matches        []ReferenceMatch      `json:"matches"`
	AmbiguityQueue []ReferenceReviewItem `json:"ambiguityQueue,omitempty"`
}

// NormalizeParsedReferences queries a source connector for each parsed reference and records the top match.
func NormalizeParsedReferences(ctx context.Context, connector sources.SourceConnector, doc ParsedDocument) (ReferenceNormalizationReport, error) {
	if connector == nil {
		return ReferenceNormalizationReport{}, fmt.Errorf("source connector is required")
	}
	report := ReferenceNormalizationReport{PaperID: doc.PaperID, Connector: connector.Name(), References: len(doc.References)}
	for i, ref := range doc.References {
		query := referenceQuery(ref)
		request := sources.SourceQuery{Terms: query, Limit: 3}
		match := ReferenceMatch{Index: i, Title: ref.Title, DOI: strings.TrimSpace(ref.DOI), Raw: ref.Raw, ParserConfidence: ref.Confidence, Request: request}
		if query == "" {
			match.Ambiguous = true
			match.AmbiguityReason = "blank_reference"
			report.Ambiguous++
			report.AmbiguityQueue = append(report.AmbiguityQueue, ReferenceReviewItem{Index: i, Title: ref.Title, DOI: ref.DOI, Raw: ref.Raw, Confidence: ref.Confidence, Reason: match.AmbiguityReason})
			report.Matches = append(report.Matches, match)
			continue
		}
		response, err := connector.Search(ctx, request)
		if err != nil {
			return ReferenceNormalizationReport{}, err
		}
		match.ResponseRawRef = response.RawRef
		match.CandidateCount = len(response.Records)
		if len(response.Records) > 0 {
			record := response.Records[0]
			match.Matched = true
			match.Source = record.Source
			match.SourceID = record.SourceID
			match.MatchedDOI = record.Identifiers.DOI
			match.MatchedTitle = record.Title
			match.Metadata = record.Metadata
			match.MatchConfidence = referenceMatchConfidence(ref, record, len(response.Records))
			report.Matched++
		}
		if reason := referenceAmbiguityReason(ref, match); reason != "" {
			match.Ambiguous = true
			match.AmbiguityReason = reason
			report.Ambiguous++
			report.AmbiguityQueue = append(report.AmbiguityQueue, ReferenceReviewItem{Index: i, Title: ref.Title, DOI: ref.DOI, Raw: ref.Raw, Confidence: ref.Confidence, Reason: reason})
		}
		report.Matches = append(report.Matches, match)
	}
	return report, nil
}

func referenceMatchConfidence(ref Reference, record sources.SourceRecord, candidateCount int) float64 {
	score := 0.5
	if strings.TrimSpace(ref.DOI) != "" && strings.EqualFold(strings.TrimSpace(ref.DOI), strings.TrimSpace(record.Identifiers.DOI)) {
		score = 0.95
	} else if strings.TrimSpace(ref.Title) != "" && strings.EqualFold(strings.TrimSpace(ref.Title), strings.TrimSpace(record.Title)) {
		score = 0.85
	} else if strings.TrimSpace(record.Title) != "" {
		score = 0.65
	}
	if candidateCount > 1 {
		score -= 0.15
	}
	if ref.Confidence > 0 && ref.Confidence < 0.75 {
		score -= 0.1
	}
	if score < 0.1 {
		return 0.1
	}
	return score
}

func referenceAmbiguityReason(ref Reference, match ReferenceMatch) string {
	switch {
	case !match.Matched:
		return "no_source_match"
	case match.CandidateCount > 1:
		return "multiple_candidates"
	case ref.Confidence > 0 && ref.Confidence < 0.75:
		return "low_parser_confidence"
	case match.MatchConfidence > 0 && match.MatchConfidence < 0.75:
		return "low_match_confidence"
	case strings.TrimSpace(ref.Raw) != "" && strings.TrimSpace(ref.Title) == "" && strings.TrimSpace(ref.DOI) == "":
		return "raw_only"
	default:
		return ""
	}
}

func referenceQuery(ref Reference) string {
	if doi := strings.TrimSpace(ref.DOI); doi != "" {
		return doi
	}
	if title := strings.TrimSpace(ref.Title); title != "" {
		return title
	}
	return strings.TrimSpace(ref.Raw)
}
