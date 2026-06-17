package screening

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ActiveLearningRunInput struct {
	RunID         string
	Records       []ScreeningRecord
	Events        []DecisionEvent
	Stage         Stage
	RankingMethod string
	TargetRecall  float64
}

type ActiveLearningRun struct {
	SchemaVersion       string                 `json:"schemaVersion"`
	RunID               string                 `json:"runId"`
	Stage               Stage                  `json:"stage"`
	InputHash           string                 `json:"inputHash"`
	DecisionHash        string                 `json:"decisionHash"`
	RankingMethod       string                 `json:"rankingMethod"`
	SeedDecisions       []DecisionEvent        `json:"seedDecisions"`
	RankedOutput        []PrioritizedRecord    `json:"rankedOutput"`
	ReviewerProgress    ProgressReport         `json:"reviewerProgress"`
	StoppingDiagnostics StoppingRecommendation `json:"stoppingDiagnostics"`
	AdjudicationState   AdjudicationState      `json:"adjudicationState"`
}

type AdjudicationState struct {
	Conflicts        int      `json:"conflicts"`
	ConflictPaperIDs []string `json:"conflictPaperIds"`
	Adjudicated      int      `json:"adjudicated"`
	Pending          int      `json:"pending"`
}

func BuildActiveLearningRun(input ActiveLearningRunInput) (ActiveLearningRun, error) {
	if len(input.Records) == 0 {
		return ActiveLearningRun{}, fmt.Errorf("active-learning input records are required")
	}
	stage := input.Stage
	if stage == "" {
		stage = StageTitleAbstract
	}
	method := strings.TrimSpace(input.RankingMethod)
	if method == "" {
		method = "active-learning"
	}
	var ranked []PrioritizedRecord
	switch method {
	case "active-learning", "asreview":
		method = "active-learning"
		ranked = PrioritizeActiveLearningRecords(input.Records, input.Events, stage)
	case "model", "naive-bayes":
		method = "model"
		ranked = PrioritizeModelRecords(input.Records, input.Events, stage)
	case "uncertainty":
		ranked = PrioritizeUncertaintyRecords(input.Records, input.Events, stage)
	default:
		return ActiveLearningRun{}, fmt.Errorf("unknown active-learning ranking method %q", method)
	}
	runID := strings.TrimSpace(input.RunID)
	if runID == "" {
		runID = fmt.Sprintf("%s-%s", stage, method)
	}
	return ActiveLearningRun{SchemaVersion: "1", RunID: runID, Stage: stage, InputHash: hashJSON(input.Records), DecisionHash: hashJSON(input.Events), RankingMethod: method, SeedDecisions: seedDecisions(input.Events, stage), RankedOutput: ranked, ReviewerProgress: Progress(input.Events, stage, len(input.Records)), StoppingDiagnostics: StoppingCriteria(input.Events, stage, input.TargetRecall), AdjudicationState: buildAdjudicationState(input.Events, stage)}, nil
}

func seedDecisions(events []DecisionEvent, stage Stage) []DecisionEvent {
	out := []DecisionEvent{}
	for _, event := range events {
		if event.Stage == stage && (event.Decision == DecisionInclude || event.Decision == DecisionExclude) {
			out = append(out, event)
		}
	}
	return out
}

func buildAdjudicationState(events []DecisionEvent, stage Stage) AdjudicationState {
	decisionsByPaper := map[string]map[Decision]bool{}
	adjudicated := map[string]bool{}
	for _, event := range events {
		if event.Stage != stage {
			continue
		}
		if decisionsByPaper[event.PaperID] == nil {
			decisionsByPaper[event.PaperID] = map[Decision]bool{}
		}
		decisionsByPaper[event.PaperID][event.Decision] = true
		if event.Adjudicated {
			adjudicated[event.PaperID] = true
		}
	}
	conflicts := []string{}
	for paperID, decisions := range decisionsByPaper {
		if adjudicated[paperID] {
			continue
		}
		if decisions[DecisionInclude] && decisions[DecisionExclude] {
			conflicts = append(conflicts, paperID)
		}
	}
	sort.Strings(conflicts)
	state := AdjudicationState{Conflicts: len(conflicts), ConflictPaperIDs: conflicts, Adjudicated: len(adjudicated)}
	state.Pending = state.Conflicts
	return state
}

func hashJSON(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:])
}
