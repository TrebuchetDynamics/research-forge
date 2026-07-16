package analysis

import (
	"math"
	"testing"
)

func TestCompareAnalysisEnginesCapturesLocksParityWarningsDeltasAndDisagreement(t *testing.T) {
	run := AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 3, Variance: 1}}}
	metafor := EngineResult{Engine: "metafor", Estimate: 2, Variance: 0.5, Versions: map[string]string{"R": "4.3", "metafor": "4.0"}, Warnings: []string{"metafor warning"}, ModelSettings: DefaultEngineModelSettings()}
	pymare := EngineResult{Engine: "pymare", Estimate: 2.25, Variance: 0.55, Versions: map[string]string{"python": "3.11", "pymare": "0.0-fixture"}, Warnings: []string{"pymare warning"}, ModelSettings: DefaultEngineModelSettings()}
	report := CompareAnalysisEngines(run, metafor, pymare, 0.1)
	if report.SchemaVersion != "1" || report.RunID != "run1" || len(report.EnvironmentLocks) != 2 || !report.ModelSettingParity {
		t.Fatalf("report metadata = %#v", report)
	}
	if report.OutputDeltas.EstimateDelta != 0.25 || math.Abs(report.OutputDeltas.VarianceDelta-0.05) > 1e-12 {
		t.Fatalf("deltas = %#v", report.OutputDeltas)
	}
	if len(report.Warnings) != 2 || !report.Disagreement.RequiresReview || report.Disagreement.Reason == "" {
		t.Fatalf("warnings/disagreement = %#v %#v", report.Warnings, report.Disagreement)
	}
}

func TestCompareAnalysisEnginesRequiresReviewForNonfiniteOutput(t *testing.T) {
	run := AnalysisRun{SchemaVersion: "1", ID: "run1"}
	settings := DefaultEngineModelSettings()
	primary := EngineResult{Engine: "metafor", Estimate: 2, Variance: 0.5, ModelSettings: settings}
	secondary := EngineResult{Engine: "pymare", Estimate: math.NaN(), Variance: 0.5, ModelSettings: settings}
	report := CompareAnalysisEngines(run, primary, secondary, 0.1)
	if !report.Disagreement.RequiresReview {
		t.Fatalf("non-finite engine output did not require review: %#v", report.Disagreement)
	}
}

func TestCompareAnalysisEnginesRequiresReviewForNegativeVariance(t *testing.T) {
	run := AnalysisRun{SchemaVersion: "1", ID: "run1"}
	settings := DefaultEngineModelSettings()
	primary := EngineResult{Engine: "metafor", Estimate: 2, Variance: -1, ModelSettings: settings}
	secondary := EngineResult{Engine: "pymare", Estimate: 2, Variance: -1, ModelSettings: settings}
	report := CompareAnalysisEngines(run, primary, secondary, 0.1)
	if !report.Disagreement.RequiresReview {
		t.Fatalf("negative engine variances did not require review: %#v", report.Disagreement)
	}
}

func TestBuildPyMAREFixtureResultUsesSameInputSnapshot(t *testing.T) {
	run := AnalysisRun{ID: "run1", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 3, Variance: 1}}}
	result, err := BuildPyMAREFixtureResult(run, 0)
	if err != nil {
		t.Fatalf("BuildPyMAREFixtureResult: %v", err)
	}
	if result.Engine != "pymare-fixture" || result.Estimate != 2 || result.Variance != 0.5 || result.InputHash == "" || result.Versions["pymare"] == "" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBuildPyMAREFixtureResultRejectsNonfiniteRows(t *testing.T) {
	run := AnalysisRun{ID: "run1", InputRows: []InputRow{{PaperID: "p1", EffectSize: math.NaN(), Variance: 1}}}
	if _, err := BuildPyMAREFixtureResult(run, 0); err == nil {
		t.Fatal("BuildPyMAREFixtureResult returned nil error for a non-finite effect size")
	}
}
