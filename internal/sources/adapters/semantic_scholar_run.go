package adapters

import (
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
)

import "strings"

type SemanticScholarGraphRunOptions struct {
	SeedID                   string
	Direction                core.SemanticScholarGraphDirection
	Limit, Depth, MaxRecords int
	RequestedFields          []string
	QuotaRemaining           int
	Budget                   core.GraphExpansionBudget
}
type SemanticScholarGraphRun struct {
	SchemaVersion     string                             `json:"schemaVersion"`
	SeedID            string                             `json:"seedId"`
	Direction         core.SemanticScholarGraphDirection `json:"direction"`
	Limit             int                                `json:"limit"`
	Depth             int                                `json:"depth"`
	MaxRecords        int                                `json:"maxRecords"`
	RequestedFields   []string                           `json:"requestedFields"`
	FieldRestrictions []string                           `json:"fieldRestrictions,omitempty"`
	QuotaRemaining    int                                `json:"quotaRemaining"`
	Budget            core.GraphExpansionBudget          `json:"budget"`
	BudgetEstimate    core.GraphExpansionBudgetEstimate  `json:"budgetEstimate"`
	ResumeCursor      string                             `json:"resumeCursor,omitempty"`
	Visited           []string                           `json:"visited"`
	NextFrontier      []string                           `json:"nextFrontier"`
	EdgeCount         int                                `json:"edgeCount"`
	RecordCount       int                                `json:"recordCount"`
	Completed         bool                               `json:"completed"`
}

func NewSemanticScholarGraphRun(opts SemanticScholarGraphRunOptions) SemanticScholarGraphRun {
	direction := opts.Direction
	if direction == "" {
		direction = core.SemanticScholarDirectionBoth
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	depth := opts.Depth
	if depth <= 0 {
		depth = 1
	}
	budget := core.NormalizeGraphExpansionBudget(opts.Budget)
	if opts.MaxRecords > 0 {
		budget.MaxNodes = opts.MaxRecords
	}
	if opts.Depth > 0 {
		budget.MaxDepth = opts.Depth
	}
	run := SemanticScholarGraphRun{SchemaVersion: "1", SeedID: strings.TrimSpace(opts.SeedID), Direction: direction, Limit: limit, Depth: depth, MaxRecords: opts.MaxRecords, RequestedFields: append([]string{}, opts.RequestedFields...), QuotaRemaining: opts.QuotaRemaining, Budget: budget, ResumeCursor: budget.ResumeCursor}
	run.BudgetEstimate = core.EstimateGraphExpansionBudget("semantic-scholar", run.SeedID, direction, limit, budget)
	if run.SeedID != "" {
		run.NextFrontier = []string{run.SeedID}
	}
	for _, field := range opts.RequestedFields {
		if semanticScholarRestrictedField(field) {
			run.FieldRestrictions = append(run.FieldRestrictions, field+": field requires explicit API entitlement or may drift")
		}
	}
	return run
}
func (r SemanticScholarGraphRun) HasFieldRestriction(field string) bool {
	for _, restriction := range r.FieldRestrictions {
		if strings.HasPrefix(restriction, field+":") {
			return true
		}
	}
	return false
}
func (r SemanticScholarGraphRun) RecordExpansion(expansion core.CitationGraphExpansion, nextFrontier []string, quotaRemaining int) SemanticScholarGraphRun {
	r.EdgeCount += len(expansion.Edges)
	r.RecordCount += len(expansion.Records)
	r.Visited = append(r.Visited, expansion.SeedID)
	r.NextFrontier = append([]string{}, nextFrontier...)
	r.QuotaRemaining = quotaRemaining
	r.Completed = len(r.NextFrontier) == 0
	return r
}
func semanticScholarRestrictedField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "embedding", "tldr", "s2fields", "influentialcitationcount":
		return true
	default:
		return false
	}
}
