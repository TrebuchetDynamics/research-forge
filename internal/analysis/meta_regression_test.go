package analysis

import (
	"math"
	"testing"
)

func TestMetaRegressionFitsWeightedModeratorSlope(t *testing.T) {
	run := AnalysisRun{ID: "reg-run", InputRows: []InputRow{
		{PaperID: "p1", EffectSize: 1, Variance: 1},
		{PaperID: "p2", EffectSize: 2, Variance: 1},
		{PaperID: "p3", EffectSize: 3, Variance: 1},
	}}

	report, err := MetaRegression(run, "dose", map[string]float64{"p1": 1, "p2": 2, "p3": 3})
	if err != nil {
		t.Fatalf("MetaRegression returned error: %v", err)
	}
	if report.RunID != "reg-run" || report.Moderator != "dose" || report.Studies != 3 || report.Intercept != 0 || report.Slope != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestMetaRegressionRequiresModeratorValues(t *testing.T) {
	run := AnalysisRun{ID: "reg-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 2, Variance: 1}}}
	_, err := MetaRegression(run, "dose", map[string]float64{"p1": 1})
	if err == nil {
		t.Fatalf("expected missing moderator value error")
	}
}

func TestMetaRegressionRejectsConstantModerator(t *testing.T) {
	run := AnalysisRun{ID: "reg-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 2, Variance: 1}}}
	if _, err := MetaRegression(run, "dose", map[string]float64{"p1": 1, "p2": 1}); err == nil {
		t.Fatal("MetaRegression returned nil error for a constant moderator")
	}
}

func TestMetaRegressionRejectsNonfiniteRows(t *testing.T) {
	run := AnalysisRun{ID: "reg-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: math.Inf(1), Variance: 1}, {PaperID: "p2", EffectSize: 2, Variance: 1}}}
	if _, err := MetaRegression(run, "dose", map[string]float64{"p1": 1, "p2": 2}); err == nil {
		t.Fatal("MetaRegression returned nil error for a non-finite effect size")
	}
}

func TestMetaRegressionRejectsNonfiniteModerator(t *testing.T) {
	run := AnalysisRun{ID: "reg-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 2, Variance: 1}}}
	if _, err := MetaRegression(run, "dose", map[string]float64{"p1": math.NaN(), "p2": 2}); err == nil {
		t.Fatal("MetaRegression returned nil error for a non-finite moderator")
	}
}
