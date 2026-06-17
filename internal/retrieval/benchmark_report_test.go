package retrieval

import "testing"

func TestRunRetrievalBenchmarkComparesBackendsOnDeterministicFixtures(t *testing.T) {
	fixture := DefaultRetrievalBenchmarkFixture()
	report, err := RunRetrievalBenchmark(fixture, 2)
	if err != nil {
		t.Fatalf("RunRetrievalBenchmark returned error: %v", err)
	}
	if report.SchemaVersion != "1" || report.QuerySetChecksum == "" || report.QueryCount == 0 {
		t.Fatalf("report metadata = %#v", report)
	}
	want := map[string]bool{"sqlite-fts": false, "opensearch": false, "qdrant": false, "hybrid": false}
	for _, result := range report.Backends {
		want[result.Backend] = true
		if result.RecallAtK < 0 || result.RecallAtK > 1 || result.MRR < 0 || result.MRR > 1 {
			t.Fatalf("bad result = %#v", result)
		}
	}
	for backend, seen := range want {
		if !seen {
			t.Fatalf("missing backend %s in %#v", backend, report.Backends)
		}
	}
	if report.SelectedBackend == "" || len(report.QueryResults) == 0 {
		t.Fatalf("selection/query results missing: %#v", report)
	}
}

func TestRunRetrievalBenchmarkRequiresFixtureQueries(t *testing.T) {
	_, err := RunRetrievalBenchmark(RetrievalBenchmarkFixture{}, 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}
