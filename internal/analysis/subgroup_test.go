package analysis

import "testing"

func TestSubgroupAnalysisComputesPooledGroupEstimates(t *testing.T) {
	run := AnalysisRun{ID: "sub-run", InputRows: []InputRow{
		{PaperID: "p1", EffectSize: 1, Variance: 1},
		{PaperID: "p2", EffectSize: 3, Variance: 1},
		{PaperID: "p3", EffectSize: 10, Variance: 2},
	}}

	report, err := SubgroupAnalysis(run, "region", map[string]string{"p1": "EU", "p2": "EU", "p3": "US"})
	if err != nil {
		t.Fatalf("SubgroupAnalysis returned error: %v", err)
	}
	if report.RunID != "sub-run" || report.Variable != "region" || len(report.Groups) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.Groups[0].Group != "EU" || report.Groups[0].Studies != 2 || report.Groups[0].Estimate != 2 || report.Groups[0].Variance != 0.5 {
		t.Fatalf("EU group = %#v", report.Groups[0])
	}
	if report.Groups[1].Group != "US" || report.Groups[1].Studies != 1 || report.Groups[1].Estimate != 10 || report.Groups[1].Variance != 2 {
		t.Fatalf("US group = %#v", report.Groups[1])
	}
}

func TestSubgroupAnalysisRequiresAllGroupValues(t *testing.T) {
	run := AnalysisRun{ID: "sub-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}}}
	_, err := SubgroupAnalysis(run, "region", map[string]string{})
	if err == nil {
		t.Fatalf("expected missing group error")
	}
}
