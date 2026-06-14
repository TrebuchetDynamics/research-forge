package screening

// StoppingRecommendation summarizes whether observed screening can stop at a recall target.
type StoppingRecommendation struct {
	Stage         Stage   `json:"stage"`
	TargetRecall  float64 `json:"targetRecall"`
	CurrentRecall float64 `json:"currentRecall"`
	Screened      int     `json:"screened"`
	Included      int     `json:"included"`
	CanStop       bool    `json:"canStop"`
	Reason        string  `json:"reason"`
}

// StoppingCriteria evaluates a simple auditable recall threshold over observed decisions.
func StoppingCriteria(events []DecisionEvent, stage Stage, targetRecall float64) StoppingRecommendation {
	if targetRecall <= 0 || targetRecall > 1 {
		targetRecall = 0.95
	}
	curve := RecallEffortCurve(events, stage)
	rec := StoppingRecommendation{Stage: stage, TargetRecall: targetRecall, Reason: "no screening decisions for stage"}
	if len(curve) == 0 {
		return rec
	}
	last := curve[len(curve)-1]
	rec.Screened = last.Screened
	rec.Included = last.Included
	rec.CurrentRecall = last.Recall
	if last.Included == 0 {
		rec.Reason = "no included seed records observed"
		return rec
	}
	if rec.CurrentRecall >= targetRecall {
		rec.CanStop = true
		rec.Reason = "target recall reached by observed decisions"
		return rec
	}
	rec.Reason = "target recall not reached"
	return rec
}
