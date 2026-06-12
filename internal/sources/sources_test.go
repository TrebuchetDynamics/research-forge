package sources

import (
	"context"
	"testing"
)

type fakeConnector struct{}

func (fakeConnector) Name() string { return "fake" }

func (fakeConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	return SourceResponse{
		Records: []SourceRecord{{Source: "fake", SourceID: "paper-1", Title: "Artificial photosynthesis catalyst"}},
		RawRef:  "fixtures/fake/search/artificial-photosynthesis.json",
	}, nil
}

func TestRunSearchRecordsConnectorRequestAndResponseProvenance(t *testing.T) {
	query := SourceQuery{Terms: "artificial photosynthesis", Limit: 10}

	run, err := RunSearch(context.Background(), fakeConnector{}, query)
	if err != nil {
		t.Fatalf("RunSearch returned error: %v", err)
	}
	if run.Connector != "fake" {
		t.Fatalf("Connector = %q, want fake", run.Connector)
	}
	if len(run.Response.Records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(run.Response.Records))
	}
	if run.RequestProvenance.Source != "fake" {
		t.Fatalf("request source = %q", run.RequestProvenance.Source)
	}
	if run.RequestProvenance.Query.Terms != "artificial photosynthesis" {
		t.Fatalf("request query = %#v", run.RequestProvenance.Query)
	}
	if run.ResponseProvenance.RecordCount != 1 {
		t.Fatalf("response record count = %d", run.ResponseProvenance.RecordCount)
	}
	if run.ResponseProvenance.RawRef != "fixtures/fake/search/artificial-photosynthesis.json" {
		t.Fatalf("response raw ref = %q", run.ResponseProvenance.RawRef)
	}
}

func TestRunSearchRejectsEmptyQuery(t *testing.T) {
	_, err := RunSearch(context.Background(), fakeConnector{}, SourceQuery{Terms: "   "})
	if err == nil {
		t.Fatalf("RunSearch returned nil error, want empty query validation error")
	}
}
