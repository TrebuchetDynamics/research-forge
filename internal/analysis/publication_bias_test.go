package analysis

import "testing"

func TestEggerRegressionComputesPublicationBiasDiagnostic(t *testing.T) {
	run := AnalysisRun{ID: "bias-run", InputRows: []InputRow{
		{PaperID: "p1", EffectSize: 0.2, Variance: 0.04},
		{PaperID: "p2", EffectSize: 0.3, Variance: 0.05},
		{PaperID: "p3", EffectSize: 0.4, Variance: 0.06},
	}}

	report, err := EggerRegression(run)
	if err != nil {
		t.Fatalf("EggerRegression returned error: %v", err)
	}
	if report.RunID != "bias-run" || report.Method != "egger" || report.Studies != 3 {
		t.Fatalf("report = %#v", report)
	}
	if report.Warning == "" {
		t.Fatalf("expected underpowered warning: %#v", report)
	}
}

func TestEggerRegressionRequiresAtLeastThreeStudies(t *testing.T) {
	_, err := EggerRegression(AnalysisRun{ID: "bias-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}}})
	if err == nil {
		t.Fatalf("expected error for too few studies")
	}
}
