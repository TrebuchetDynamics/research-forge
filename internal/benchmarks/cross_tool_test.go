package benchmarks

import "testing"

func TestBuildCrossToolBenchmarkReportCoversMetaAnalysisSpineMetrics(t *testing.T) {
	report := BuildCrossToolBenchmarkReport(CrossToolBenchmarkInput{
		DiscoveryRelevantFound: 18, DiscoveryRelevantTotal: 20,
		DedupeTrueMerges: 9, DedupeProposedMerges: 10,
		ParserCorrectFields: 45, ParserTotalFields: 50,
		ReferenceNormalized: 27, ReferenceTotal: 30,
		RetrievalRelevantAtK: 16, RetrievalRelevantTotal: 20,
		ScreeningBaselineRecords: 100, ScreeningModelRecords: 40,
		ReportReproducibleChecks: 5, ReportReproducibleTotal: 5,
		PackageReproducibleChecks: 4, PackageReproducibleTotal: 5,
	})
	want := []string{"discovery_recall", "dedupe_precision", "parser_accuracy", "reference_normalization", "retrieval_quality", "screening_effort_savings", "report_package_reproducibility"}
	for _, id := range want {
		if !report.HasMetric(id) {
			t.Fatalf("missing metric %s in %#v", id, report.Metrics)
		}
	}
	if report.Score("discovery_recall") != 0.9 || report.Score("screening_effort_savings") != 0.6 || report.Score("report_package_reproducibility") != 0.9 {
		t.Fatalf("unexpected scores: %#v", report.Metrics)
	}
	for _, want := range []string{"discovery", "dedupe", "parsing", "reference-normalization", "retrieval", "screening", "report-package"} {
		if !report.HasFixture(want) {
			t.Fatalf("missing deterministic fixture %s in %#v", want, report.Fixtures)
		}
	}
}

func TestDefaultCrossToolBenchmarkInputIsDeterministic(t *testing.T) {
	a := BuildCrossToolBenchmarkReport(DefaultCrossToolBenchmarkInput())
	b := BuildCrossToolBenchmarkReport(DefaultCrossToolBenchmarkInput())
	if len(a.Metrics) != 7 || a.Markdown != b.Markdown {
		t.Fatalf("default report not deterministic")
	}
}
