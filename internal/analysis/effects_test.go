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

func TestAdditionalEffectCalculatorsMeanDifferenceRiskDifferenceAndCorrelation(t *testing.T) {
	md, mdVar, err := (MeanDifference{}).Calculate(map[string]string{"mean_treatment": "10", "mean_control": "7", "sd_treatment": "4", "sd_control": "5", "n_treatment": "50", "n_control": "60"})
	if err != nil || math.Abs(md-3) > 1e-12 || mdVar <= 0 {
		t.Fatalf("mean difference = %g var=%g err=%v", md, mdVar, err)
	}
	rd, rdVar, err := (RiskDifference{}).Calculate(map[string]string{"events_treatment": "30", "n_treatment": "100", "events_control": "20", "n_control": "100"})
	if err != nil || math.Abs(rd-0.1) > 1e-12 || rdVar <= 0 {
		t.Fatalf("risk difference = %g var=%g err=%v", rd, rdVar, err)
	}
	z, zVar, err := (FisherZCorrelation{}).Calculate(map[string]string{"correlation": "0.5", "n": "30"})
	if err != nil || math.Abs(z-math.Atanh(0.5)) > 1e-12 || math.Abs(zVar-(1.0/27.0)) > 1e-12 {
		t.Fatalf("fisher z = %g var=%g err=%v", z, zVar, err)
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
