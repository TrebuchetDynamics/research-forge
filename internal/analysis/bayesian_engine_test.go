package analysis

import (
	"math"
	"strings"
	"testing"
)

type recordingBayesianEngine struct {
	called bool
}

func (e *recordingBayesianEngine) Name() string { return "recording" }

func (e *recordingBayesianEngine) Run(AnalysisRun, BayesianEngineOptions) (BayesianReport, error) {
	e.called = true
	return BayesianReport{}, nil
}

func TestBayesianEnginePathIsSeparateFromNormalApproximation(t *testing.T) {
	run := AnalysisRun{ID: "bayes-engine", InputRows: []InputRow{{PaperID: "p1", EffectSize: 0.2, Variance: 0.04}, {PaperID: "p2", EffectSize: 0.4, Variance: 0.09}}}
	report, err := RunBayesianEngine(run, GridBayesianEngine{GridPoints: 101}, BayesianEngineOptions{PriorMean: 0, PriorVariance: 1})
	if err != nil {
		t.Fatalf("RunBayesianEngine: %v", err)
	}
	if report.Method != "grid-bayesian-engine" || report.Engine != "grid" || report.PosteriorMean == 0 || report.CredibleLow95 >= report.CredibleHigh95 {
		t.Fatalf("report = %#v", report)
	}
	approx, err := BayesianNormalApproximation(run, 0, 1)
	if err != nil {
		t.Fatalf("approx: %v", err)
	}
	if report.Method == approx.Method {
		t.Fatalf("engine path reused normal approx method")
	}
}

func TestGridBayesianEngineReturnsFinitePosteriorForExtremeDisagreement(t *testing.T) {
	run := AnalysisRun{ID: "bayes-extreme", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1_000_000, Variance: 0.000001}}}
	report, err := RunBayesianEngine(run, GridBayesianEngine{GridPoints: 201}, BayesianEngineOptions{PriorMean: 0, PriorVariance: 1})
	if err != nil {
		t.Fatalf("RunBayesianEngine returned error: %v", err)
	}
	for name, value := range map[string]float64{
		"posterior mean": report.PosteriorMean,
		"credible low":   report.CredibleLow95,
		"credible high":  report.CredibleHigh95,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Errorf("%s is not finite: %v", name, value)
		}
	}
	if report.CredibleLow95 >= report.CredibleHigh95 {
		t.Errorf("credible interval is not ordered: [%v, %v]", report.CredibleLow95, report.CredibleHigh95)
	}
}

func TestRunBayesianEngineRejectsNonfiniteRowsBeforeDelegating(t *testing.T) {
	engine := &recordingBayesianEngine{}
	run := AnalysisRun{ID: "bayes-invalid", InputRows: []InputRow{{PaperID: "p1", EffectSize: math.NaN(), Variance: 1}}}
	if _, err := RunBayesianEngine(run, engine, BayesianEngineOptions{PriorVariance: 1}); err == nil {
		t.Fatal("RunBayesianEngine returned nil error for a non-finite effect size")
	}
	if engine.called {
		t.Fatal("RunBayesianEngine delegated invalid rows to the engine")
	}
}

func TestRunBayesianEngineRejectsNonfinitePriorBeforeDelegating(t *testing.T) {
	engine := &recordingBayesianEngine{}
	run := AnalysisRun{ID: "bayes-invalid", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}}}
	if _, err := RunBayesianEngine(run, engine, BayesianEngineOptions{PriorMean: math.NaN(), PriorVariance: 1}); err == nil {
		t.Fatal("RunBayesianEngine returned nil error for a non-finite prior mean")
	}
	if engine.called {
		t.Fatal("RunBayesianEngine delegated a non-finite prior to the engine")
	}
}

func TestPublicationReadyPlotStylingAddsAccessibleCSS(t *testing.T) {
	run := AnalysisRun{ID: "style", InputRows: []InputRow{{PaperID: "p1", EffectSize: 0.1, Variance: 0.04}}}
	forest := forestPlotSVG(run)
	funnel := funnelPlotSVG(run)
	for _, svg := range []string{forest, funnel} {
		if !containsAll(svg, []string{"font-family", "ResearchForge publication-ready", "stroke-width"}) {
			t.Fatalf("plot missing publication styling: %s", svg)
		}
	}
}

func containsAll(text string, wants []string) bool {
	for _, want := range wants {
		if !strings.Contains(text, want) {
			return false
		}
	}
	return true
}
