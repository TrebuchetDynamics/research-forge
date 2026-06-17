package analysis

import "strings"

type MethodComparisonWorkbench struct {
	SchemaVersion string           `json:"schemaVersion"`
	GeneratedFrom string           `json:"generatedFrom"`
	Categories    []MethodCategory `json:"categories"`
}

type MethodCategory struct {
	Name    string         `json:"name"`
	Options []MethodOption `json:"options"`
}

type MethodOption struct {
	Name               string   `json:"name"`
	Role               string   `json:"role"`
	AuditArtifact      string   `json:"auditArtifact"`
	ComparisonCriteria []string `json:"comparisonCriteria"`
	Tradeoffs          []string `json:"tradeoffs"`
}

type MethodComparisonReport struct {
	Category               string         `json:"category"`
	Options                []MethodOption `json:"options"`
	RequiresReviewerChoice bool           `json:"requiresReviewerChoice"`
	Recommendation         string         `json:"recommendation"`
}

func DefaultMethodComparisonWorkbench() MethodComparisonWorkbench {
	return MethodComparisonWorkbench{SchemaVersion: "1", GeneratedFrom: "researchforge-method-registry", Categories: []MethodCategory{
		{Name: "parser choices", Options: []MethodOption{
			{Name: "grobid", Role: "structured PDF parser", AuditArtifact: "parser arbitration report", ComparisonCriteria: []string{"field coverage", "offset stability", "reference extraction", "warnings"}, Tradeoffs: []string{"external service optional", "strong TEI structure"}},
			{Name: "s2orc", Role: "JSON full-text parser", AuditArtifact: "parser comparison report", ComparisonCriteria: []string{"section coverage", "citation spans", "license provenance"}, Tradeoffs: []string{"best when source JSON is available"}},
		}},
		{Name: "retrieval backends", Options: []MethodOption{
			{Name: "sqlite-fts", Role: "local lexical retrieval", AuditArtifact: "retrieval benchmark report", ComparisonCriteria: []string{"determinism", "recall@k", "latency"}, Tradeoffs: []string{"fast local baseline"}},
			{Name: "opensearch", Role: "external lexical retrieval", AuditArtifact: "opensearch bulk report", ComparisonCriteria: []string{"mapping lock", "highlight passages", "bulk failures"}, Tradeoffs: []string{"optional external service"}},
			{Name: "qdrant", Role: "vector retrieval", AuditArtifact: "qdrant index report", ComparisonCriteria: []string{"embedding model", "payload privacy", "vector lock"}, Tradeoffs: []string{"requires embedding policy"}},
		}},
		{Name: "screening rankers", Options: []MethodOption{
			{Name: "active-learning", Role: "seed driven prioritization", AuditArtifact: "active learning run", ComparisonCriteria: []string{"input hash", "decision hash", "recall simulation"}, Tradeoffs: []string{"depends on reviewer seeds"}},
			{Name: "uncertainty", Role: "uncertain record prioritization", AuditArtifact: "sensitivity diagnostics", ComparisonCriteria: []string{"uncertainty score", "exploration score"}, Tradeoffs: []string{"useful for adjudication queues"}},
		}},
		{Name: "effect-size models", Options: []MethodOption{
			{Name: "smd", Role: "continuous outcomes", AuditArtifact: "analysis input table", ComparisonCriteria: []string{"required fields", "variance", "scale"}, Tradeoffs: []string{"standardized but less interpretable"}},
			{Name: "log-odds-ratio", Role: "binary outcomes", AuditArtifact: "analysis input table", ComparisonCriteria: []string{"zero-cell correction", "variance"}, Tradeoffs: []string{"log scale"}},
			{Name: "risk-difference", Role: "absolute binary difference", AuditArtifact: "analysis input table", ComparisonCriteria: []string{"event rates", "absolute scale"}, Tradeoffs: []string{"scale depends on baseline risk"}},
		}},
		{Name: "publication-bias diagnostics", Options: []MethodOption{
			{Name: "egger", Role: "small-study effect regression", AuditArtifact: "publication bias report", ComparisonCriteria: []string{"intercept", "precision", "warnings"}, Tradeoffs: []string{"underpowered with few studies"}},
			{Name: "begg", Role: "rank-correlation diagnostic", AuditArtifact: "publication bias report", ComparisonCriteria: []string{"kendall tau", "warnings"}, Tradeoffs: []string{"non-parametric cross-check"}},
		}},
	}}
}

func (w MethodComparisonWorkbench) OptionsByCategory(category string) []MethodOption {
	for _, item := range w.Categories {
		if strings.EqualFold(item.Name, strings.TrimSpace(category)) {
			return append([]MethodOption{}, item.Options...)
		}
	}
	return nil
}

func (w MethodComparisonWorkbench) Compare(category string, names []string) MethodComparisonReport {
	available := w.OptionsByCategory(category)
	want := map[string]bool{}
	for _, name := range names {
		want[strings.ToLower(strings.TrimSpace(name))] = true
	}
	report := MethodComparisonReport{Category: category}
	for _, option := range available {
		if len(want) == 0 || want[strings.ToLower(option.Name)] {
			report.Options = append(report.Options, option)
		}
	}
	report.RequiresReviewerChoice = len(report.Options) > 1
	if report.RequiresReviewerChoice {
		report.Recommendation = "review tradeoffs and audit artifacts before changing the primary method"
	} else if len(report.Options) == 1 {
		report.Recommendation = "single method selected; verify required audit artifact exists"
	} else {
		report.Recommendation = "no matching methods found"
	}
	return report
}
