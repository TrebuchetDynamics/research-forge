package analysis

import (
	"math"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestRawContinuousOutcomeUsesCI(t *testing.T) {
	calc := RawContinuousOutcome{VarianceFloor: 0.0025}
	res, err := calc.CalculateRaw(map[string]string{
		"value_pct": "12.5",
		"ci_lower":  "11.0",
		"ci_upper":  "14.0",
	})
	if err != nil {
		t.Fatalf("CalculateRaw error: %v", err)
	}
	if res.Yi != 12.5 {
		t.Fatalf("yi = %g, want 12.5", res.Yi)
	}
	wantSE := (14.0 - 11.0) / (2 * 1.96)
	wantVi := wantSE * wantSE
	if math.Abs(res.Vi-wantVi) > 1e-10 {
		t.Fatalf("vi = %g, want %g", res.Vi, wantVi)
	}
	if res.ViSource != "ci" {
		t.Fatalf("vi_source = %q, want \"ci\"", res.ViSource)
	}
}

func TestRawContinuousOutcomeUsesSE(t *testing.T) {
	calc := RawContinuousOutcome{VarianceFloor: 0.0025}
	res, err := calc.CalculateRaw(map[string]string{"value_pct": "8.3", "se": "0.4"})
	if err != nil {
		t.Fatalf("CalculateRaw error: %v", err)
	}
	if math.Abs(res.Vi-0.16) > 1e-10 {
		t.Fatalf("vi = %g, want 0.16", res.Vi)
	}
	if res.ViSource != "se" {
		t.Fatalf("vi_source = %q, want \"se\"", res.ViSource)
	}
}

func TestRawContinuousOutcomeFallsBackToFloor(t *testing.T) {
	calc := RawContinuousOutcome{VarianceFloor: 0.0025}
	res, err := calc.CalculateRaw(map[string]string{"value_pct": "5.0"})
	if err != nil {
		t.Fatalf("CalculateRaw error: %v", err)
	}
	if res.Vi != 0.0025 {
		t.Fatalf("vi = %g, want 0.0025", res.Vi)
	}
	if res.ViSource != "floor" {
		t.Fatalf("vi_source = %q, want \"floor\"", res.ViSource)
	}
}

func TestRawContinuousOutcomeUsesDefaultFloor(t *testing.T) {
	calc := RawContinuousOutcome{} // zero VarianceFloor → defaults to 0.0025
	res, err := calc.CalculateRaw(map[string]string{"value_pct": "3.0"})
	if err != nil {
		t.Fatalf("CalculateRaw error: %v", err)
	}
	if res.Vi != 0.0025 {
		t.Fatalf("vi = %g, want default 0.0025", res.Vi)
	}
}

func TestRawContinuousOutcomeErrorsWithoutValuePct(t *testing.T) {
	calc := RawContinuousOutcome{VarianceFloor: 0.0025}
	_, err := calc.CalculateRaw(map[string]string{"se": "0.5"})
	if err == nil {
		t.Fatal("expected error when value_pct missing")
	}
}

func TestRawContinuousOutcomeImplementsEffectSizeCalculator(t *testing.T) {
	calc := RawContinuousOutcome{VarianceFloor: 0.0025}
	yi, vi, err := calc.Calculate(map[string]string{"value_pct": "10.0"})
	if err != nil || yi != 10.0 || vi != 0.0025 {
		t.Fatalf("Calculate = %g, %g, %v", yi, vi, err)
	}
}

func TestPrepareRawContinuousPopulatesViSourceAndModerators(t *testing.T) {
	items := []evidence.EvidenceItem{
		{
			PaperID: "paper-a",
			Status:  evidence.StatusAccepted,
			Values: map[string]string{
				"value_pct":            "11.0",
				"ci_lower":             "10.0",
				"ci_upper":             "12.0",
				"device_type":          "pec",
				"auxiliary_bias":       "unassisted",
				"measurement_standard": "am1.5g-100",
			},
		},
		{
			PaperID: "paper-b",
			Status:  evidence.StatusAccepted,
			Values:  map[string]string{"value_pct": "7.5", "device_type": "pv-electrolysis", "auxiliary_bias": "assisted", "measurement_standard": "non-standard"},
		},
	}
	run, err := PrepareRawContinuous("run-rc", items, 0.0025, []string{"device_type", "auxiliary_bias", "measurement_standard"})
	if err != nil {
		t.Fatalf("PrepareRawContinuous error: %v", err)
	}
	if len(run.InputRows) != 2 {
		t.Fatalf("rows = %d, want 2", len(run.InputRows))
	}
	if run.InputRows[0].ViSource != "ci" {
		t.Fatalf("row[0] vi_source = %q, want ci", run.InputRows[0].ViSource)
	}
	if run.InputRows[1].ViSource != "floor" {
		t.Fatalf("row[1] vi_source = %q, want floor", run.InputRows[1].ViSource)
	}
	if run.InputRows[0].Moderators["device_type"] != "pec" {
		t.Fatalf("row[0] moderators = %v", run.InputRows[0].Moderators)
	}
	if run.InputRows[1].Moderators["auxiliary_bias"] != "assisted" {
		t.Fatalf("row[1] moderators = %v", run.InputRows[1].Moderators)
	}
}

func TestExcludeByViSourceRemovesFloorRows(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{
		{PaperID: "a", EffectSize: 10, Variance: 0.04, ViSource: "ci"},
		{PaperID: "b", EffectSize: 8, Variance: 0.0025, ViSource: "floor"},
		{PaperID: "c", EffectSize: 12, Variance: 0.09, ViSource: "se"},
	}}
	filtered := ExcludeByViSource(run, "floor")
	if len(filtered.InputRows) != 2 {
		t.Fatalf("rows = %d, want 2", len(filtered.InputRows))
	}
	for _, row := range filtered.InputRows {
		if row.ViSource == "floor" {
			t.Fatalf("floor row not excluded: %v", row)
		}
	}
}

