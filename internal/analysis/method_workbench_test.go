package analysis

import "testing"

func TestDefaultMethodComparisonWorkbenchCoversSpineMethodFamilies(t *testing.T) {
	workbench := DefaultMethodComparisonWorkbench()
	want := []string{"parser choices", "retrieval backends", "screening rankers", "effect-size models", "publication-bias diagnostics"}
	for _, category := range want {
		if len(workbench.OptionsByCategory(category)) == 0 {
			t.Fatalf("missing category %q in %#v", category, workbench)
		}
	}
	if workbench.SchemaVersion != "1" || workbench.GeneratedFrom == "" {
		t.Fatalf("workbench metadata = %#v", workbench)
	}
	parser := workbench.OptionsByCategory("parser choices")[0]
	if parser.Name == "" || parser.AuditArtifact == "" || len(parser.ComparisonCriteria) == 0 {
		t.Fatalf("parser option = %#v", parser)
	}
}

func TestMethodComparisonWorkbenchRecommendationFlagsReviewRequiredTradeoffs(t *testing.T) {
	workbench := DefaultMethodComparisonWorkbench()
	report := workbench.Compare("publication-bias diagnostics", []string{"egger", "begg"})
	if report.Category != "publication-bias diagnostics" || len(report.Options) != 2 || !report.RequiresReviewerChoice || report.Recommendation == "" {
		t.Fatalf("report = %#v", report)
	}
}
