package screening

import "sort"

// ReviewerProgress summarizes one reviewer's work for a screening stage.
type ReviewerProgress struct {
	Reviewer  string `json:"reviewer"`
	Decisions int    `json:"decisions"`
	Included  int    `json:"included"`
	Excluded  int    `json:"excluded"`
	Uncertain int    `json:"uncertain"`
}

// ProgressReport summarizes auditable human screening progress for one stage.
type ProgressReport struct {
	Stage           Stage              `json:"stage"`
	TotalRecords    int                `json:"totalRecords"`
	ScreenedRecords int                `json:"screenedRecords"`
	Remaining       int                `json:"remaining"`
	Conflicts       int                `json:"conflicts"`
	Reviewers       []ReviewerProgress `json:"reviewers"`
}

// Progress computes deterministic reviewer progress metrics for a stage.
func Progress(events []DecisionEvent, stage Stage, totalRecords int) ProgressReport {
	seenPapers := map[string]bool{}
	decisionsByPaper := map[string]map[Decision]bool{}
	adjudicated := map[string]bool{}
	byReviewer := map[string]*ReviewerProgress{}
	for _, event := range events {
		if event.Stage != stage {
			continue
		}
		seenPapers[event.PaperID] = true
		if event.Adjudicated {
			adjudicated[event.PaperID] = true
		}
		if decisionsByPaper[event.PaperID] == nil {
			decisionsByPaper[event.PaperID] = map[Decision]bool{}
		}
		decisionsByPaper[event.PaperID][event.Decision] = true
		reviewer := event.Reviewer
		if reviewer == "" {
			reviewer = "unknown"
		}
		progress := byReviewer[reviewer]
		if progress == nil {
			progress = &ReviewerProgress{Reviewer: reviewer}
			byReviewer[reviewer] = progress
		}
		progress.Decisions++
		switch event.Decision {
		case DecisionInclude:
			progress.Included++
		case DecisionExclude:
			progress.Excluded++
		case DecisionUncertain:
			progress.Uncertain++
		}
	}
	reviewers := make([]ReviewerProgress, 0, len(byReviewer))
	for _, progress := range byReviewer {
		reviewers = append(reviewers, *progress)
	}
	sort.Slice(reviewers, func(i, j int) bool { return reviewers[i].Reviewer < reviewers[j].Reviewer })
	for paper := range adjudicated {
		delete(decisionsByPaper, paper)
	}
	conflicts := 0
	for _, decisions := range decisionsByPaper {
		if decisions[DecisionInclude] && decisions[DecisionExclude] {
			conflicts++
		}
	}
	screened := len(seenPapers)
	remaining := 0
	if totalRecords > screened {
		remaining = totalRecords - screened
	}
	return ProgressReport{Stage: stage, TotalRecords: totalRecords, ScreenedRecords: screened, Remaining: remaining, Conflicts: conflicts, Reviewers: reviewers}
}
