package evidence

import "testing"

func TestCitationLockedSynthesisSupportsAllAssistedKindsWithSentenceLocks(t *testing.T) {
	for _, kind := range []CitationLockedSuggestionKind{CitationLockedQueryExpansion, CitationLockedScreeningRationale, CitationLockedExtraction, CitationLockedReportProse} {
		queue, err := DraftCitationLockedSuggestions(CitationLockedSuggestionRequest{PaperID: "paper-1", Kind: kind, Prompt: "draft", Supports: []CitationLockedSupport{{Ref: "paper-1:p1", ExactText: "First supported sentence."}, {Ref: "paper-1:p2", ExactText: "Second supported sentence."}}})
		if err != nil {
			t.Fatalf("kind %s: %v", kind, err)
		}
		if queue.Suggestions[0].Status != StatusSuggested || !EverySuggestedSentenceCitationLocked(queue.Suggestions[0]) {
			t.Fatalf("kind %s suggestion not locked/unaccepted: %#v", kind, queue.Suggestions[0])
		}
	}
}

func TestCitationLockedLLMSuggestionsRequireSupportAndReviewerApproval(t *testing.T) {
	queue, err := DraftCitationLockedSuggestions(CitationLockedSuggestionRequest{
		PaperID:      "paper-1",
		Kind:         CitationLockedReportProse,
		Prompt:       "summarize primary outcome",
		Supports:     []CitationLockedSupport{{Ref: "paper-1:p1", ExactText: "Mortality was lower in the treatment group."}},
		ModelName:    "local-llm-fixture",
		ModelVersion: "1.0",
	})
	if err != nil {
		t.Fatalf("DraftCitationLockedSuggestions returned error: %v", err)
	}
	if queue.SchemaVersion != "1" || len(queue.Suggestions) != 1 {
		t.Fatalf("queue = %#v", queue)
	}
	suggestion := queue.Suggestions[0]
	if suggestion.Status != StatusSuggested || suggestion.Kind != CitationLockedReportProse || suggestion.SuggestedText == "" || suggestion.ModelName != "local-llm-fixture" || suggestion.ModelVersion != "1.0" {
		t.Fatalf("suggestion = %#v", suggestion)
	}
	if len(suggestion.CitationLocks) != 1 || suggestion.CitationLocks[0].ExactText == "" || suggestion.CitationLocks[0].Ref != "paper-1:p1" {
		t.Fatalf("citation locks = %#v", suggestion.CitationLocks)
	}
	if err := suggestion.AcceptWithoutReview(); err == nil {
		t.Fatalf("expected direct accept rejection")
	}
	if _, err := DraftCitationLockedSuggestions(CitationLockedSuggestionRequest{PaperID: "paper-1", Kind: CitationLockedExtraction, Prompt: "extract"}); err == nil {
		t.Fatalf("expected missing citation support error")
	}
	reviewed, err := ReviewCitationLockedSuggestion(queue, CitationLockedReviewInput{SuggestionID: suggestion.ID, Decision: StatusAccepted, Reviewer: "ada", Note: "quote checked"})
	if err != nil {
		t.Fatalf("ReviewCitationLockedSuggestion returned error: %v", err)
	}
	if reviewed.Suggestions[0].Status != StatusAccepted || reviewed.Suggestions[0].ReviewerDecision.Reviewer != "ada" {
		t.Fatalf("reviewed = %#v", reviewed.Suggestions[0])
	}
	if _, err := ReviewCitationLockedSuggestion(queue, CitationLockedReviewInput{SuggestionID: suggestion.ID, Decision: StatusAccepted}); err == nil {
		t.Fatalf("expected reviewer required")
	}
}
