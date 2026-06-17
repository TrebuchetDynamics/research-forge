package protocol

import (
	"fmt"
	"sort"
	"strings"
)

type Framework string

const (
	FrameworkPICO     Framework = "pico"
	FrameworkPECO     Framework = "peco"
	FrameworkSPIDER   Framework = "spider"
	FrameworkFreeform Framework = "freeform"
)

type QuestionInput struct {
	Framework    string
	Question     string
	Population   string
	Intervention string
	Comparator   string
	Outcome      string
	Exposure     string
	Sample       string
	Phenomenon   string
	Design       string
	Evaluation   string
	ResearchType string
}

type CompiledQuestionPlan struct {
	SchemaVersion           string                     `json:"schemaVersion"`
	Framework               Framework                  `json:"framework"`
	Question                string                     `json:"question"`
	Components              map[string]string          `json:"components"`
	SourceQueries           map[string]SourceQueryPlan `json:"sourceQueries"`
	InclusionCriteria       []string                   `json:"inclusionCriteria"`
	ExclusionCriteria       []string                   `json:"exclusionCriteria"`
	ExtractionSchema        ExtractionSchemaSeed       `json:"extractionSchema"`
	ReviewerPrompts         []ReviewerPrompt         `json:"reviewerPrompts"`
	ReviewerApprovalRequired bool                       `json:"reviewerApprovalRequired"`
	AutoAcceptedClaims      bool                       `json:"autoAcceptedClaims"`
	Warnings                []string                   `json:"warnings,omitempty"`
}

type SourceQueryPlan struct {
	Source string `json:"source"`
	Query  string `json:"query"`
	Note   string `json:"note"`
}

type ExtractionSchemaSeed struct {
	Name   string            `json:"name"`
	Fields []ExtractionField `json:"fields"`
}

type ExtractionField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type ReviewerPrompt struct {
	ID            string `json:"id"`
	Text          string `json:"text"`
	ProvenanceTag string `json:"provenanceTag"`
	Accepted      bool   `json:"accepted"`
}

func CompileQuestion(input QuestionInput) (CompiledQuestionPlan, error) {
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return CompiledQuestionPlan{}, fmt.Errorf("research question is required")
	}
	framework := normalizeFramework(input.Framework)
	components := componentsFor(framework, input, question)
	terms := queryTerms(components, question)
	query := booleanQuery(terms)
	if query == "" {
		query = quoteTerm(question)
	}
	queries := map[string]SourceQueryPlan{}
	for _, source := range []string{"openalex", "semantic-scholar", "crossref", "arxiv", "pubmed", "europepmc"} {
		queries[source] = SourceQueryPlan{Source: source, Query: query, Note: "draft query; reviewer approval required before source execution"}
	}
	queries["unpaywall"] = SourceQueryPlan{Source: "unpaywall", Query: "DOI lookup for included/imported records", Note: "not a discovery query; use after DOI-bearing records exist"}
	return CompiledQuestionPlan{
		SchemaVersion:           "1",
		Framework:               framework,
		Question:                question,
		Components:              components,
		SourceQueries:           queries,
		InclusionCriteria:       inclusionCriteria(framework, components),
		ExclusionCriteria:       exclusionCriteria(framework, components),
		ExtractionSchema:        extractionSchema(framework),
		ReviewerPrompts:         reviewerPrompts(framework),
		ReviewerApprovalRequired: true,
		AutoAcceptedClaims:      false,
		Warnings:                warningsFor(framework, components),
	}, nil
}

func normalizeFramework(value string) Framework {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pico":
		return FrameworkPICO
	case "peco":
		return FrameworkPECO
	case "spider":
		return FrameworkSPIDER
	default:
		return FrameworkFreeform
	}
}

func componentsFor(framework Framework, input QuestionInput, question string) map[string]string {
	components := map[string]string{"question": question}
	add := func(key, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			components[key] = value
		}
	}
	switch framework {
	case FrameworkPICO:
		add("population", input.Population)
		add("intervention", input.Intervention)
		add("comparator", input.Comparator)
		add("outcome", input.Outcome)
	case FrameworkPECO:
		add("population", input.Population)
		add("exposure", input.Exposure)
		add("comparator", input.Comparator)
		add("outcome", input.Outcome)
	case FrameworkSPIDER:
		add("sample", input.Sample)
		add("phenomenon", input.Phenomenon)
		add("design", input.Design)
		add("evaluation", input.Evaluation)
		add("research_type", input.ResearchType)
	default:
		for _, term := range splitQuestionTerms(question) {
			if term != "" {
				components["topic"] = strings.TrimSpace(components["topic"] + " " + term)
			}
		}
	}
	return components
}

func queryTerms(components map[string]string, question string) []string {
	priority := []string{"population", "intervention", "exposure", "phenomenon", "outcome", "comparator", "sample", "design", "evaluation", "topic"}
	terms := []string{}
	for _, key := range priority {
		if value := strings.TrimSpace(components[key]); value != "" {
			terms = append(terms, value)
		}
	}
	if len(terms) == 0 {
		terms = splitQuestionTerms(question)
	}
	return dedupeStrings(terms)
}

func booleanQuery(terms []string) string {
	quoted := []string{}
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term != "" {
			quoted = append(quoted, quoteTerm(term))
		}
	}
	return strings.Join(quoted, " AND ")
}

