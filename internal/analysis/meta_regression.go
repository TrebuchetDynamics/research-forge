package analysis

import (
	"fmt"
	"math"
)

// MetaRegressionReport records a simple weighted meta-regression diagnostic.
type MetaRegressionReport struct {
	RunID     string  `json:"runId"`
	Moderator string  `json:"moderator"`
	Studies   int     `json:"studies"`
	Intercept float64 `json:"intercept"`
	Slope     float64 `json:"slope"`
}

// MetaRegression fits a weighted linear regression of effect size on a numeric moderator.
func MetaRegression(run AnalysisRun, moderator string, values map[string]float64) (MetaRegressionReport, error) {
	if len(run.InputRows) < 2 {
		return MetaRegressionReport{}, fmt.Errorf("meta-regression requires at least two input rows")
	}
	if moderator == "" {
		return MetaRegressionReport{}, fmt.Errorf("moderator is required")
	}
	if err := validateAnalysisRows(run.InputRows); err != nil {
		return MetaRegressionReport{}, err
	}
	xs := []float64{}
	ys := []float64{}
	weights := []float64{}
	for _, row := range run.InputRows {
		x, ok := values[row.PaperID]
		if !ok {
			return MetaRegressionReport{}, fmt.Errorf("missing moderator value for paper %s", row.PaperID)
		}
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return MetaRegressionReport{}, fmt.Errorf("moderator value must be finite for paper %s", row.PaperID)
		}
		if row.Variance <= 0 {
			return MetaRegressionReport{}, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		xs = append(xs, x)
		ys = append(ys, row.EffectSize)
		weights = append(weights, 1/row.Variance)
	}
	intercept, slope, err := weightedLeastSquares(xs, ys, weights)
	if err != nil {
		return MetaRegressionReport{}, err
	}
	return MetaRegressionReport{RunID: run.ID, Moderator: moderator, Studies: len(run.InputRows), Intercept: intercept, Slope: slope}, nil
}

func weightedLeastSquares(xs, ys, weights []float64) (float64, float64, error) {
	sw, swx, swy, swxx, swxy := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		w := weights[i]
		sw += w
		swx += w * xs[i]
		swy += w * ys[i]
		swxx += w * xs[i] * xs[i]
		swxy += w * xs[i] * ys[i]
	}
	denom := sw*swxx - swx*swx
	if denom <= 0 || sw <= 0 {
		return 0, 0, fmt.Errorf("meta-regression moderator must vary across studies")
	}
	slope := (sw*swxy - swx*swy) / denom
	intercept := (swy - slope*swx) / sw
	return intercept, slope, nil
}
