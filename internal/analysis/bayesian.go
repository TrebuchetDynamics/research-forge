package analysis

import (
	"fmt"
	"math"
)

// BayesianReport records a lightweight normal-normal Bayesian meta-analysis approximation.
type BayesianReport struct {
	RunID          string  `json:"runId"`
	Method         string  `json:"method"`
	Engine         string  `json:"engine,omitempty"`
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
	if err := validateAnalysisRows(run.InputRows); err != nil {
		return BayesianReport{}, err
	}
	if err := validateBayesianPrior(priorMean, priorVariance); err != nil {
		return BayesianReport{}, err
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

type BayesianEngineOptions struct{ PriorMean, PriorVariance float64 }

type BayesianEngine interface {
	Name() string
	Run(AnalysisRun, BayesianEngineOptions) (BayesianReport, error)
}

type GridBayesianEngine struct{ GridPoints int }

func (GridBayesianEngine) Name() string { return "grid" }

func RunBayesianEngine(run AnalysisRun, engine BayesianEngine, opts BayesianEngineOptions) (BayesianReport, error) {
	if engine == nil {
		return BayesianReport{}, fmt.Errorf("bayesian engine is required")
	}
	if err := validateAnalysisRows(run.InputRows); err != nil {
		return BayesianReport{}, err
	}
	if opts.PriorVariance <= 0 {
		opts.PriorVariance = 100
	}
	if err := validateBayesianPrior(opts.PriorMean, opts.PriorVariance); err != nil {
		return BayesianReport{}, err
	}
	return engine.Run(run, opts)
}

func validateBayesianPrior(priorMean, priorVariance float64) error {
	if math.IsNaN(priorMean) || math.IsInf(priorMean, 0) {
		return fmt.Errorf("prior mean must be finite")
	}
	if priorVariance <= 0 || math.IsNaN(priorVariance) || math.IsInf(priorVariance, 0) {
		return fmt.Errorf("prior variance must be finite and positive")
	}
	return nil
}

func (g GridBayesianEngine) Run(run AnalysisRun, opts BayesianEngineOptions) (BayesianReport, error) {
	approx, err := BayesianNormalApproximation(run, opts.PriorMean, opts.PriorVariance)
	if err != nil {
		return BayesianReport{}, err
	}
	points := g.GridPoints
	if points < 21 {
		points = 201
	}
	width := 6 * math.Sqrt(approx.PosteriorVar)
	if width == 0 {
		width = 1
	}
	start := approx.PosteriorMean - width
	step := 2 * width / float64(points-1)
	logWeights := make([]float64, points)
	maxLogWeight := math.Inf(-1)
	for i := 0; i < points; i++ {
		theta := start + float64(i)*step
		logp := -0.5 * ((theta - opts.PriorMean) * (theta - opts.PriorMean) / opts.PriorVariance)
		for _, row := range run.InputRows {
			logp += -0.5 * ((row.EffectSize - theta) * (row.EffectSize - theta) / row.Variance)
		}
		logWeights[i] = logp
		if logp > maxLogWeight {
			maxLogWeight = logp
		}
	}
	if math.IsNaN(maxLogWeight) || math.IsInf(maxLogWeight, 0) {
		return BayesianReport{}, fmt.Errorf("bayesian grid produced non-finite log weights")
	}
	weights := make([]float64, points)
	total := 0.0
	for i, logWeight := range logWeights {
		w := math.Exp(logWeight - maxLogWeight)
		weights[i] = w
		total += w
	}
	if total <= 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		return BayesianReport{}, fmt.Errorf("bayesian grid produced invalid total weight")
	}
	mean := 0.0
	for i, w := range weights {
		mean += (start + float64(i)*step) * w / total
	}
	low := gridQuantile(start, step, weights, total, 0.025)
	high := gridQuantile(start, step, weights, total, 0.975)
	return BayesianReport{RunID: run.ID, Method: "grid-bayesian-engine", Engine: g.Name(), Studies: len(run.InputRows), PriorMean: opts.PriorMean, PriorVariance: opts.PriorVariance, PosteriorMean: mean, PosteriorVar: approx.PosteriorVar, CredibleLow95: low, CredibleHigh95: high}, nil
}

func gridQuantile(start, step float64, weights []float64, total, q float64) float64 {
	acc := 0.0
	for i, w := range weights {
		acc += w / total
		if acc >= q {
			return start + float64(i)*step
		}
	}
	return start + float64(len(weights)-1)*step
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
