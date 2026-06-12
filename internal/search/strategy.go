package search

import (
	"fmt"
	"strings"
)

// Strategy is a saved, versioned scholarly search strategy.
type Strategy struct {
	SchemaVersion string
	ID            string
	Title         string
	Concepts      []Concept
	Fields        []FieldQuery
	Schedule      WatchedSchedule
}

// StrategyInput is caller-provided search strategy metadata before normalization.
type StrategyInput struct {
	ID       string
	Title    string
	Concepts []ConceptInput
	Fields   []FieldQuery
	Schedule WatchedSchedule
}

// Concept is one Boolean search concept with expansion terms.
type Concept struct {
	Name     string
	Terms    []string
	Synonyms []string
}

// ConceptInput is caller-provided concept metadata before normalization.
type ConceptInput struct {
	Name     string
	Terms    []string
	Synonyms []string
}

// FieldQuery stores a field-specific query fragment.
type FieldQuery struct {
	Field string
	Query string
}

// WatchedSchedule stores optional watched-search refresh metadata.
type WatchedSchedule struct {
	Enabled  bool
	Interval string
}

// NewStrategy validates and normalizes a saved search strategy.
func NewStrategy(input StrategyInput) (Strategy, error) {
	id := strings.TrimSpace(input.ID)
	title := strings.TrimSpace(input.Title)
	if id == "" {
		return Strategy{}, fmt.Errorf("search strategy id is required")
	}
	if title == "" {
		return Strategy{}, fmt.Errorf("search strategy title is required")
	}
	concepts := normalizeConcepts(input.Concepts)
	if len(concepts) == 0 {
		return Strategy{}, fmt.Errorf("at least one search concept is required")
	}
	return Strategy{
		SchemaVersion: "1",
		ID:            id,
		Title:         title,
		Concepts:      concepts,
		Fields:        normalizeFields(input.Fields),
		Schedule:      WatchedSchedule{Enabled: input.Schedule.Enabled, Interval: strings.TrimSpace(input.Schedule.Interval)},
	}, nil
}

func normalizeConcepts(inputs []ConceptInput) []Concept {
	concepts := make([]Concept, 0, len(inputs))
	for _, input := range inputs {
		concept := Concept{Name: strings.TrimSpace(input.Name), Terms: normalizeStrings(input.Terms), Synonyms: normalizeStrings(input.Synonyms)}
		if concept.Name == "" && len(concept.Terms) == 0 && len(concept.Synonyms) == 0 {
			continue
		}
		concepts = append(concepts, concept)
	}
	return concepts
}

func normalizeFields(inputs []FieldQuery) []FieldQuery {
	fields := make([]FieldQuery, 0, len(inputs))
	for _, input := range inputs {
		field := FieldQuery{Field: strings.TrimSpace(input.Field), Query: strings.TrimSpace(input.Query)}
		if field.Field == "" && field.Query == "" {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

// ProvenanceMetadata returns stable metadata for provenance events that reference this strategy.
func (s Strategy) ProvenanceMetadata() map[string]string {
	return map[string]string{
		"schema_version":   s.SchemaVersion,
		"strategy_id":      s.ID,
		"watched_enabled":  fmt.Sprintf("%t", s.Schedule.Enabled),
		"watched_interval": s.Schedule.Interval,
	}
}

// BooleanQuery renders concepts as OR groups joined by AND plus field-scoped clauses.
func (s Strategy) BooleanQuery() string {
	clauses := []string{}
	for _, concept := range s.Concepts {
		terms := append([]string{}, concept.Terms...)
		terms = append(terms, concept.Synonyms...)
		if len(terms) == 0 {
			continue
		}
		quoted := make([]string, 0, len(terms))
		for _, term := range terms {
			quoted = append(quoted, quoteQueryTerm(term))
		}
		clauses = append(clauses, "("+strings.Join(quoted, " OR ")+")")
	}
	for _, field := range s.Fields {
		if field.Field == "" || field.Query == "" {
			continue
		}
		clauses = append(clauses, field.Field+":"+quoteQueryTerm(field.Query))
	}
	return strings.Join(clauses, " AND ")
}

func quoteQueryTerm(term string) string {
	term = strings.TrimSpace(term)
	if strings.ContainsAny(term, " \t\n\r") {
		return `"` + strings.ReplaceAll(term, `"`, `\"`) + `"`
	}
	return term
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
