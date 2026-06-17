package sources

import "testing"

func TestSemanticScholarGraphRunFileTracksQuotaResumeAndFieldRestrictions(t *testing.T) {
	run := NewSemanticScholarGraphRun(SemanticScholarGraphRunOptions{SeedID: "S2", Direction: SemanticScholarDirectionBoth, Limit: 10, Depth: 2, MaxRecords: 50, RequestedFields: []string{"paperId", "title", "abstract", "embedding"}, QuotaRemaining: 99})
	if run.SchemaVersion != "1" || run.NextFrontier[0] != "S2" || run.QuotaRemaining != 99 || run.BudgetEstimate.EstimatedAPICalls == 0 {
		t.Fatalf("run = %#v", run)
	}
	if !run.HasFieldRestriction("embedding") {
		t.Fatalf("field restrictions missing: %#v", run.FieldRestrictions)
	}
	run = run.RecordExpansion(CitationGraphExpansion{Edges: []CitationEdge{{SourceID: "S2", TargetID: "R1"}}, Records: map[string]SourceRecord{"R1": {SourceID: "R1"}}}, []string{"R1"}, 98)
	if run.EdgeCount != 1 || run.RecordCount != 1 || run.NextFrontier[0] != "R1" || run.QuotaRemaining != 98 || run.Completed {
		t.Fatalf("updated run = %#v", run)
	}
}
