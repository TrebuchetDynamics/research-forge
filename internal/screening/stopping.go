package screening

import "fmt"

// StoppingRecommendation summarizes whether observed screening can stop at a recall target.
type StoppingRecommendation struct {
	Stage         Stage   `json:"stage"`
	TargetRecall  float64 `json:"targetRecall"`
	CurrentRecall float64 `json:"currentRecall"`
	Screened      int     `json:"screened"`
	Included      int     `json:"included"`
	Unscreened    int     `json:"unscreened"`
	CanStop       bool    `json:"canStop"`
	Reason        string  `json:"reason"`
}

// StoppingCriteria evaluates a simple auditable recall threshold over observed decisions.
//
// CurrentRecall is only ever measured over records already decided, so it
// reaches 1.0 as soon as a single Include exists — it is not an estimate of
// recall against the full candidate pool. totalRecords guards against
// recommending a stop while records outside "events" remain unscreened;
// richer recall estimation against an unscreened backlog is a deferred
// roadmap item (see opensource/inventory/asreview.md), not implemented here.
func StoppingCriteria(events []DecisionEvent, stage Stage, targetRecall float64, totalRecords int) StoppingRecommendation {
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
	if totalRecords > rec.Screened {
		rec.Unscreened = totalRecords - rec.Screened
	}
	rec.CurrentRecall = last.Recall
	if last.Included == 0 {
		rec.Reason = "no included seed records observed"
		return rec
	}
	if rec.Unscreened > 0 {
		rec.Reason = fmt.Sprintf("%d record(s) still unscreened; recall is only measured over observed decisions", rec.Unscreened)
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
