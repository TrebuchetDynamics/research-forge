package analysis

import "testing"

func TestInfluenceDiagnosticsAndRichSensitivityArtifacts(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 2, Variance: 1}, {PaperID: "p3", EffectSize: 10, Variance: 1}}}
	influence, err := InfluenceDiagnostics(run)
	if err != nil {
		t.Fatalf("InfluenceDiagnostics: %v", err)
	}
	if influence.RunID != "run-1" || influence.BaselineEstimate == 0 || len(influence.Rows) != 3 || influence.Rows[2].AbsoluteDelta <= influence.Rows[0].AbsoluteDelta {
		t.Fatalf("influence = %#v", influence)
	}
	sensitivity, err := LeaveOneOut(run)
	if err != nil {
		t.Fatalf("LeaveOneOut: %v", err)
	}
	if sensitivity.BaselineEstimate == 0 || sensitivity.MaxAbsoluteDelta == 0 || sensitivity.Rows[0].Delta == 0 {
		t.Fatalf("sensitivity = %#v", sensitivity)
	}
}
