package protocol

import "testing"

func TestDraftQueryExpansionSuggestionsRequireSourceTextAndReview(t *testing.T) {
	records, err := DraftQueryExpansionSuggestions(QueryExpansionInput{
		Question:    "Do high entropy alloy catalysts improve hydrogen evolution?",
		SourceTexts: []SourceTextLink{{ID: "abstract-1", PaperID: "paper-1", PassageRef: "abstract:0-128", Text: "High entropy alloy catalysts accelerate the hydrogen evolution reaction."}},
	})
	if err != nil {
		t.Fatalf("DraftQueryExpansionSuggestions: %v", err)
	}
	if len(records) < 3 {
		t.Fatalf("expected suggestions from KeyBERT/SciSpaCy/LLM assistants: %#v", records)
	}
	seen := map[SuggestionAssistant]bool{}
	for _, record := range records {
		seen[record.Assistant] = true
		if len(record.SourceTextLinks) == 0 {
			t.Fatalf("suggestion missing source text links: %#v", record)
		}
		if record.ReviewerApprovalRequired != true || record.ReviewerApproved != false {
			t.Fatalf("suggestion must be gated by reviewer approval: %#v", record)
		}
		if record.ProvenanceTag == "" || record.SuggestedTerm == "" || record.Score <= 0 || record.DiversityScore <= 0 || record.ExtractionMethod == "" {
			t.Fatalf("suggestion missing provenance/term/scoring: %#v", record)
		}
	}
	for _, assistant := range []SuggestionAssistant{AssistantKeyBERT, AssistantSciSpaCy, AssistantLLM} {
		if !seen[assistant] {
			t.Fatalf("missing assistant %s in %#v", assistant, records)
		}
	}
}

func TestQueryExpansionCannotChangeSourcePlanWithoutApproval(t *testing.T) {
	plan, err := CompileSourcePlanFromQuestion(QuestionInput{Question: "Do catalysts improve hydrogen evolution?"})
	if err != nil {
		t.Fatalf("CompileSourcePlanFromQuestion: %v", err)
	}
	unapproved := QueryExpansionSuggestion{ID: "qe-1", Assistant: AssistantKeyBERT, SuggestedTerm: "hydrogen evolution reaction", SourceTextLinks: []SourceTextLink{{ID: "abstract-1", Text: "hydrogen evolution reaction"}}, ReviewerApprovalRequired: true, ReviewerApproved: false}
	if _, err := ApplyApprovedQueryExpansions(plan, []QueryExpansionSuggestion{unapproved}); err == nil {
		t.Fatalf("expected unapproved suggestion to be rejected")
	}
	approved := unapproved
	approved.ReviewerApproved = true
	updated, err := ApplyApprovedQueryExpansions(plan, []QueryExpansionSuggestion{approved})
	if err != nil {
		t.Fatalf("ApplyApprovedQueryExpansions approved suggestion: %v", err)
	}
	if updated.MustSource("openalex").Query == plan.MustSource("openalex").Query {
		t.Fatalf("approved suggestion did not change query")
	}
	if !contains(updated.MustSource("openalex").Query, "hydrogen evolution reaction") {
		t.Fatalf("expanded query missing approved term: %s", updated.MustSource("openalex").Query)
	}
	if len(updated.QueryExpansionProvenance) == 0 || updated.QueryExpansionProvenance[0].BeforeQuery == "" || updated.QueryExpansionProvenance[0].AfterQuery == "" || updated.QueryExpansionProvenance[0].SuggestionID != "qe-1" {
		t.Fatalf("missing before/after search-plan provenance: %#v", updated.QueryExpansionProvenance)
	}
}

func TestQueryExpansionRejectsUnsupportedSuggestion(t *testing.T) {
	plan, err := CompileSourcePlanFromQuestion(QuestionInput{Question: "Do catalysts improve hydrogen evolution?"})
	if err != nil {
		t.Fatalf("CompileSourcePlanFromQuestion: %v", err)
	}
	unsupported := QueryExpansionSuggestion{ID: "qe-2", Assistant: AssistantLLM, SuggestedTerm: "unsupported", ReviewerApprovalRequired: true, ReviewerApproved: true}
	if _, err := ApplyApprovedQueryExpansions(plan, []QueryExpansionSuggestion{unsupported}); err == nil {
		t.Fatalf("expected missing source text link rejection")
	}
}
