package analysis

import (
	"math"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestLogOddsRatioCalculatesBinaryOutcomeEffect(t *testing.T) {
	es, variance, err := (LogOddsRatio{}).Calculate(map[string]string{
		"events_treatment": "30",
		"n_treatment":      "100",
		"events_control":   "20",
		"n_control":        "100",
	})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	want := math.Log((30.0 * 80.0) / (70.0 * 20.0))
	if math.Abs(es-want) > 1e-12 {
		t.Fatalf("effect size = %g, want %g", es, want)
	}
	if variance <= 0 {
		t.Fatalf("variance = %g, want positive", variance)
	}
}

func TestRiskRatioCalculatesBinaryOutcomeEffect(t *testing.T) {
	es, variance, err := (RiskRatio{}).Calculate(map[string]string{
		"events_treatment": "40",
		"n_treatment":      "100",
		"events_control":   "20",
		"n_control":        "100",
	})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	want := math.Log(2.0)
	if math.Abs(es-want) > 1e-12 {
		t.Fatalf("effect size = %g, want %g", es, want)
	}
	if variance <= 0 {
		t.Fatalf("variance = %g, want positive", variance)
	}
}

func TestPrepareWithCalculatorSupportsLogOddsRatio(t *testing.T) {
	items := []evidence.EvidenceItem{{
		PaperID: "paper-1",
		Values:  map[string]string{"events_treatment": "30", "n_treatment": "100", "events_control": "20", "n_control": "100"},
		Status:  evidence.StatusAccepted,
	}}
	run, err := PrepareWithCalculator("run-lor", items, LogOddsRatio{})
	if err != nil {
		t.Fatalf("PrepareWithCalculator returned error: %v", err)
	}
	if len(run.InputRows) != 1 || run.InputRows[0].PaperID != "paper-1" || run.InputRows[0].EffectSize == 0 {
		t.Fatalf("run = %#v", run)
	}
}
