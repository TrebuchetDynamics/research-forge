package parsing

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

type fakeReferenceConnector struct {
	queries []sources.SourceQuery
}

func (f *fakeReferenceConnector) Name() string { return "fake-source" }

func (f *fakeReferenceConnector) Search(_ context.Context, query sources.SourceQuery) (sources.SourceResponse, error) {
	f.queries = append(f.queries, query)
	return sources.SourceResponse{Records: []sources.SourceRecord{{
		Source:      "fake-source",
		SourceID:    "work-1",
		Title:       "Normalized reference",
		Identifiers: sources.Identifiers{DOI: "10.1000/normalized"},
		Metadata:    map[string]string{"matched_by": "fixture"},
	}}}, nil
}

func TestNormalizeParsedReferencesQueriesConnectorAndReportsMatches(t *testing.T) {
	connector := &fakeReferenceConnector{}
	doc := ParsedDocument{PaperID: "paper-1", References: []Reference{{Title: "Reference A", DOI: "10.1000/ref-a"}, {Title: "Reference B"}}}

	report, err := NormalizeParsedReferences(context.Background(), connector, doc)
	if err != nil {
		t.Fatalf("NormalizeParsedReferences returned error: %v", err)
	}
	if report.PaperID != "paper-1" || report.Connector != "fake-source" || report.References != 2 || report.Matched != 2 {
		t.Fatalf("report = %#v", report)
	}
	if connector.queries[0].Terms != "10.1000/ref-a" || connector.queries[1].Terms != "Reference B" {
		t.Fatalf("queries = %#v", connector.queries)
	}
	if !report.Matches[0].Matched || report.Matches[0].MatchedDOI != "10.1000/normalized" || report.Matches[0].Metadata["matched_by"] != "fixture" {
		t.Fatalf("match = %#v", report.Matches[0])
	}
}

func TestNormalizeParsedReferencesSkipsBlankReferences(t *testing.T) {
	connector := &fakeReferenceConnector{}
	report, err := NormalizeParsedReferences(context.Background(), connector, ParsedDocument{References: []Reference{{}}})
	if err != nil {
		t.Fatalf("NormalizeParsedReferences returned error: %v", err)
	}
	if report.References != 1 || report.Matched != 0 || len(connector.queries) != 0 {
		t.Fatalf("report = %#v queries=%#v", report, connector.queries)
	}
}
