package sources

import "testing"

func TestGraphExpansionBudgetEstimateAndResumeCursor(t *testing.T) {
	budget := NormalizeGraphExpansionBudget(GraphExpansionBudget{MaxDepth: 3, MaxNodes: 50, MaxAPICalls: 7, RetryBudget: 2, ResumeCursor: "frontier:W2"})
	estimate := EstimateGraphExpansionBudget("openalex", "W1", SemanticScholarDirectionBoth, 25, budget)
	if estimate.Source != "openalex" || estimate.SeedID != "W1" || estimate.EstimatedAPICalls != 6 || estimate.MaxNodes != 50 || estimate.ResumeCursor != "frontier:W2" {
		t.Fatalf("estimate = %#v", estimate)
	}
	if !estimate.WithinBudget || estimate.DryRunPlan == "" {
		t.Fatalf("bad estimate = %#v", estimate)
	}
}

func TestGraphExpansionBudgetDefaults(t *testing.T) {
	budget := NormalizeGraphExpansionBudget(GraphExpansionBudget{})
	if budget.MaxDepth != 1 || budget.MaxNodes != 25 || budget.MaxAPICalls != 1 || budget.RetryBudget != 0 {
		t.Fatalf("budget = %#v", budget)
	}
}
