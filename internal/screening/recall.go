package screening

// RecallEffortPoint records cumulative recall over screening effort.
type RecallEffortPoint struct {
	Screened int     `json:"screened"`
	Included int     `json:"included"`
	Recall   float64 `json:"recall"`
}

// RecallEffortCurve computes a deterministic cumulative recall curve for one stage.
func RecallEffortCurve(events []DecisionEvent, stage Stage) []RecallEffortPoint {
	totalIncluded := 0
	for _, event := range events {
		if event.Stage == stage && event.Decision == DecisionInclude {
			totalIncluded++
		}
	}
	points := []RecallEffortPoint{}
	included := 0
	screened := 0
	for _, event := range events {
		if event.Stage != stage {
			continue
		}
		screened++
		if event.Decision == DecisionInclude {
			included++
		}
		recall := 0.0
		if totalIncluded > 0 {
			recall = float64(included) / float64(totalIncluded)
		}
		points = append(points, RecallEffortPoint{Screened: screened, Included: included, Recall: recall})
	}
	return points
}
