package analysis

import "fmt"

// SubgroupEstimate is one inverse-variance pooled subgroup estimate.
type SubgroupEstimate struct {
	Group    string  `json:"group"`
	Studies  int     `json:"studies"`
	Estimate float64 `json:"estimate"`
	Variance float64 `json:"variance"`
}

// SubgroupReport records pooled estimates split by a categorical moderator.
type SubgroupReport struct {
	RunID    string             `json:"runId"`
	Variable string             `json:"variable"`
	Groups   []SubgroupEstimate `json:"groups"`
}

// SubgroupAnalysis computes fixed-effect inverse-variance pooled estimates per group.
func SubgroupAnalysis(run AnalysisRun, variable string, groups map[string]string) (SubgroupReport, error) {
	if len(run.InputRows) == 0 {
		return SubgroupReport{}, fmt.Errorf("subgroup analysis requires input rows")
	}
	if variable == "" {
		return SubgroupReport{}, fmt.Errorf("subgroup variable is required")
	}
	byGroup := map[string][]InputRow{}
	order := []string{}
	seen := map[string]bool{}
	for _, row := range run.InputRows {
		group := groups[row.PaperID]
		if group == "" {
			return SubgroupReport{}, fmt.Errorf("missing subgroup value for paper %s", row.PaperID)
		}
		if !seen[group] {
			seen[group] = true
			order = append(order, group)
		}
		byGroup[group] = append(byGroup[group], row)
	}
	report := SubgroupReport{RunID: run.ID, Variable: variable}
	for _, group := range order {
		estimate, variance, studies, err := pooledEstimate(byGroup[group])
		if err != nil {
			return SubgroupReport{}, err
		}
		report.Groups = append(report.Groups, SubgroupEstimate{Group: group, Studies: studies, Estimate: estimate, Variance: variance})
	}
	return report, nil
}

func pooledEstimate(rows []InputRow) (float64, float64, int, error) {
	weighted := 0.0
	weights := 0.0
	for _, row := range rows {
		if row.Variance <= 0 {
			return 0, 0, 0, fmt.Errorf("variance must be positive for paper %s", row.PaperID)
		}
		weight := 1 / row.Variance
		weighted += row.EffectSize * weight
		weights += weight
	}
	if weights == 0 {
		return 0, 0, 0, fmt.Errorf("no estimable rows")
	}
	return weighted / weights, 1 / weights, len(rows), nil
}
