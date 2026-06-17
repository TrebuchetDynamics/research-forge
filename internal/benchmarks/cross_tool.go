package benchmarks

import (
	"fmt"
	"sort"
	"strings"
)

type CrossToolBenchmarkInput struct {
	DiscoveryRelevantFound, DiscoveryRelevantTotal      int
	DedupeTrueMerges, DedupeProposedMerges              int
	ParserCorrectFields, ParserTotalFields              int
	ReferenceNormalized, ReferenceTotal                 int
	RetrievalRelevantAtK, RetrievalRelevantTotal        int
	ScreeningBaselineRecords, ScreeningModelRecords     int
	ReportReproducibleChecks, ReportReproducibleTotal   int
	PackageReproducibleChecks, PackageReproducibleTotal int
}

type CrossToolBenchmarkReport struct {
	SchemaVersion string            `json:"schemaVersion"`
	Metrics       []BenchmarkMetric `json:"metrics"`
	Markdown      string            `json:"markdown"`
}

type BenchmarkMetric struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	Score          float64 `json:"score"`
	Numerator      int     `json:"numerator"`
	Denominator    int     `json:"denominator"`
	Interpretation string  `json:"interpretation"`
}

func DefaultCrossToolBenchmarkInput() CrossToolBenchmarkInput {
	return CrossToolBenchmarkInput{18, 20, 9, 10, 45, 50, 27, 30, 16, 20, 100, 40, 5, 5, 4, 5}
}

func BuildCrossToolBenchmarkReport(input CrossToolBenchmarkInput) CrossToolBenchmarkReport {
	report := CrossToolBenchmarkReport{SchemaVersion: "1"}
	report.Metrics = []BenchmarkMetric{
		metric("discovery_recall", "Discovery recall", input.DiscoveryRelevantFound, input.DiscoveryRelevantTotal, ratio(input.DiscoveryRelevantFound, input.DiscoveryRelevantTotal), "relevant seed/source records recovered"),
		metric("dedupe_precision", "Dedupe precision", input.DedupeTrueMerges, input.DedupeProposedMerges, ratio(input.DedupeTrueMerges, input.DedupeProposedMerges), "proposed identity merges that are true duplicates"),
		metric("parser_accuracy", "Parser accuracy", input.ParserCorrectFields, input.ParserTotalFields, ratio(input.ParserCorrectFields, input.ParserTotalFields), "parsed fields matching gold fixtures"),
		metric("reference_normalization", "Reference normalization", input.ReferenceNormalized, input.ReferenceTotal, ratio(input.ReferenceNormalized, input.ReferenceTotal), "references normalized to DOI/PMID/arXiv/ADS identifiers"),
		metric("retrieval_quality", "Retrieval quality", input.RetrievalRelevantAtK, input.RetrievalRelevantTotal, ratio(input.RetrievalRelevantAtK, input.RetrievalRelevantTotal), "relevant passages found at K"),
		metric("screening_effort_savings", "Screening effort savings", input.ScreeningBaselineRecords-input.ScreeningModelRecords, input.ScreeningBaselineRecords, effortSavings(input.ScreeningBaselineRecords, input.ScreeningModelRecords), "records avoided before target recall"),
		metric("report_package_reproducibility", "Report/package reproducibility", input.ReportReproducibleChecks+input.PackageReproducibleChecks, input.ReportReproducibleTotal+input.PackageReproducibleTotal, ratio(input.ReportReproducibleChecks+input.PackageReproducibleChecks, input.ReportReproducibleTotal+input.PackageReproducibleTotal), "report and package audit/replay checks passing"),
	}
	sort.Slice(report.Metrics, func(i, j int) bool { return report.Metrics[i].ID < report.Metrics[j].ID })
	report.Markdown = crossToolMarkdown(report)
	return report
}

func (r CrossToolBenchmarkReport) HasMetric(id string) bool {
	for _, m := range r.Metrics {
		if m.ID == id {
			return true
		}
	}
	return false
}
func (r CrossToolBenchmarkReport) Score(id string) float64 {
	for _, m := range r.Metrics {
		if m.ID == id {
			return m.Score
		}
	}
	return 0
}
func metric(id, label string, n, d int, score float64, interp string) BenchmarkMetric {
	return BenchmarkMetric{ID: id, Label: label, Numerator: n, Denominator: d, Score: score, Interpretation: interp}
}
func ratio(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / float64(d)
}
func effortSavings(baseline, model int) float64 {
	if baseline <= 0 || model > baseline {
		return 0
	}
	return float64(baseline-model) / float64(baseline)
}
func crossToolMarkdown(report CrossToolBenchmarkReport) string {
	var b strings.Builder
	b.WriteString("# Cross-tool benchmark report\n\n| Metric | Score | Evidence | Interpretation |\n| --- | ---: | --- | --- |\n")
	for _, m := range report.Metrics {
		fmt.Fprintf(&b, "| %s | %.3f | %d/%d | %s |\n", m.Label, m.Score, m.Numerator, m.Denominator, m.Interpretation)
	}
	return b.String()
}
