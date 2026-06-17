package analysis

import (
	"fmt"
	"math"
)

// PublicationBiasReport records a lightweight publication-bias diagnostic.
type PublicationBiasReport struct {
	RunID      string  `json:"runId"`
	Method     string  `json:"method"`
	Studies    int     `json:"studies"`
	Intercept  float64 `json:"intercept,omitempty"`
	Slope      float64 `json:"slope,omitempty"`
	KendallTau float64 `json:"kendallTau,omitempty"`
	Warning    string  `json:"warning,omitempty"`
}

// EggerRegression computes an Egger-style regression of standardized effects on precision.
func EggerRegression(run AnalysisRun) (PublicationBiasReport, error) {
	if len(run.InputRows) < 3 {
		return PublicationBiasReport{}, fmt.Errorf("egger publication-bias test requires at least three input rows")
	}
	xs := make([]float64, 0, len(run.InputRows))
	ys := make([]float64, 0, len(run.InputRows))
	for _, row := range run.InputRows {
		if row.Variance <= 0 {
			return PublicationBiasReport{}, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		se := math.Sqrt(row.Variance)
		xs = append(xs, 1/se)
		ys = append(ys, row.EffectSize/se)
	}
	intercept, slope := ordinaryLeastSquares(xs, ys)
	report := PublicationBiasReport{RunID: run.ID, Method: "egger", Studies: len(run.InputRows), Intercept: intercept, Slope: slope}
	if len(run.InputRows) < 10 {
		report.Warning = "egger test is underpowered with fewer than 10 studies"
	}
	return report, nil
}

func BeggRankCorrelation(run AnalysisRun) (PublicationBiasReport, error) {
	if len(run.InputRows) < 3 {
		return PublicationBiasReport{}, fmt.Errorf("begg publication-bias test requires at least three input rows")
	}
	effects := make([]float64, 0, len(run.InputRows))
	variances := make([]float64, 0, len(run.InputRows))
	for _, row := range run.InputRows {
		if row.Variance <= 0 {
			return PublicationBiasReport{}, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		effects = append(effects, row.EffectSize)
		variances = append(variances, row.Variance)
	}
	report := PublicationBiasReport{RunID: run.ID, Method: "begg-rank-correlation", Studies: len(run.InputRows), KendallTau: kendallTau(effects, variances)}
	if len(run.InputRows) < 10 {
		report.Warning = "begg rank-correlation test is underpowered with fewer than 10 studies"
	}
	return report, nil
}

func ordinaryLeastSquares(xs, ys []float64) (float64, float64) {
	n := float64(len(xs))
	sumX, sumY, sumXX, sumXY := 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXX += xs[i] * xs[i]
		sumXY += xs[i] * ys[i]
	}
	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		return 0, 0
	}
	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n
	return intercept, slope
}
