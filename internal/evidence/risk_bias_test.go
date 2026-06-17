package evidence

import "testing"

func TestRiskOfBiasTemplatesAndSuggestionReviewQueue(t *testing.T) {
	templates := DefaultRiskOfBiasSchemaTemplates()
	if len(templates) < 2 || templates[0].SchemaVersion != "1" || templates[0].Domain == "" || len(templates[0].Fields) == 0 {
		t.Fatalf("templates = %#v", templates)
	}
	queue := DraftRiskOfBiasSuggestionQueue(RiskOfBiasSuggestionRequest{
		PaperID:      "paper-1",
		Passages:     []SupportText{{Ref: "sec-1:p1", Text: "Random sequence generation was computer generated, but participants and personnel were not blinded."}},
		ModelName:    "robotreviewer-inspired-fixture",
		ModelVersion: "2026-06-17",
	})
	if len(queue.Suggestions) == 0 {
		t.Fatalf("expected suggestions")
	}
	first := queue.Suggestions[0]
	if first.ExactSupportText == "" || first.SupportRef == "" || first.ModelName == "" || first.ModelVersion == "" || first.Uncertainty <= 0 || first.Status != RiskOfBiasSuggested {
		t.Fatalf("suggestion missing audit metadata: %#v", first)
	}
	if !EveryRiskOfBiasJudgmentAuditable(queue) {
		t.Fatalf("queue should require exact support, uncertainty, model/version, and reviewer state: %#v", queue)
	}
	reviewed, err := ReviewRiskOfBiasSuggestion(queue, RiskOfBiasReviewInput{SuggestionID: first.ID, Decision: RiskOfBiasAccepted, Reviewer: "ada", Note: "support text checked"})
	if err != nil {
		t.Fatalf("review returned error: %v", err)
	}
	if reviewed.Suggestions[0].Status != RiskOfBiasAccepted || reviewed.Suggestions[0].ReviewerDecision.Reviewer != "ada" {
		t.Fatalf("reviewed = %#v", reviewed.Suggestions[0])
	}
	if _, err := ReviewRiskOfBiasSuggestion(queue, RiskOfBiasReviewInput{SuggestionID: first.ID, Decision: RiskOfBiasAccepted}); err == nil {
		t.Fatalf("expected missing reviewer error")
	}
	if _, err := ReviewRiskOfBiasSuggestion(queue, RiskOfBiasReviewInput{SuggestionID: first.ID, Decision: RiskOfBiasCorrected, Reviewer: "ada"}); err != nil {
		t.Fatalf("correct decision should be allowed: %v", err)
	}
	if _, err := ReviewRiskOfBiasSuggestion(queue, RiskOfBiasReviewInput{SuggestionID: first.ID, Decision: RiskOfBiasRejected, Reviewer: "ada"}); err != nil {
		t.Fatalf("reject decision should be allowed: %v", err)
	}
	queue.Suggestions[0].ExactSupportText = ""
	if EveryRiskOfBiasJudgmentAuditable(queue) {
		t.Fatalf("missing support text should fail audit")
	}
}
