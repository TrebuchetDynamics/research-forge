package screening

import (
	"fmt"
	"sort"
	"strings"
)

type BalancedPolicy struct {
	Name               string  `json:"name"`
	ExploitationWeight float64 `json:"exploitationWeight"`
	ExplorationWeight  float64 `json:"explorationWeight"`
}

type RecallEffortSimulation struct {
	TotalRelevant int                  `json:"totalRelevant"`
	Points        []RecallEffortPoint  `json:"points"`
	Targets       []RecallTargetResult `json:"targets"`
}

type RecallTargetResult struct {
	TargetRecall float64 `json:"targetRecall"`
	ReachedAt    int     `json:"reachedAt"`
}

type ActiveLearningSensitivityInput struct {
	Records          []ScreeningRecord
	Events           []DecisionEvent
	Stage            Stage
	RelevantPaperIDs []string
	Policies         []BalancedPolicy
	TargetRecalls    []float64
}

type ActiveLearningSensitivityReport struct {
	SchemaVersion  string                       `json:"schemaVersion"`
	Stage          Stage                        `json:"stage"`
	InputHash      string                       `json:"inputHash"`
	DecisionHash   string                       `json:"decisionHash"`
	SelectedPolicy string                       `json:"selectedPolicy"`
	PolicyResults  []ActiveLearningPolicyResult `json:"policyResults"`
}

type ActiveLearningPolicyResult struct {
	Policy      BalancedPolicy         `json:"policy"`
	RankedCount int                    `json:"rankedCount"`
	Simulation  RecallEffortSimulation `json:"simulation"`
	Score       float64                `json:"score"`
}

func PrioritizeBalancedRecords(records []ScreeningRecord, events []DecisionEvent, stage Stage, policy BalancedPolicy) []PrioritizedRecord {
	policy = normalizeBalancedPolicy(policy)
	exploitation := PrioritizeModelRecords(records, events, stage)
	exploration := PrioritizeUncertaintyRecords(records, events, stage)
	exploitScores := map[string]float64{}
	exploreScores := map[string]float64{}
	for i, record := range exploitation {
		exploitScores[record.ID] = reciprocalRankScore(i)
	}
	for i, record := range exploration {
		exploreScores[record.ID] = reciprocalRankScore(i)
	}
	ids := map[string]bool{}
	for _, record := range exploitation {
		ids[record.ID] = true
	}
	for _, record := range exploration {
		ids[record.ID] = true
	}
	out := []PrioritizedRecord{}
	for id := range ids {
		exploit := exploitScores[id]
		explore := exploreScores[id]
		score := policy.ExploitationWeight*exploit + policy.ExplorationWeight*explore
		out = append(out, PrioritizedRecord{ID: id, Score: score, ExploitationScore: exploit, ExplorationScore: explore, Policy: "balanced"})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func SimulateRecallEffort(ranked []PrioritizedRecord, relevantIDs []string, targets []float64) RecallEffortSimulation {
	relevant := map[string]bool{}
	for _, id := range relevantIDs {
		if strings.TrimSpace(id) != "" {
			relevant[id] = true
		}
	}
	if len(targets) == 0 {
		targets = []float64{0.8, 0.9, 0.95}
	}
	sim := RecallEffortSimulation{TotalRelevant: len(relevant)}
	found := 0
	seenRelevant := map[string]bool{}
	for i, record := range ranked {
		if relevant[record.ID] && !seenRelevant[record.ID] {
			found++
			seenRelevant[record.ID] = true
		}
		recall := 0.0
		if len(relevant) > 0 {
			recall = float64(found) / float64(len(relevant))
		}
		sim.Points = append(sim.Points, RecallEffortPoint{Screened: i + 1, Included: found, Recall: recall})
	}
	for _, target := range targets {
		if target <= 0 || target > 1 {
			continue
		}
		result := RecallTargetResult{TargetRecall: target}
		for _, point := range sim.Points {
			if point.Recall >= target {
				result.ReachedAt = point.Screened
				break
			}
		}
		sim.Targets = append(sim.Targets, result)
	}
	return sim
}

func ActiveLearningSensitivityDiagnostics(input ActiveLearningSensitivityInput) (ActiveLearningSensitivityReport, error) {
	if len(input.Records) == 0 {
		return ActiveLearningSensitivityReport{}, fmt.Errorf("sensitivity diagnostics require screening records")
	}
	stage := input.Stage
	if stage == "" {
		stage = StageTitleAbstract
	}
	policies := input.Policies
	if len(policies) == 0 {
		policies = DefaultBalancedPolicies()
	}
	report := ActiveLearningSensitivityReport{SchemaVersion: "1", Stage: stage, InputHash: hashJSON(input.Records), DecisionHash: hashJSON(input.Events)}
	for _, policy := range policies {
		policy = normalizeBalancedPolicy(policy)
		ranked := PrioritizeBalancedRecords(input.Records, input.Events, stage, policy)
		sim := SimulateRecallEffort(ranked, input.RelevantPaperIDs, input.TargetRecalls)
		result := ActiveLearningPolicyResult{Policy: policy, RankedCount: len(ranked), Simulation: sim, Score: sensitivityScore(sim)}
		report.PolicyResults = append(report.PolicyResults, result)
	}
	sort.SliceStable(report.PolicyResults, func(i, j int) bool {
		if report.PolicyResults[i].Score != report.PolicyResults[j].Score {
			return report.PolicyResults[i].Score > report.PolicyResults[j].Score
		}
		return report.PolicyResults[i].Policy.Name < report.PolicyResults[j].Policy.Name
	})
	if len(report.PolicyResults) > 0 {
		report.SelectedPolicy = report.PolicyResults[0].Policy.Name
	}
	return report, nil
}

func DefaultBalancedPolicies() []BalancedPolicy {
	return []BalancedPolicy{{Name: "balanced", ExploitationWeight: 0.7, ExplorationWeight: 0.3}, {Name: "exploit-heavy", ExploitationWeight: 0.9, ExplorationWeight: 0.1}, {Name: "explore-heavy", ExploitationWeight: 0.5, ExplorationWeight: 0.5}}
}

func normalizeBalancedPolicy(policy BalancedPolicy) BalancedPolicy {
	if strings.TrimSpace(policy.Name) == "" {
		policy.Name = fmt.Sprintf("exploit-%.2f-explore-%.2f", policy.ExploitationWeight, policy.ExplorationWeight)
	}
	if policy.ExploitationWeight == 0 && policy.ExplorationWeight == 0 {
		policy.ExploitationWeight = 0.7
		policy.ExplorationWeight = 0.3
	}
	return policy
}

func reciprocalRankScore(rank int) float64 { return 1.0 / float64(rank+1) }

func sensitivityScore(sim RecallEffortSimulation) float64 {
	if len(sim.Points) == 0 {
		return 0
	}
	last := sim.Points[len(sim.Points)-1].Recall
	penalty := 0.0
	for _, target := range sim.Targets {
		if target.ReachedAt > 0 {
			penalty += float64(target.ReachedAt) * 0.001
		} else {
			penalty += 1
		}
	}
	return last - penalty
}
