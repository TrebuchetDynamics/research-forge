package evidence

import (
	"fmt"
	"sort"
	"strings"
)

type RiskOfBiasJudgment string

const (
	RiskOfBiasLow     RiskOfBiasJudgment = "low"
	RiskOfBiasHigh    RiskOfBiasJudgment = "high"
	RiskOfBiasUnclear RiskOfBiasJudgment = "unclear"
)

type RiskOfBiasSuggestionStatus string

const (
	RiskOfBiasSuggested RiskOfBiasSuggestionStatus = "suggested"
	RiskOfBiasAccepted  RiskOfBiasSuggestionStatus = "accepted"
	RiskOfBiasRejected  RiskOfBiasSuggestionStatus = "rejected"
	RiskOfBiasCorrected RiskOfBiasSuggestionStatus = "corrected"
)

type RiskOfBiasTemplate struct {
	SchemaVersion string  `json:"schemaVersion"`
	Name          string  `json:"name"`
	Domain        string  `json:"domain"`
	Fields        []Field `json:"fields"`
}

type SupportText struct {
	Ref  string `json:"ref"`
	Text string `json:"text"`
}

type RiskOfBiasSuggestionRequest struct {
	PaperID      string        `json:"paperId"`
	Passages     []SupportText `json:"passages"`
	ModelName    string        `json:"modelName"`
	ModelVersion string        `json:"modelVersion"`
}

type RiskOfBiasSuggestionQueue struct {
	SchemaVersion string                 `json:"schemaVersion"`
	PaperID       string                 `json:"paperId"`
	Suggestions   []RiskOfBiasSuggestion `json:"suggestions"`
}

type RiskOfBiasSuggestion struct {
	ID               string                     `json:"id"`
	PaperID          string                     `json:"paperId"`
	Domain           string                     `json:"domain"`
	Judgment         RiskOfBiasJudgment         `json:"judgment"`
	ExactSupportText string                     `json:"exactSupportText"`
	SupportRef       string                     `json:"supportRef"`
	Uncertainty      float64                    `json:"uncertainty"`
	ModelName        string                     `json:"modelName"`
	ModelVersion     string                     `json:"modelVersion"`
	Status           RiskOfBiasSuggestionStatus `json:"status"`
	ReviewerDecision RiskOfBiasReviewerDecision `json:"reviewerDecision,omitempty"`
}

type RiskOfBiasReviewerDecision struct {
	Decision RiskOfBiasSuggestionStatus `json:"decision"`
	Reviewer string                     `json:"reviewer"`
	Note     string                     `json:"note,omitempty"`
}

type RiskOfBiasReviewInput struct {
	SuggestionID string
	Decision     RiskOfBiasSuggestionStatus
	Reviewer     string
	Note         string
}

func DefaultRiskOfBiasSchemaTemplates() []RiskOfBiasTemplate {
	fields := []Field{{Name: "judgment", Type: "low|high|unclear"}, {Name: "supportText", Type: "string"}, {Name: "supportRef", Type: "string"}, {Name: "uncertainty", Type: "number"}, {Name: "reviewerDecision", Type: "accepted|rejected|corrected"}}
	return []RiskOfBiasTemplate{
		{SchemaVersion: "1", Name: "robotreviewer-randomization", Domain: "random_sequence_generation", Fields: append([]Field{}, fields...)},
		{SchemaVersion: "1", Name: "robotreviewer-allocation", Domain: "allocation_concealment", Fields: append([]Field{}, fields...)},
		{SchemaVersion: "1", Name: "robotreviewer-blinding", Domain: "blinding_participants_personnel", Fields: append([]Field{}, fields...)},
		{SchemaVersion: "1", Name: "robotreviewer-incomplete-outcome", Domain: "incomplete_outcome_data", Fields: append([]Field{}, fields...)},
		{SchemaVersion: "1", Name: "robotreviewer-selective-reporting", Domain: "selective_reporting", Fields: append([]Field{}, fields...)},
	}
}

