package analysis

import "fmt"

type InfluenceReport struct {
	RunID            string         `json:"runId"`
	BaselineEstimate float64        `json:"baselineEstimate"`
	BaselineVariance float64        `json:"baselineVariance"`
	Rows             []InfluenceRow `json:"rows"`
}

type InfluenceRow struct {
	PaperID       string  `json:"paperId"`
	LeaveOneOut   float64 `json:"leaveOneOut"`
	Delta         float64 `json:"delta"`
	AbsoluteDelta float64 `json:"absoluteDelta"`
	Weight        float64 `json:"weight"`
}

func InfluenceDiagnostics(run AnalysisRun) (InfluenceReport, error) {
	if len(run.InputRows) < 2 {
		return InfluenceReport{}, fmt.Errorf("influence diagnostics require at least two input rows")
	}
	if err := validateAnalysisRows(run.InputRows); err != nil {
		return InfluenceReport{}, err
	}
	baselineEstimate, baselineVariance, _, err := pooledEstimateExcluding(run.InputRows, -1)
	if err != nil {
		return InfluenceReport{}, err
	}
	report := InfluenceReport{RunID: run.ID, BaselineEstimate: baselineEstimate, BaselineVariance: baselineVariance}
	for omit, row := range run.InputRows {
		loo, _, _, err := pooledEstimateExcluding(run.InputRows, omit)
		if err != nil {
			return InfluenceReport{}, err
		}
		delta := loo - baselineEstimate
		absDelta := delta
		if absDelta < 0 {
			absDelta = -absDelta
		}
		if row.Variance <= 0 {
			return InfluenceReport{}, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		report.Rows = append(report.Rows, InfluenceRow{PaperID: row.PaperID, LeaveOneOut: loo, Delta: delta, AbsoluteDelta: absDelta, Weight: 1 / row.Variance})
	}
	return report, nil
}
