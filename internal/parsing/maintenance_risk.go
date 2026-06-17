package parsing

import "sort"

type ParserBenchmarkFixture struct {
	ParserName         string
	FixtureID          string
	CorrectFields      int
	TotalFields        int
	ActiveMaintenance  bool
	HistoricalFallback bool
}

type ParserMaintenanceRiskReport struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Parsers       []ParserMaintenanceRisk `json:"parsers"`
}

type ParserMaintenanceRisk struct {
	ParserName         string  `json:"parserName"`
	FixtureCount       int     `json:"fixtureCount"`
	Accuracy           float64 `json:"accuracy"`
	ActiveMaintenance  bool    `json:"activeMaintenance"`
	HistoricalFallback bool    `json:"historicalFallback"`
	RiskLevel          string  `json:"riskLevel"`
	ReviewerGate       string  `json:"reviewerGate"`
	EnableFallback     bool    `json:"enableFallback"`
}

func DefaultParserBenchmarkFixtures() []ParserBenchmarkFixture {
	return []ParserBenchmarkFixture{{"grobid", "gold", 9, 10, true, false}, {"s2orc-doc2json", "gold", 8, 10, true, false}, {"papermage", "gold", 8, 10, true, false}, {"cermine", "gold", 7, 10, true, false}, {"science-parse", "gold", 7, 10, false, true}, {"anystyle", "references", 9, 10, true, false}}
}

func BuildParserMaintenanceRiskReport(fixtures []ParserBenchmarkFixture) ParserMaintenanceRiskReport {
	byParser := map[string][]ParserBenchmarkFixture{}
	for _, fixture := range fixtures {
		byParser[canonicalParserName(fixture.ParserName)] = append(byParser[canonicalParserName(fixture.ParserName)], fixture)
	}
	report := ParserMaintenanceRiskReport{SchemaVersion: "1"}
	for parser, list := range byParser {
		report.Parsers = append(report.Parsers, scoreParserMaintenance(parser, list))
	}
	sort.Slice(report.Parsers, func(i, j int) bool { return report.Parsers[i].ParserName < report.Parsers[j].ParserName })
	return report
}
func scoreParserMaintenance(parser string, fixtures []ParserBenchmarkFixture) ParserMaintenanceRisk {
	total, correct := 0, 0
	active, historical := false, false
	for _, f := range fixtures {
		total += f.TotalFields
		correct += f.CorrectFields
		active = active || f.ActiveMaintenance
		historical = historical || f.HistoricalFallback
	}
	acc := 0.0
	if total > 0 {
		acc = float64(correct) / float64(total)
	}
	risk := "low"
	gate := "none"
	enable := true
	if historical || !active {
		risk = "high"
		gate = "maintenance-risk-review"
		enable = false
	} else if acc < 0.75 {
		risk = "medium"
		gate = "parser-benchmark-review"
		enable = false
	}
	return ParserMaintenanceRisk{ParserName: parser, FixtureCount: len(fixtures), Accuracy: acc, ActiveMaintenance: active, HistoricalFallback: historical, RiskLevel: risk, ReviewerGate: gate, EnableFallback: enable}
}
func (r ParserMaintenanceRiskReport) HasParser(parser string) bool {
	return r.Parser(parser).ParserName != ""
}
func (r ParserMaintenanceRiskReport) Parser(parser string) ParserMaintenanceRisk {
	parser = canonicalParserName(parser)
	for _, p := range r.Parsers {
		if p.ParserName == parser {
			return p
		}
	}
	return ParserMaintenanceRisk{}
}
