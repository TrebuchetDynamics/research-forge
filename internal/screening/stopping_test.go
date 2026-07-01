package screening

import "testing"

func TestStoppingCriteriaRecommendsStopWhenRecallTargetReachedAndPoolFullyScreened(t *testing.T) {
	events := []DecisionEvent{
		{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude},
		{PaperID: "p2", Stage: StageTitleAbstract, Decision: DecisionInclude},
	}

	rec := StoppingCriteria(events, StageTitleAbstract, 0.95, 2)

	if !rec.CanStop || rec.CurrentRecall != 1 || rec.Screened != 2 || rec.Included != 1 {
		t.Fatalf("recommendation = %#v", rec)
	}
}

func TestStoppingCriteriaDoesNotStopWithoutIncludedSeeds(t *testing.T) {
	events := []DecisionEvent{{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude}}

	rec := StoppingCriteria(events, StageTitleAbstract, 0.95, 1)

	if rec.CanStop || rec.Reason != "no included seed records observed" {
		t.Fatalf("recommendation = %#v", rec)
	}
}

// TestStoppingCriteriaDoesNotStopWithUnscreenedBacklog guards against the
// degenerate case where "recall over observed decisions" is tautologically
// 1.0 the moment a single Include exists, regardless of targetRecall. A
// large unscreened backlog must block CanStop even though CurrentRecall
// reports 100% among the records reviewed so far.
func TestStoppingCriteriaDoesNotStopWithUnscreenedBacklog(t *testing.T) {
	events := []DecisionEvent{
		{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude},
		{PaperID: "p2", Stage: StageTitleAbstract, Decision: DecisionInclude},
	}

	rec := StoppingCriteria(events, StageTitleAbstract, 0.95, 100)

	if rec.CanStop {
		t.Fatalf("recommendation = %#v, want CanStop=false with 98 unscreened records remaining", rec)
	}
	if rec.CurrentRecall != 1 {
		t.Fatalf("CurrentRecall = %v, want 1 (recall is still only measured over observed decisions)", rec.CurrentRecall)
	}
	if rec.Unscreened != 98 {
		t.Fatalf("Unscreened = %d, want 98", rec.Unscreened)
	}
}
