package search

import "testing"

func TestNewStrategyDefinesSavedSearchModelWithVersionAndSchedule(t *testing.T) {
	strategy, err := NewStrategy(StrategyInput{
		ID:    " artificial-photosynthesis ",
		Title: " Artificial Photosynthesis Review ",
		Concepts: []ConceptInput{{
			Name:     "photosynthesis",
			Terms:    []string{"artificial photosynthesis"},
			Synonyms: []string{"solar fuels", "photoelectrochemical water splitting"},
		}},
		Fields:   []FieldQuery{{Field: "title", Query: "catalyst"}},
		Schedule: WatchedSchedule{Enabled: true, Interval: "weekly"},
	})
	if err != nil {
		t.Fatalf("NewStrategy returned error: %v", err)
	}
	if strategy.SchemaVersion != "1" || strategy.ID != "artificial-photosynthesis" || strategy.Title != "Artificial Photosynthesis Review" {
		t.Fatalf("strategy identity = %#v", strategy)
	}
	if len(strategy.Concepts) != 1 || strategy.Concepts[0].Terms[0] != "artificial photosynthesis" || strategy.Concepts[0].Synonyms[1] != "photoelectrochemical water splitting" {
		t.Fatalf("concepts = %#v", strategy.Concepts)
	}
	if strategy.Fields[0].Field != "title" || strategy.Fields[0].Query != "catalyst" {
		t.Fatalf("fields = %#v", strategy.Fields)
	}
	if !strategy.Schedule.Enabled || strategy.Schedule.Interval != "weekly" {
		t.Fatalf("schedule = %#v", strategy.Schedule)
	}
}

func TestStrategyProvenanceIncludesVersionAndWatchedSchedule(t *testing.T) {
	strategy, err := NewStrategy(StrategyInput{
		ID:       "ap-review",
		Title:    "AP Review",
		Concepts: []ConceptInput{{Name: "process", Terms: []string{"artificial photosynthesis"}}},
		Schedule: WatchedSchedule{Enabled: true, Interval: "daily"},
	})
	if err != nil {
		t.Fatalf("NewStrategy returned error: %v", err)
	}

	metadata := strategy.ProvenanceMetadata()
	if metadata["schema_version"] != "1" || metadata["strategy_id"] != "ap-review" || metadata["watched_interval"] != "daily" || metadata["watched_enabled"] != "true" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestStrategyBuildsBooleanQueryFromConceptsSynonymsAndFields(t *testing.T) {
	strategy, err := NewStrategy(StrategyInput{
		ID:    "ap-review",
		Title: "AP Review",
		Concepts: []ConceptInput{
			{Name: "process", Terms: []string{"artificial photosynthesis"}, Synonyms: []string{"solar fuels"}},
			{Name: "material", Terms: []string{"catalyst"}, Synonyms: []string{"photocatalyst"}},
		},
		Fields: []FieldQuery{{Field: "title", Query: "review"}, {Field: "abstract", Query: "water splitting"}},
	})
	if err != nil {
		t.Fatalf("NewStrategy returned error: %v", err)
	}

	query := strategy.BooleanQuery()
	want := `("artificial photosynthesis" OR "solar fuels") AND (catalyst OR photocatalyst) AND title:review AND abstract:"water splitting"`
	if query != want {
		t.Fatalf("BooleanQuery = %q, want %q", query, want)
	}
}