func quoteTerm(term string) string {
	term = strings.TrimSpace(term)
	term = strings.ReplaceAll(term, `"`, `\"`)
	if strings.ContainsAny(term, " \t\n\r") {
		return `"` + term + `"`
	}
	return term
}

func inclusionCriteria(framework Framework, components map[string]string) []string {
	criteria := []string{"Record is relevant to the research question."}
	keys := componentKeys(framework)
	for _, key := range keys {
		if value := strings.TrimSpace(components[key]); value != "" {
			criteria = append(criteria, fmt.Sprintf("Study includes %s: %s.", strings.ReplaceAll(key, "_", " "), value))
		}
	}
	criteria = append(criteria, "Full-text or metadata provides enough source support for reviewer screening.")
	return criteria
}

func exclusionCriteria(framework Framework, components map[string]string) []string {
	criteria := []string{"Record is outside the approved research question or source plan."}
	for _, key := range componentKeys(framework) {
		if strings.TrimSpace(components[key]) != "" {
			criteria = append(criteria, fmt.Sprintf("Study does not address the specified %s.", strings.ReplaceAll(key, "_", " ")))
		}
	}
	criteria = append(criteria, "Record lacks enough source support for the current screening stage.")
	return criteria
}

func extractionSchema(framework Framework) ExtractionSchemaSeed {
	fields := []ExtractionField{{Name: "paper_id", Type: "string", Required: true}, {Name: "finding", Type: "string", Required: true}, {Name: "support_ref", Type: "string", Required: true}}
	switch framework {
	case FrameworkPICO:
		fields = append(fields,
			ExtractionField{Name: "population", Type: "string", Required: true},
			ExtractionField{Name: "intervention", Type: "string", Required: true},
			ExtractionField{Name: "comparator", Type: "string", Required: false},
			ExtractionField{Name: "outcome", Type: "string", Required: true},
			ExtractionField{Name: "effect_size", Type: "number", Required: false},
		)
	case FrameworkPECO:
		fields = append(fields,
			ExtractionField{Name: "population", Type: "string", Required: true},
			ExtractionField{Name: "exposure", Type: "string", Required: true},
			ExtractionField{Name: "comparator", Type: "string", Required: false},
			ExtractionField{Name: "outcome", Type: "string", Required: true},
			ExtractionField{Name: "effect_size", Type: "number", Required: false},
		)
	case FrameworkSPIDER:
		fields = append(fields,
			ExtractionField{Name: "sample", Type: "string", Required: false},
			ExtractionField{Name: "phenomenon", Type: "string", Required: true},
			ExtractionField{Name: "design", Type: "string", Required: false},
			ExtractionField{Name: "evaluation", Type: "string", Required: false},
			ExtractionField{Name: "research_type", Type: "string", Required: false},
		)
	}
	return ExtractionSchemaSeed{Name: string(framework) + "_draft_extraction", Fields: fields}
}

func reviewerPrompts(framework Framework) []ReviewerPrompt {
	texts := []string{
		"Review and edit the drafted source queries before any live source execution.",
		"Confirm inclusion and exclusion criteria before screening decisions are treated as final.",
		"Confirm the extraction schema fields map to source-supported evidence needed for the review.",
		fmt.Sprintf("Check whether the %s framework is the right structure for this question.", framework),
	}
	prompts := make([]ReviewerPrompt, 0, len(texts))
	for i, text := range texts {
		prompts = append(prompts, ReviewerPrompt{ID: fmt.Sprintf("prompt-%02d", i+1), Text: text, ProvenanceTag: "protocol.plan.created", Accepted: false})
	}
	return prompts
}

func warningsFor(framework Framework, components map[string]string) []string {
	warnings := []string{"Draft only: no source query, criterion, prompt, or extraction field is accepted until reviewer approval."}
	for _, key := range componentKeys(framework) {
		if strings.TrimSpace(components[key]) == "" {
			warnings = append(warnings, "missing "+strings.ReplaceAll(key, "_", " ")+" component")
		}
	}
	return warnings
}

func componentKeys(framework Framework) []string {
	switch framework {
	case FrameworkPICO:
		return []string{"population", "intervention", "comparator", "outcome"}
	case FrameworkPECO:
		return []string{"population", "exposure", "comparator", "outcome"}
	case FrameworkSPIDER:
		return []string{"sample", "phenomenon", "design", "evaluation", "research_type"}
	default:
		return []string{"topic"}
	}
}

func splitQuestionTerms(question string) []string {
	replacer := strings.NewReplacer("?", " ", ",", " ", ";", " ", ":", " ", "(", " ", ")", " ")
	words := strings.Fields(replacer.Replace(strings.ToLower(question)))
	stop := map[string]bool{"a": true, "an": true, "and": true, "are": true, "do": true, "does": true, "for": true, "how": true, "in": true, "is": true, "of": true, "or": true, "the": true, "to": true, "what": true, "which": true, "with": true}
	terms := []string{}
	for _, word := range words {
		word = strings.Trim(word, `"'`)
		if len(word) > 2 && !stop[word] {
			terms = append(terms, word)
		}
	}
	if len(terms) > 6 {
		terms = terms[:6]
	}
	return terms
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

func (p CompiledQuestionPlan) SortedSourceQueries() []SourceQueryPlan {
	keys := make([]string, 0, len(p.SourceQueries))
	for key := range p.SourceQueries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]SourceQueryPlan, 0, len(keys))
	for _, key := range keys {
		out = append(out, p.SourceQueries[key])
	}
	return out
}
