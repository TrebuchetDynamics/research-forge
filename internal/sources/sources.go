package sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// SourceConnector searches one scholarly metadata source.
type SourceConnector interface {
	Name() string
	Search(context.Context, SourceQuery) (SourceResponse, error)
}

// SourceQuery describes a scholarly metadata search request.
type SourceQuery struct {
	Terms      string
	Limit      int
	PageCursor string
	Filters    map[string]string
}

// Identifiers are normalized scholarly IDs on a source result.
type Identifiers struct {
	DOI               string
	OpenAlexID        string
	ArXivID           string
	PMID              string
	PMCID             string
	CrossrefID        string
	SemanticScholarID string
	ADSBibcode        string
}

// SourceRecord is a source-specific scholarly metadata result.
type SourceRecord struct {
	Source      string
	SourceID    string
	Title       string
	Identifiers Identifiers
	Year        int
	Abstract    string
	Venue       string
	Publisher   string
	URLs        []string
	License     string
	OpenAccess  bool
	Metadata    map[string]string
}

// SourceResponse is the normalized response from a SourceConnector.
type SourceResponse struct {
	Records        []SourceRecord
	RawRef         string
	NextPageCursor string
}

// PaperRecords converts source records into library PaperRecords.
func PaperRecords(response SourceResponse) ([]library.PaperRecord, error) {
	papers := make([]library.PaperRecord, 0, len(response.Records))
	for _, record := range response.Records {
		paper, err := library.NewPaperRecord(library.PaperRecordInput{
			Title:    record.Title,
			Abstract: record.Abstract,
			Identifiers: library.Identifiers{
				DOI:               record.Identifiers.DOI,
				OpenAlexID:        record.Identifiers.OpenAlexID,
				ArXivID:           record.Identifiers.ArXivID,
				PMID:              record.Identifiers.PMID,
				PMCID:             record.Identifiers.PMCID,
				CrossrefID:        record.Identifiers.CrossrefID,
				SemanticScholarID: record.Identifiers.SemanticScholarID,
				ADSBibcode:        record.Identifiers.ADSBibcode,
			},
			Year:       record.Year,
			Venue:      record.Venue,
			Publisher:  record.Publisher,
			URLs:       record.URLs,
			License:    record.License,
			OpenAccess: record.OpenAccess,
			SourceRefs: []library.SourceRef{{
				Source:        record.Source,
				RawPayloadRef: response.RawRef,
				Metadata:      record.Metadata,
			}},
		})
		if err != nil {
			continue
		}
		papers = append(papers, paper)
	}
	return papers, nil
}

// RequestProvenance records what was sent to a SourceConnector.
type RequestProvenance struct {
	Source string
	Query  SourceQuery
}

// ResponseProvenance records what came back from a SourceConnector.
type ResponseProvenance struct {
	Source      string
	RecordCount int
	RawRef      string
}

// SearchRun ties connector output to request and response Provenance metadata.
type SearchRun struct {
	Connector          string
	Query              SourceQuery
	Response           SourceResponse
	RequestProvenance  RequestProvenance
	ResponseProvenance ResponseProvenance
}

// RunSearch validates a query, invokes a connector, and records request/response Provenance metadata.
func RunSearch(ctx context.Context, connector SourceConnector, query SourceQuery) (SearchRun, error) {
	query.Terms = strings.TrimSpace(query.Terms)
	if query.Terms == "" {
		return SearchRun{}, fmt.Errorf("source query terms are required")
	}
	name := connector.Name()
	response, err := connector.Search(ctx, query)
	if err != nil {
		return SearchRun{}, err
	}
	return SearchRun{
		Connector: name,
		Query:     query,
		Response:  response,
		RequestProvenance: RequestProvenance{
			Source: name,
			Query:  query,
		},
		ResponseProvenance: ResponseProvenance{
			Source:      name,
			RecordCount: len(response.Records),
			RawRef:      response.RawRef,
		},
	}, nil
}
