package screening

import "testing"

func TestStoppingCriteriaRecommendsStopWhenRecallTargetReached(t *testing.T) {
	events := []DecisionEvent{
		{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude},
		{PaperID: "p2", Stage: StageTitleAbstract, Decision: DecisionInclude},
	}

	rec := StoppingCriteria(events, StageTitleAbstract, 0.95)

	if !rec.CanStop || rec.CurrentRecall != 1 || rec.Screened != 2 || rec.Included != 1 {
		t.Fatalf("recommendation = %#v", rec)
	}
}

func TestStoppingCriteriaDoesNotStopWithoutIncludedSeeds(t *testing.T) {
	events := []DecisionEvent{{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude}}

	rec := StoppingCriteria(events, StageTitleAbstract, 0.95)

	if rec.CanStop || rec.Reason != "no included seed records observed" {
		t.Fatalf("recommendation = %#v", rec)
	}
}