func TestBenchmarkingReadinessPassesWhenAllFieldsPresent(t *testing.T) {
	items := []evidence.EvidenceItem{{
		PaperID: "paper-x",
		Status:  evidence.StatusAccepted,
		Values: map[string]string{
			"value_pct":            "9.0",
			"device_type":          "pec",
			"auxiliary_bias":       "unassisted",
			"measurement_standard": "am1.5g-100",
		},
	}}
	report := BenchmarkingReadiness("run-1", items, nil)
	if !report.Ready || report.TotalItems != 1 || report.ReadyItems != 1 || len(report.Issues) != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestBenchmarkingReadinessFailsWhenFieldsMissing(t *testing.T) {
	items := []evidence.EvidenceItem{
		{
			PaperID: "paper-y",
			Status:  evidence.StatusAccepted,
			Values:  map[string]string{"value_pct": "6.0", "device_type": "pec"},
			// missing auxiliary_bias and measurement_standard
		},
	}
	report := BenchmarkingReadiness("run-1", items, nil)
	if report.Ready {
		t.Fatal("expected not ready")
	}
	if len(report.Issues) != 2 {
		t.Fatalf("issues = %v, want 2", report.Issues)
	}
}

func TestBenchmarkingReadinessSkipsNonAccepted(t *testing.T) {
	items := []evidence.EvidenceItem{
		{PaperID: "s", Status: evidence.StatusSuggested, Values: map[string]string{}},
		{PaperID: "r", Status: evidence.StatusRejected, Values: map[string]string{}},
	}
	report := BenchmarkingReadiness("run-1", items, nil)
	if report.TotalItems != 0 || report.Ready {
		t.Fatalf("report = %+v", report)
	}
}

func TestGenerateMetaforScriptIncludesModeratorColumns(t *testing.T) {
	run := AnalysisRun{ID: "run-mod", InputRows: []InputRow{
		{PaperID: "a", EffectSize: 10, Variance: 0.04, Moderators: map[string]string{"device_type": "pec", "auxiliary_bias": "unassisted"}},
		{PaperID: "b", EffectSize: 8, Variance: 0.0025, Moderators: map[string]string{"device_type": "pv-electrolysis", "auxiliary_bias": "assisted"}},
	}}
	script := GenerateMetaforScript(run)
	if !strings.Contains(script, "device_type=c(") {
		t.Fatalf("script missing device_type column:\n%s", script)
	}
	if !strings.Contains(script, "auxiliary_bias=c(") {
		t.Fatalf("script missing auxiliary_bias column:\n%s", script)
	}
	if !strings.Contains(script, "mods = ~auxiliary_bias+device_type") {
		t.Fatalf("script missing mods formula:\n%s", script)
	}
	if !strings.Contains(script, `"pec"`) {
		t.Fatalf("script missing pec value:\n%s", script)
	}
}

func TestGenerateMetaforScriptNoModsWhenNoModerators(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{{PaperID: "a", EffectSize: 1, Variance: 0.1}}}
	script := GenerateMetaforScript(run)
	if strings.Contains(script, "mods") {
		t.Fatalf("script should not contain mods for run without moderators:\n%s", script)
	}
	if !strings.Contains(script, "rma(yi = yi, vi = vi, data=data)") {
		t.Fatalf("script missing plain rma call:\n%s", script)
	}
}

func TestGenerateMetaforScriptModeratorMissingValueBecomesNA(t *testing.T) {
	run := AnalysisRun{ID: "run-1", InputRows: []InputRow{
		{PaperID: "a", EffectSize: 10, Variance: 0.04, Moderators: map[string]string{"device_type": "pec"}},
		{PaperID: "b", EffectSize: 8, Variance: 0.0025}, // no moderators
	}}
	script := GenerateMetaforScript(run)
	if !strings.Contains(script, "NA") {
		t.Fatalf("script should contain NA for missing moderator value:\n%s", script)
	}
}
