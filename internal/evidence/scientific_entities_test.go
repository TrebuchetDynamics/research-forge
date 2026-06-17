package evidence

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestScientificEntitySuggestionsCaptureOffsetsAbbreviationsLinksModelAndReview(t *testing.T) {
	passage := parsing.Passage{ID: "p1", PaperID: "paper-1", Text: "Tumor necrosis factor (TNF) increased after aspirin treatment.", Offset: parsing.TextOffset{Start: 100, End: 160}}
	queue := DraftScientificEntitySuggestions(ScientificEntitySuggestionRequest{PaperID: "paper-1", Passages: []parsing.Passage{passage}, ModelName: "scispacy-inspired-fixture", ModelVersion: "1.0"})
	if queue.SchemaVersion != "1" || len(queue.Suggestions) == 0 {
		t.Fatalf("queue = %#v", queue)
	}
	var tnf ScientificEntitySuggestion
	for _, suggestion := range queue.Suggestions {
		if suggestion.Mention == "TNF" {
			tnf = suggestion
		}
	}
	if tnf.ID == "" || tnf.PassageID != "p1" || tnf.Offset.Start <= passage.Offset.Start || tnf.Abbreviation.LongForm != "Tumor necrosis factor" {
		t.Fatalf("tnf suggestion = %#v", tnf)
	}
	if len(tnf.EntityLinkCandidates) == 0 || tnf.Confidence <= 0 || tnf.ModelName != "scispacy-inspired-fixture" || tnf.ModelVersion != "1.0" || tnf.Status != EntitySuggested {
		t.Fatalf("tnf metadata = %#v", tnf)
	}
	if !EveryScientificEntitySuggestionAuditable(queue) {
		t.Fatalf("queue should be auditable: %#v", queue)
	}
	reviewed, err := ReviewScientificEntitySuggestion(queue, ScientificEntityReviewInput{SuggestionID: tnf.ID, Decision: EntityAccepted, Reviewer: "ada", Note: "matches passage"})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if reviewed.Suggestions[0].Status == EntitySuggested {
		t.Fatalf("review did not update queue: %#v", reviewed)
	}
	if _, err := ReviewScientificEntitySuggestion(queue, ScientificEntityReviewInput{SuggestionID: tnf.ID, Decision: EntityAccepted}); err == nil {
		t.Fatalf("expected reviewer required error")
	}
	if _, err := ReviewScientificEntitySuggestion(queue, ScientificEntityReviewInput{SuggestionID: tnf.ID, Decision: EntityCorrected, Reviewer: "ada"}); err != nil {
		t.Fatalf("corrected decision should be allowed: %v", err)
	}
	queue.Suggestions[0].EntityLinkCandidates = nil
	if EveryScientificEntitySuggestionAuditable(queue) {
		t.Fatalf("missing entity-link candidates should fail audit")
	}
}
