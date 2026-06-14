package analysis

import "fmt"

// BayesianReport records a lightweight normal-normal Bayesian meta-analysis approximation.
type BayesianReport struct {
	RunID          string  `json:"runId"`
	Method         string  `json:"method"`
	Studies        int     `json:"studies"`
	PriorMean      float64 `json:"priorMean"`
	PriorVariance  float64 `json:"priorVariance"`
	PosteriorMean  float64 `json:"posteriorMean"`
	PosteriorVar   float64 `json:"posteriorVariance"`
	CredibleLow95  float64 `json:"credibleLow95"`
	CredibleHigh95 float64 `json:"credibleHigh95"`
}

// BayesianNormalApproximation computes a conjugate normal fixed-effect approximation.
func BayesianNormalApproximation(run AnalysisRun, priorMean, priorVariance float64) (BayesianReport, error) {
	if len(run.InputRows) == 0 {
		return BayesianReport{}, fmt.Errorf("bayesian analysis requires input rows")
	}
	if priorVariance <= 0 {
		return BayesianReport{}, fmt.Errorf("prior variance must be positive")
	}
	precision := 1 / priorVariance
	weighted := priorMean * precision
	for _, row := range run.InputRows {
		if row.Variance <= 0 {
			return BayesianReport{}, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		rowPrecision := 1 / row.Variance
		precision += rowPrecision
		weighted += row.EffectSize * rowPrecision
	}
	posteriorVariance := 1 / precision
	posteriorMean := weighted / precision
	margin := 1.96 * sqrtFloat(posteriorVariance)
	return BayesianReport{RunID: run.ID, Method: "normal-approx", Studies: len(run.InputRows), PriorMean: priorMean, PriorVariance: priorVariance, PosteriorMean: posteriorMean, PosteriorVar: posteriorVariance, CredibleLow95: posteriorMean - margin, CredibleHigh95: posteriorMean + margin}, nil
}

func sqrtFloat(value float64) float64 {
	guess := value
	if guess <= 0 {
		return 0
	}
	for i := 0; i < 20; i++ {
		guess = 0.5 * (guess + value/guess)
	}
	return guess
}
