package evidence

import "testing"

func TestAnalyzeEvidenceGapsFindsMissingOutcomesComparatorsUnsupportedClaimsFullTextAndAnalysisReadiness(t *testing.T) {
	items := []EvidenceItem{
		{PaperID: "p1", SchemaName: "outcomes", Values: map[string]string{"outcome": "mortality", "comparator": "placebo", "effect": "0.8"}, Support: Support{Kind: SupportPassage, Ref: "p1:p1"}, Status: StatusAccepted},
		{PaperID: "p2", SchemaName: "outcomes", Values: map[string]string{"outcome": "quality of life", "comparator": "usual care"}, Status: StatusAccepted},
	}
	claims := []CitationLockedSuggestion{{ID: "claim-1", PaperID: "p2", Status: StatusSuggested, SuggestedText: "Unsupported draft claim"}}
	report := AnalyzeEvidenceGaps(EvidenceGapAnalysisInput{
		Items:                     items,
		Claims:                    claims,
		RequiredOutcomes:          []string{"mortality", "hospitalization"},
		RequiredComparators:       []string{"placebo", "standard care"},
		FullTextRequiredPaperIDs:  []string{"p1", "p2"},
		AvailableFullTextPaperIDs: []string{"p1"},
		AnalysisIncludedPaperIDs:  []string{"p1"},
	})
	if report.SchemaVersion != "1" || report.ReadyForAnalysis {
		t.Fatalf("report = %#v", report)
	}
	wantCodes := map[string]bool{"missing_outcome": false, "missing_comparator": false, "unsupported_claim": false, "incomplete_full_text": false, "analysis_input_not_ready": false}
	for _, gap := range report.Gaps {
		if _, ok := wantCodes[gap.Code]; ok {
			wantCodes[gap.Code] = true
		}
	}
	for code, seen := range wantCodes {
		if !seen {
			t.Fatalf("missing gap code %s in %#v", code, report.Gaps)
		}
	}
}

func TestAnalyzeEvidenceGapsReadyWhenRequirementsCovered(t *testing.T) {
	items := []EvidenceItem{{PaperID: "p1", Values: map[string]string{"outcome": "mortality", "comparator": "placebo", "effect": "0.8"}, Support: Support{Kind: SupportPassage, Ref: "p1:p1"}, Status: StatusAccepted}}
	report := AnalyzeEvidenceGaps(EvidenceGapAnalysisInput{Items: items, RequiredOutcomes: []string{"mortality"}, RequiredComparators: []string{"placebo"}, FullTextRequiredPaperIDs: []string{"p1"}, AvailableFullTextPaperIDs: []string{"p1"}, AnalysisIncludedPaperIDs: []string{"p1"}})
	if !report.ReadyForAnalysis || len(report.Gaps) != 0 {
		t.Fatalf("report = %#v", report)
	}
}