func DraftRiskOfBiasSuggestionQueue(request RiskOfBiasSuggestionRequest) RiskOfBiasSuggestionQueue {
	paperID := strings.TrimSpace(request.PaperID)
	model := strings.TrimSpace(request.ModelName)
	if model == "" {
		model = "robotreviewer-inspired-rules"
	}
	version := strings.TrimSpace(request.ModelVersion)
	if version == "" {
		version = "fixture-v1"
	}
	queue := RiskOfBiasSuggestionQueue{SchemaVersion: "1", PaperID: paperID}
	for _, template := range DefaultRiskOfBiasSchemaTemplates() {
		passage, judgment, uncertainty, ok := matchRiskOfBiasPassage(template.Domain, request.Passages)
		if !ok {
			continue
		}
		queue.Suggestions = append(queue.Suggestions, RiskOfBiasSuggestion{ID: fmt.Sprintf("rob-%s-%d", template.Domain, len(queue.Suggestions)+1), PaperID: paperID, Domain: template.Domain, Judgment: judgment, ExactSupportText: passage.Text, SupportRef: passage.Ref, Uncertainty: uncertainty, ModelName: model, ModelVersion: version, Status: RiskOfBiasSuggested})
	}
	sort.Slice(queue.Suggestions, func(i, j int) bool { return queue.Suggestions[i].ID < queue.Suggestions[j].ID })
	return queue
}

func ReviewRiskOfBiasSuggestion(queue RiskOfBiasSuggestionQueue, input RiskOfBiasReviewInput) (RiskOfBiasSuggestionQueue, error) {
	if strings.TrimSpace(input.SuggestionID) == "" {
		return queue, fmt.Errorf("suggestion id is required")
	}
	if strings.TrimSpace(input.Reviewer) == "" {
		return queue, fmt.Errorf("reviewer is required")
	}
	if input.Decision != RiskOfBiasAccepted && input.Decision != RiskOfBiasRejected && input.Decision != RiskOfBiasCorrected {
		return queue, fmt.Errorf("review decision must be accepted, rejected, or corrected")
	}
	for i := range queue.Suggestions {
		if queue.Suggestions[i].ID == input.SuggestionID {
			queue.Suggestions[i].Status = input.Decision
			queue.Suggestions[i].ReviewerDecision = RiskOfBiasReviewerDecision{Decision: input.Decision, Reviewer: input.Reviewer, Note: input.Note}
			return queue, nil
		}
	}
	return queue, fmt.Errorf("suggestion not found")
}

func matchRiskOfBiasPassage(domain string, passages []SupportText) (SupportText, RiskOfBiasJudgment, float64, bool) {
	keywords := map[string][]string{
		"random_sequence_generation":      {"random", "sequence", "generated"},
		"allocation_concealment":          {"allocation", "sealed", "conceal"},
		"blinding_participants_personnel": {"blind", "blinded", "mask"},
		"incomplete_outcome_data":         {"withdraw", "lost to follow", "attrition", "missing"},
		"selective_reporting":             {"protocol", "registered", "reported outcomes"},
	}
	for _, passage := range passages {
		text := strings.ToLower(passage.Text)
		for _, keyword := range keywords[domain] {
			if strings.Contains(text, keyword) {
				judgment := RiskOfBiasUnclear
				uncertainty := 0.45
				if strings.Contains(text, "not blinded") || strings.Contains(text, "no blinding") || strings.Contains(text, "not conceal") {
					judgment = RiskOfBiasHigh
					uncertainty = 0.2
				} else if strings.Contains(text, "computer generated") || strings.Contains(text, "sealed") || strings.Contains(text, "double blind") || strings.Contains(text, "registered") {
					judgment = RiskOfBiasLow
					uncertainty = 0.25
				}
				return passage, judgment, uncertainty, true
			}
		}
	}
	return SupportText{}, RiskOfBiasUnclear, 1, false
}
