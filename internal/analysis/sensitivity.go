package analysis

import "fmt"

// LeaveOneOutRow is one leave-one-study-out sensitivity result.
type LeaveOneOutRow struct {
	OmittedPaperID string  `json:"omittedPaperId"`
	Estimate       float64 `json:"estimate"`
	Variance       float64 `json:"variance"`
	Studies        int     `json:"studies"`
}

// SensitivityReport records deterministic sensitivity-analysis artifacts.
type SensitivityReport struct {
	RunID  string           `json:"runId"`
	Method string           `json:"method"`
	Rows   []LeaveOneOutRow `json:"rows"`
}

// LeaveOneOut computes inverse-variance fixed-effect estimates after omitting each study.
func LeaveOneOut(run AnalysisRun) (SensitivityReport, error) {
	if len(run.InputRows) < 2 {
		return SensitivityReport{}, fmt.Errorf("leave-one-out sensitivity requires at least two input rows")
	}
	report := SensitivityReport{RunID: run.ID, Method: "leave-one-out"}
	for omit := range run.InputRows {
		estimate, variance, studies, err := pooledEstimateExcluding(run.InputRows, omit)
		if err != nil {
			return SensitivityReport{}, err
		}
		report.Rows = append(report.Rows, LeaveOneOutRow{OmittedPaperID: run.InputRows[omit].PaperID, Estimate: estimate, Variance: variance, Studies: studies})
	}
	return report, nil
}

func pooledEstimateExcluding(rows []InputRow, omit int) (float64, float64, int, error) {
	weighted := 0.0
	weights := 0.0
	studies := 0
	for i, row := range rows {
		if i == omit {
			continue
		}
		if row.Variance <= 0 {
			return 0, 0, 0, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		weight := 1 / row.Variance
		weighted += row.EffectSize * weight
		weights += weight
		studies++
	}
	if weights == 0 {
		return 0, 0, 0, fmt.Errorf("no estimable rows")
	}
	return weighted / weights, 1 / weights, studies, nil
}
