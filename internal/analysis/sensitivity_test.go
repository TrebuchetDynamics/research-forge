package analysis

import "testing"

func TestLeaveOneOutComputesInverseVarianceSensitivity(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{
		{PaperID: "paper-1", EffectSize: 1.0, Variance: 0.5},
		{PaperID: "paper-2", EffectSize: 2.0, Variance: 0.25},
		{PaperID: "paper-3", EffectSize: 4.0, Variance: 0.25},
	}}

	report, err := LeaveOneOut(run)
	if err != nil {
		t.Fatalf("LeaveOneOut returned error: %v", err)
	}
	if report.RunID != "run-1" || report.Method != "leave-one-out" || len(report.Rows) != 3 {
		t.Fatalf("report = %#v", report)
	}
	if report.Rows[0].OmittedPaperID != "paper-1" || report.Rows[0].Estimate != 3.0 || report.Rows[0].Studies != 2 {
		t.Fatalf("row[0] = %#v", report.Rows[0])
	}
	if report.Rows[1].OmittedPaperID != "paper-2" || report.Rows[1].Estimate != 3.0 || report.Rows[1].Variance != 1.0/6.0 {
		t.Fatalf("row[1] = %#v", report.Rows[1])
	}
}

func TestLeaveOneOutRequiresAtLeastTwoRows(t *testing.T) {
	_, err := LeaveOneOut(AnalysisRun{ID: "run-1", InputRows: []InputRow{{PaperID: "paper-1", EffectSize: 1, Variance: 1}}})
	if err == nil {
		t.Fatalf("expected error for too few rows")
	}
}
