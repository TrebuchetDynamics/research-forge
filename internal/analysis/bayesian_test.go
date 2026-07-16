package analysis

import (
	"math"
	"testing"
)

func TestBayesianNormalApproximationCombinesPriorAndStudies(t *testing.T) {
	run := AnalysisRun{ID: "bayes-run", InputRows: []InputRow{
		{PaperID: "p1", EffectSize: 1, Variance: 1},
		{PaperID: "p2", EffectSize: 3, Variance: 1},
	}}

	report, err := BayesianNormalApproximation(run, 0, 1)
	if err != nil {
		t.Fatalf("BayesianNormalApproximation returned error: %v", err)
	}
	if report.RunID != "bayes-run" || report.Method != "normal-approx" || report.Studies != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.PosteriorMean <= 1.3 || report.PosteriorMean >= 1.4 || report.PosteriorVar <= 0 {
		t.Fatalf("posterior = %#v", report)
	}
	if report.CredibleLow95 >= report.PosteriorMean || report.CredibleHigh95 <= report.PosteriorMean {
		t.Fatalf("credible interval = %#v", report)
	}
}

func TestBayesianNormalApproximationRequiresPositivePriorVariance(t *testing.T) {
	_, err := BayesianNormalApproximation(AnalysisRun{ID: "bayes-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}}}, 0, 0)
	if err == nil {
		t.Fatalf("expected prior variance error")
	}
}

func TestBayesianNormalApproximationRejectsNonfiniteRows(t *testing.T) {
	run := AnalysisRun{ID: "bayes-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: math.NaN()}}}
	if _, err := BayesianNormalApproximation(run, 0, 1); err == nil {
		t.Fatal("BayesianNormalApproximation returned nil error for a non-finite variance")
	}
}

func TestBayesianNormalApproximationRejectsNonfinitePrior(t *testing.T) {
	run := AnalysisRun{ID: "bayes-run", InputRows: []InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}}}
	for name, prior := range map[string][2]float64{
		"mean":     {math.NaN(), 1},
		"variance": {0, math.NaN()},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := BayesianNormalApproximation(run, prior[0], prior[1]); err == nil {
				t.Fatal("BayesianNormalApproximation returned nil error for a non-finite prior")
			}
		})
	}
}
