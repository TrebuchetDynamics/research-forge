package quality

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing/model"
)

func TestArbitrationReportIncludesReconciliationOutputs(t *testing.T) {
	report := ArbitrateParserOutputs([]model.ParsedDocument{
		{PaperID: "paper-1", ParserName: "grobid", Title: "Title", Abstract: "Abstract"},
		{PaperID: "paper-1", ParserName: "s2orc", Title: "Other", Warnings: []string{"missing abstract"}},
	}, ArbitrationDecisionInput{AcceptedParser: "grobid", Reason: "reviewed field coverage"})
	if len(report.ReconciliationOutputs) == 0 {
		t.Fatalf("reconciliation outputs missing: %#v", report)
	}
	if report.ReconciliationOutputs[0].AcceptedParser != "grobid" || report.ReconciliationOutputs[0].Reason == "" || len(report.ReconciliationOutputs[0].Alternatives) == 0 {
		t.Fatalf("bad reconciliation output: %#v", report.ReconciliationOutputs[0])
	}
}
