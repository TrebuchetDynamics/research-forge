package analysis

import (
	"strings"
	"testing"
)

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
