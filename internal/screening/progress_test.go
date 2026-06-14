package screening

import "testing"

func TestProgressReportsReviewerMetricsAndConflicts(t *testing.T) {
	events := []DecisionEvent{
		{PaperID: "paper-1", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "ada"},
		{PaperID: "paper-2", Stage: StageTitleAbstract, Decision: DecisionExclude, Reviewer: "ada"},
		{PaperID: "paper-2", Stage: StageTitleAbstract, Decision: DecisionInclude, Reviewer: "grace"},
		{PaperID: "paper-3", Stage: StageTitleAbstract, Decision: DecisionUncertain, Reviewer: "grace"},
		{PaperID: "paper-4", Stage: StageFullText, Decision: DecisionInclude, Reviewer: "ada"},
	}

	report := Progress(events, StageTitleAbstract, 5)

	if report.Stage != StageTitleAbstract || report.TotalRecords != 5 || report.ScreenedRecords != 3 || report.Remaining != 2 || report.Conflicts != 1 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Reviewers) != 2 || report.Reviewers[0].Reviewer != "ada" || report.Reviewers[1].Reviewer != "grace" {
		t.Fatalf("reviewers = %#v", report.Reviewers)
	}
	if report.Reviewers[0].Decisions != 2 || report.Reviewers[0].Included != 1 || report.Reviewers[0].Excluded != 1 {
		t.Fatalf("ada progress = %#v", report.Reviewers[0])
	}
	if report.Reviewers[1].Decisions != 2 || report.Reviewers[1].Included != 1 || report.Reviewers[1].Uncertain != 1 {
		t.Fatalf("grace progress = %#v", report.Reviewers[1])
	}
}
