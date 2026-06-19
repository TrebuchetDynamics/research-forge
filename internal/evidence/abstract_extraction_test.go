package evidence

import "testing"

func TestSTHExtractionSchemaPresetHasRequiredFields(t *testing.T) {
	schema := STHExtractionSchemaPreset()
	if schema.Name != "sth-efficiency" {
		t.Fatalf("name = %q", schema.Name)
	}
	wantRequired := map[string]bool{
		"value_pct": true, "device_type": true, "auxiliary_bias": true,
		"measurement_standard": true, "verbatim_quote": true, "confidence": true,
	}
	for _, f := range schema.Required {
		if !wantRequired[f.Name] {
			t.Fatalf("unexpected required field %q", f.Name)
		}
		if !f.Required {
			t.Fatalf("field %q should have Required=true", f.Name)
		}
		delete(wantRequired, f.Name)
	}
	if len(wantRequired) != 0 {
		t.Fatalf("missing required fields: %v", wantRequired)
	}
}

func TestSTHExtractionSchemaPresetHasOptionalFields(t *testing.T) {
	schema := STHExtractionSchemaPreset()
	wantOptional := map[string]bool{
		"ci_lower": true, "ci_upper": true, "se": true,
		"target_reaction": true, "electrode_material": true,
		"electrolyte": true, "illumination_intensity_mwcm2": true, "active_area_cm2": true,
	}
	for _, f := range schema.Optional {
		if !wantOptional[f.Name] {
			t.Fatalf("unexpected optional field %q", f.Name)
		}
		delete(wantOptional, f.Name)
	}
	if len(wantOptional) != 0 {
		t.Fatalf("missing optional fields: %v", wantOptional)
	}
}

func TestSuggestRequestCarriesAbstractTextAndTargetField(t *testing.T) {
	req := SuggestRequest{
		PaperID:      "paper-1",
		AbstractText: "We demonstrate 12.5% STH efficiency under AM1.5G illumination.",
		TargetField:  ExtractionTarget{Name: "value_pct", Unit: "%", PromptHint: "solar-to-hydrogen efficiency"},
	}
	if req.PaperID != "paper-1" || req.AbstractText == "" || req.TargetField.Name != "value_pct" {
		t.Fatalf("request = %+v", req)
	}
}

func TestNoopSuggestionAdapterIgnoresAbstractText(t *testing.T) {
	adapter := NoopSuggestionAdapter{}
	req := SuggestRequest{
		PaperID:      "paper-1",
		AbstractText: "12.5% STH efficiency.",
		TargetField:  ExtractionTarget{Name: "value_pct", Unit: "%"},
	}
	item, err := SuggestWithLLM(adapter, req)
	if err != nil {
		t.Fatalf("SuggestWithLLM error: %v", err)
	}
	if item.PaperID != "paper-1" || item.Status != StatusSuggested || item.SuggestedBy != "noop-llm" {
		t.Fatalf("item = %+v", item)
	}
}
