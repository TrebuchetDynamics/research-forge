package screening

import (
	"sort"
	"strings"
)

type ReviewerAssignment struct {
	PaperID  string `json:"paperId"`
	Reviewer string `json:"reviewer"`
	Stage    Stage  `json:"stage"`
}

type ConflictAdjudicationPanel struct {
	Stage       Stage             `json:"stage"`
	Conflicts   []ConflictItem    `json:"conflicts"`
	Adjudicated []AdjudicatedItem `json:"adjudicated"`
}

type ConflictItem struct {
	PaperID   string          `json:"paperId"`
	Decisions []DecisionEvent `json:"decisions"`
}

type AdjudicatedItem struct {
	PaperID  string        `json:"paperId"`
	Decision DecisionEvent `json:"decision"`
}

type UncertainQueueItem struct {
	PaperID   string          `json:"paperId"`
	Decisions []DecisionEvent `json:"decisions"`
}

type ScreeningAuditBundleInput struct {
	Records     []ScreeningRecord
	Events      []DecisionEvent
	Assignments []ReviewerAssignment
	Stage       Stage
	ActiveRun   ActiveLearningRun
}

type ScreeningAuditBundle struct {
	SchemaVersion       string                    `json:"schemaVersion"`
	Stage               Stage                     `json:"stage"`
	InputHash           string                    `json:"inputHash"`
	DecisionHash        string                    `json:"decisionHash"`
	Assignments         []ReviewerAssignment      `json:"assignments"`
	Panel               ConflictAdjudicationPanel `json:"conflictAdjudicationPanel"`
	Uncertain           []UncertainQueueItem      `json:"uncertainQueue"`
	Progress            ProgressReport            `json:"progress"`
	ActiveRunRef        string                    `json:"activeRunRef,omitempty"`
	FrozenDataset       []ScreeningRecord         `json:"frozenDataset,omitempty"`
	SeedLabels          []DecisionEvent           `json:"seedLabels,omitempty"`
	RankingIterations   []PrioritizedRecord       `json:"rankingIterations,omitempty"`
	ReviewerActions     []DecisionEvent           `json:"reviewerActions,omitempty"`
	StoppingDiagnostics StoppingRecommendation    `json:"stoppingDiagnostics"`
	RandomSeeds         []string                  `json:"randomSeeds,omitempty"`
	ModelMetadata       ModelMetadata             `json:"modelMetadata"`
}

type ModelMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Method  string `json:"method"`
}

func AssignReviewers(records []ScreeningRecord, reviewers []string, reviewersPerRecord int) []ReviewerAssignment {
	cleanReviewers := []string{}
	for _, reviewer := range reviewers {
		if trimmed := strings.TrimSpace(reviewer); trimmed != "" {
			cleanReviewers = append(cleanReviewers, trimmed)
		}
	}
	if reviewersPerRecord <= 0 {
		reviewersPerRecord = 1
	}
	assignments := []ReviewerAssignment{}
	if len(cleanReviewers) == 0 {
		return assignments
	}
	for i, record := range records {
		if strings.TrimSpace(record.ID) == "" {
			continue
		}
		for offset := 0; offset < reviewersPerRecord && offset < len(cleanReviewers); offset++ {
			reviewer := cleanReviewers[(i+offset)%len(cleanReviewers)]
			assignments = append(assignments, ReviewerAssignment{PaperID: record.ID, Reviewer: reviewer, Stage: StageTitleAbstract})
		}
	}
	return assignments
}

func BuildConflictAdjudicationPanel(events []DecisionEvent, stage Stage) ConflictAdjudicationPanel {
	byPaper := map[string][]DecisionEvent{}
	adjudicated := map[string]DecisionEvent{}
	for _, event := range events {
		if event.Stage != stage {
			continue
		}
		byPaper[event.PaperID] = append(byPaper[event.PaperID], event)
		if event.Adjudicated {
			adjudicated[event.PaperID] = event
		}
	}
	panel := ConflictAdjudicationPanel{Stage: stage}
	for paperID, paperEvents := range byPaper {
		if decision, ok := adjudicated[paperID]; ok {
			panel.Adjudicated = append(panel.Adjudicated, AdjudicatedItem{PaperID: paperID, Decision: decision})
			continue
		}
		decisions := map[Decision]bool{}
		for _, event := range paperEvents {
			decisions[event.Decision] = true
		}
		if decisions[DecisionInclude] && decisions[DecisionExclude] {
			panel.Conflicts = append(panel.Conflicts, ConflictItem{PaperID: paperID, Decisions: paperEvents})
		}
	}
	sort.Slice(panel.Conflicts, func(i, j int) bool { return panel.Conflicts[i].PaperID < panel.Conflicts[j].PaperID })
	sort.Slice(panel.Adjudicated, func(i, j int) bool { return panel.Adjudicated[i].PaperID < panel.Adjudicated[j].PaperID })
	return panel
}

func UncertainQueue(events []DecisionEvent, stage Stage) []UncertainQueueItem {
	byPaper := map[string][]DecisionEvent{}
	adjudicated := map[string]bool{}
	for _, event := range events {
		if event.Stage != stage {
			continue
		}
		if event.Adjudicated {
			adjudicated[event.PaperID] = true
		}
		if event.Decision == DecisionUncertain {
			byPaper[event.PaperID] = append(byPaper[event.PaperID], event)
		}
	}
	items := []UncertainQueueItem{}
	for paperID, events := range byPaper {
		if !adjudicated[paperID] {
			items = append(items, UncertainQueueItem{PaperID: paperID, Decisions: events})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].PaperID < items[j].PaperID })
	return items
}

func BuildScreeningAuditBundle(input ScreeningAuditBundleInput) ScreeningAuditBundle {
	stage := input.Stage
	if stage == "" {
		stage = StageTitleAbstract
	}
	active := input.ActiveRun
	seedLabels := append([]DecisionEvent{}, active.SeedDecisions...)
	if len(seedLabels) == 0 {
		seedLabels = seedDecisions(input.Events, stage)
	}
	ranking := append([]PrioritizedRecord{}, active.RankedOutput...)
	stopping := active.StoppingDiagnostics
	if stopping.Stage == "" {
		stopping = StoppingCriteria(input.Events, stage, 0.95)
	}
	method := strings.TrimSpace(active.RankingMethod)
	if method == "" {
		method = "active-learning"
	}
	return ScreeningAuditBundle{SchemaVersion: "1", Stage: stage, InputHash: hashJSON(input.Records), DecisionHash: hashJSON(input.Events), Assignments: append([]ReviewerAssignment{}, input.Assignments...), Panel: BuildConflictAdjudicationPanel(input.Events, stage), Uncertain: UncertainQueue(input.Events, stage), Progress: Progress(input.Events, stage, len(input.Records)), ActiveRunRef: active.RunID, FrozenDataset: append([]ScreeningRecord{}, input.Records...), SeedLabels: seedLabels, RankingIterations: ranking, ReviewerActions: append([]DecisionEvent{}, input.Events...), StoppingDiagnostics: stopping, RandomSeeds: []string{hashJSON(input.Records)[:12], hashJSON(input.Events)[:12]}, ModelMetadata: ModelMetadata{Name: "ResearchForge ASReview-style ranker", Version: "1", Method: method}}
}
