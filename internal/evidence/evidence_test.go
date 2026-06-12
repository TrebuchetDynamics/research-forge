package evidence

import "testing"

func TestExtractionSchemaValidationAndEvidenceStatusTransitions(t *testing.T) {
	schema, err := NewSchema(SchemaInput{Name: "catalyst outcomes", Fields: []Field{{Name: "catalyst", Type: "string"}, {Name: "efficiency", Type: "number"}}})
	if err != nil {
		t.Fatalf("NewSchema returned error: %v", err)
	}
	if schema.SchemaVersion != "1" || schema.Fields[1].Name != "efficiency" {
		t.Fatalf("schema = %#v", schema)
	}
	item, err := NewEvidenceItem(EvidenceInput{PaperID: "paper-1", SchemaName: schema.Name, Values: map[string]string{"catalyst": "TiO2"}, Support: Support{Kind: SupportPassage, Ref: "paper-1-sec-1-p-1"}, Status: StatusSuggested})
	if err != nil {
		t.Fatalf("NewEvidenceItem returned error: %v", err)
	}
	if err := item.Transition(StatusAccepted, "ada", "verified passage"); err != nil {
		t.Fatalf("Transition accepted returned error: %v", err)
	}
	if item.Status != StatusAccepted || len(item.History) != 1 || item.History[0].Reviewer != "ada" {
		t.Fatalf("item = %#v", item)
	}
	if err := item.Transition(StatusSuggested, "ada", "regress"); err == nil {
		t.Fatalf("Transition back to suggested returned nil error")
	}
}

func TestEvidenceAuditRequiresSupportForAcceptedEvidence(t *testing.T) {
	unsupported := EvidenceItem{Status: StatusAccepted, Support: Support{}}
	issues := Audit([]EvidenceItem{unsupported})
	if len(issues) != 1 || issues[0].Code != "accepted_without_support" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestLLMConfigurationRedactsSecrets(t *testing.T) {
	config := LLMConfig{Provider: "test", Model: "fixture", APIKey: "secret-key"}
	redacted := config.Redacted()
	if redacted.APIKey != "[redacted]" || redacted.Provider != "test" || redacted.Model != "fixture" {
		t.Fatalf("redacted = %#v", redacted)
	}
}

func TestLLMSuggestionCannotBecomeAcceptedWithoutReview(t *testing.T) {
	suggestion, err := SuggestWithLLM(NoopSuggestionAdapter{}, SuggestRequest{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("SuggestWithLLM returned error: %v", err)
	}
	if suggestion.Status != StatusSuggested || suggestion.SuggestedBy != "noop-llm" {
		t.Fatalf("suggestion = %#v", suggestion)
	}
	if err := suggestion.AcceptWithoutReview(); err == nil {
		t.Fatalf("AcceptWithoutReview returned nil error")
	}
}
