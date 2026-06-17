package sources

import "fmt"

type GraphExpansionBudget struct {
	MaxDepth     int    `json:"maxDepth"`
	MaxNodes     int    `json:"maxNodes"`
	MaxAPICalls  int    `json:"maxApiCalls"`
	RetryBudget  int    `json:"retryBudget"`
	ResumeCursor string `json:"resumeCursor,omitempty"`
}

type GraphExpansionBudgetEstimate struct {
	SchemaVersion     string                        `json:"schemaVersion"`
	Source            string                        `json:"source"`
	SeedID            string                        `json:"seedId"`
	Direction         SemanticScholarGraphDirection `json:"direction"`
	Limit             int                           `json:"limit"`
	MaxDepth          int                           `json:"maxDepth"`
	MaxNodes          int                           `json:"maxNodes"`
	MaxAPICalls       int                           `json:"maxApiCalls"`
	RetryBudget       int                           `json:"retryBudget"`
	ResumeCursor      string                        `json:"resumeCursor,omitempty"`
	EstimatedAPICalls int                           `json:"estimatedApiCalls"`
	EstimatedMaxNodes int                           `json:"estimatedMaxNodes"`
	WithinBudget      bool                          `json:"withinBudget"`
	DryRunPlan        string                        `json:"dryRunPlan"`
}

func NormalizeGraphExpansionBudget(b GraphExpansionBudget) GraphExpansionBudget {
	if b.MaxDepth <= 0 {
		b.MaxDepth = 1
	}
	if b.MaxNodes <= 0 {
		b.MaxNodes = 25
	}
	if b.MaxAPICalls <= 0 {
		b.MaxAPICalls = 1
	}
	if b.RetryBudget < 0 {
		b.RetryBudget = 0
	}
	return b
}

func EstimateGraphExpansionBudget(source, seedID string, direction SemanticScholarGraphDirection, limit int, budget GraphExpansionBudget) GraphExpansionBudgetEstimate {
	budget = NormalizeGraphExpansionBudget(budget)
	if limit <= 0 {
		limit = 25
	}
	estimatedCalls := budget.MaxDepth
	if direction == SemanticScholarDirectionBoth {
		estimatedCalls *= 2
	}
	if estimatedCalls > budget.MaxAPICalls {
		estimatedCalls = budget.MaxAPICalls
	}
	estimatedNodes := limit * budget.MaxDepth
	if estimatedNodes > budget.MaxNodes {
		estimatedNodes = budget.MaxNodes
	}
	return GraphExpansionBudgetEstimate{SchemaVersion: "1", Source: source, SeedID: seedID, Direction: direction, Limit: limit, MaxDepth: budget.MaxDepth, MaxNodes: budget.MaxNodes, MaxAPICalls: budget.MaxAPICalls, RetryBudget: budget.RetryBudget, ResumeCursor: budget.ResumeCursor, EstimatedAPICalls: estimatedCalls, EstimatedMaxNodes: estimatedNodes, WithinBudget: estimatedCalls <= budget.MaxAPICalls && estimatedNodes <= budget.MaxNodes, DryRunPlan: fmt.Sprintf("expand %s %s depth=%d nodes<=%d api_calls<=%d retries=%d", source, seedID, budget.MaxDepth, budget.MaxNodes, budget.MaxAPICalls, budget.RetryBudget)}
}
