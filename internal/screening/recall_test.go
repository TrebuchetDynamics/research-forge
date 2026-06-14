package screening

import "testing"

func TestRecallEffortCurveTracksCumulativeRecall(t *testing.T) {
	events := []DecisionEvent{
		{PaperID: "p1", Stage: StageTitleAbstract, Decision: DecisionExclude},
		{PaperID: "p2", Stage: StageTitleAbstract, Decision: DecisionInclude},
		{PaperID: "p3", Stage: StageFullText, Decision: DecisionInclude},
		{PaperID: "p4", Stage: StageTitleAbstract, Decision: DecisionInclude},
	}

	curve := RecallEffortCurve(events, StageTitleAbstract)

	if len(curve) != 3 {
		t.Fatalf("curve length = %d", len(curve))
	}
	if curve[0].Screened != 1 || curve[0].Included != 0 || curve[0].Recall != 0 {
		t.Fatalf("point 0 = %#v", curve[0])
	}
	if curve[1].Screened != 2 || curve[1].Included != 1 || curve[1].Recall != 0.5 {
		t.Fatalf("point 1 = %#v", curve[1])
	}
	if curve[2].Screened != 3 || curve[2].Included != 2 || curve[2].Recall != 1.0 {
		t.Fatalf("point 2 = %#v", curve[2])
	}
}
