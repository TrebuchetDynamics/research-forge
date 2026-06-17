package analysis

import "testing"

func TestBeggRankCorrelationPublicationBiasDiagnostic(t *testing.T) {
	run := AnalysisRun{ID: "bias-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 0.1, Variance: 0.01}, {PaperID: "p2", EffectSize: 0.2, Variance: 0.04}, {PaperID: "p3", EffectSize: 0.4, Variance: 0.09}, {PaperID: "p4", EffectSize: 0.8, Variance: 0.16}}}
	report, err := BeggRankCorrelation(run)
	if err != nil {
		t.Fatalf("BeggRankCorrelation: %v", err)
	}
	if report.RunID != "bias-run" || report.Method != "begg-rank-correlation" || report.Studies != 4 || report.KendallTau == 0 {
		t.Fatalf("report = %#v", report)
	}
	if report.Warning == "" {
		t.Fatalf("expected small-study warning")
	}
}
