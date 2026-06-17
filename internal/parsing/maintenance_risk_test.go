package parsing

import "testing"

func TestParserMaintenanceRiskScoringFlagsScienceParseHistoricalFallback(t *testing.T) {
	report := BuildParserMaintenanceRiskReport([]ParserBenchmarkFixture{
		{ParserName: "grobid", FixtureID: "gold", CorrectFields: 9, TotalFields: 10, ActiveMaintenance: true},
		{ParserName: "papermage", FixtureID: "gold", CorrectFields: 8, TotalFields: 10, ActiveMaintenance: true},
		{ParserName: "science-parse", FixtureID: "gold", CorrectFields: 7, TotalFields: 10, ActiveMaintenance: false, HistoricalFallback: true},
	})
	if !report.HasParser("science-parse") || report.Parser("science-parse").EnableFallback {
		t.Fatalf("science-parse risk not gated: %#v", report)
	}
	if report.Parser("science-parse").RiskLevel != "high" || report.Parser("science-parse").ReviewerGate != "maintenance-risk-review" {
		t.Fatalf("science-parse score = %#v", report.Parser("science-parse"))
	}
	if !report.Parser("grobid").EnableFallback {
		t.Fatalf("active parser should be allowed: %#v", report.Parser("grobid"))
	}
}

func TestDefaultParserBenchmarkFixturesCoverCandidates(t *testing.T) {
	report := BuildParserMaintenanceRiskReport(DefaultParserBenchmarkFixtures())
	for _, parser := range []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"} {
		if !report.HasParser(parser) {
			t.Fatalf("missing parser %s", parser)
		}
	}
}
