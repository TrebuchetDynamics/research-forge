package protocol

import (
	"fmt"
	"strings"
)

type SuggestionAssistant string

const (
	AssistantKeyBERT  SuggestionAssistant = "keybert"
	AssistantSciSpaCy SuggestionAssistant = "scispacy"
	AssistantLLM      SuggestionAssistant = "llm"
)

type SourceTextLink struct {
	ID         string `json:"id"`
	PaperID    string `json:"paperId,omitempty"`
	PassageRef string `json:"passageRef,omitempty"`
	Text       string `json:"text,omitempty"`
}

type QueryExpansionInput struct {
	Question    string           `json:"question"`
	SourceTexts []SourceTextLink `json:"sourceTexts"`
}

type QueryExpansionSuggestion struct {
	ID                       string              `json:"id"`
	Assistant                SuggestionAssistant `json:"assistant"`
	SuggestedTerm            string              `json:"suggestedTerm"`
	Rationale                string              `json:"rationale"`
	SourceTextLinks          []SourceTextLink    `json:"sourceTextLinks"`
	ProvenanceTag            string              `json:"provenanceTag"`
	ReviewerApprovalRequired bool                `json:"reviewerApprovalRequired"`
	ReviewerApproved         bool                `json:"reviewerApproved"`
}

func DraftQueryExpansionSuggestions(input QueryExpansionInput) ([]QueryExpansionSuggestion, error) {
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return nil, fmt.Errorf("question is required")
	}
	links := validSourceTextLinks(input.SourceTexts)
	if len(links) == 0 {
		return nil, fmt.Errorf("at least one source text link is required")
	}
	terms := candidateExpansionTerms(question, links[0].Text)
	if len(terms) < 3 {
		terms = append(terms, "related concept", "scientific entity", "reviewer synonym")
	}
	assistants := []SuggestionAssistant{AssistantKeyBERT, AssistantSciSpaCy, AssistantLLM}
	records := make([]QueryExpansionSuggestion, 0, len(assistants))
	for i, assistant := range assistants {
		records = append(records, QueryExpansionSuggestion{
			ID:                       fmt.Sprintf("qe-%02d", i+1),
			Assistant:                assistant,
			SuggestedTerm:            terms[i%len(terms)],
			Rationale:                rationaleForAssistant(assistant),
			SourceTextLinks:          []SourceTextLink{links[0]},
			ProvenanceTag:            "protocol.query_expansion.suggested",
			ReviewerApprovalRequired: true,
			ReviewerApproved:         false,
		})
	}
	return records, nil
}

func ApplyApprovedQueryExpansions(plan SourcePlan, suggestions []QueryExpansionSuggestion) (SourcePlan, error) {
	terms := []string{}
	for _, suggestion := range suggestions {
		term := strings.TrimSpace(suggestion.SuggestedTerm)
		if term == "" {
			continue
		}
		if len(validSourceTextLinks(suggestion.SourceTextLinks)) == 0 {
			return SourcePlan{}, fmt.Errorf("query expansion %s missing source text link", suggestion.ID)
		}
		if suggestion.ReviewerApprovalRequired && !suggestion.ReviewerApproved {
			return SourcePlan{}, fmt.Errorf("query expansion %s requires reviewer approval before source plan changes", suggestion.ID)
		}
		terms = append(terms, term)
	}
	if len(terms) == 0 {
		return plan, nil
	}
	updated := plan
	updated.Warnings = append(append([]string{}, plan.Warnings...), "Source plan includes reviewer-approved query expansion terms; preserve suggestion provenance.")
	updated.Sources = append([]SourcePlanEntry{}, plan.Sources...)
	for i := range updated.Sources {
		if updated.Sources[i].Query == "" {
			continue
		}
		for _, term := range dedupeStrings(terms) {
			quoted := quoteTerm(term)
			if !strings.Contains(updated.Sources[i].Query, quoted) && !strings.Contains(updated.Sources[i].Query, term) {
				updated.Sources[i].Query += " AND " + quoted
			}
		}
	}
	return updated, nil
}

func validSourceTextLinks(links []SourceTextLink) []SourceTextLink {
	valid := []SourceTextLink{}
	for _, link := range links {
		if strings.TrimSpace(link.ID) != "" && strings.TrimSpace(link.Text) != "" {
			valid = append(valid, link)
		}
	}
	return valid
}

func candidateExpansionTerms(question, text string) []string {
	terms := []string{}
	seen := map[string]bool{}
	for _, value := range []string{text, question} {
		for _, token := range splitQuestionTerms(value) {
			token = strings.Trim(strings.ToLower(token), `.,;:()[]{}"'`)
			if len(token) < 5 || seen[token] {
				continue
			}
			seen[token] = true
			terms = append(terms, token)
		}
	}
	return terms
}

func rationaleForAssistant(assistant SuggestionAssistant) string {
	switch assistant {
	case AssistantKeyBERT:
		return "keyword candidate extracted from linked source text"
	case AssistantSciSpaCy:
		return "scientific entity candidate extracted from linked source text"
	default:
		return "assistant-suggested synonym grounded in linked source text"
	}
}
