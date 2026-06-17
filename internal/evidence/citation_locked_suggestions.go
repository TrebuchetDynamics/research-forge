package evidence

import (
	"fmt"
	"strings"
)

type CitationLockedSuggestionKind string

const (
	CitationLockedQueryExpansion     CitationLockedSuggestionKind = "query_expansion"
	CitationLockedScreeningRationale CitationLockedSuggestionKind = "screening_rationale"
	CitationLockedExtraction         CitationLockedSuggestionKind = "extraction"
	CitationLockedReportProse        CitationLockedSuggestionKind = "report_prose"
)

type CitationLockedSupport struct {
	Ref       string `json:"ref"`
	ExactText string `json:"exactText"`
}

type CitationLockedSuggestionRequest struct {
	PaperID      string                       `json:"paperId"`
	Kind         CitationLockedSuggestionKind `json:"kind"`
	Prompt       string                       `json:"prompt"`
	Supports     []CitationLockedSupport      `json:"supports"`
	ModelName    string                       `json:"modelName"`
	ModelVersion string                       `json:"modelVersion"`
}

type CitationLockedSuggestionQueue struct {
	SchemaVersion string                     `json:"schemaVersion"`
	PaperID       string                     `json:"paperId"`
	Suggestions   []CitationLockedSuggestion `json:"suggestions"`
}

type CitationLockedSuggestion struct {
	ID               string                       `json:"id"`
	PaperID          string                       `json:"paperId"`
	Kind             CitationLockedSuggestionKind `json:"kind"`
	Prompt           string                       `json:"prompt"`
	SuggestedText    string                       `json:"suggestedText"`
	CitationLocks    []CitationLockedSupport      `json:"citationLocks"`
	ModelName        string                       `json:"modelName"`
	ModelVersion     string                       `json:"modelVersion"`
	Status           Status                       `json:"status"`
	ReviewerDecision CitationLockedDecision       `json:"reviewerDecision,omitempty"`
}

type CitationLockedDecision struct {
	Decision Status `json:"decision"`
	Reviewer string `json:"reviewer"`
	Note     string `json:"note,omitempty"`
}

type CitationLockedReviewInput struct {
	SuggestionID string
	Decision     Status
	Reviewer     string
	Note         string
}

func DraftCitationLockedSuggestions(request CitationLockedSuggestionRequest) (CitationLockedSuggestionQueue, error) {
	paperID := strings.TrimSpace(request.PaperID)
	if paperID == "" {
		return CitationLockedSuggestionQueue{}, fmt.Errorf("paper id is required")
	}
	kind := request.Kind
	if kind == "" {
		kind = CitationLockedExtraction
	}
	if !supportedCitationLockedKind(kind) {
		return CitationLockedSuggestionQueue{}, fmt.Errorf("unsupported citation-locked suggestion kind")
	}
	locks := validCitationLocks(request.Supports)
	if len(locks) == 0 {
		return CitationLockedSuggestionQueue{}, fmt.Errorf("citation-locked suggestions require exact support text")
	}
	model := strings.TrimSpace(request.ModelName)
	if model == "" {
		model = "citation-locked-llm-fixture"
	}
	version := strings.TrimSpace(request.ModelVersion)
	if version == "" {
		version = "fixture-v1"
	}
	prompt := strings.TrimSpace(request.Prompt)
	text := citationLockedSuggestedText(kind, prompt, locks)
	return CitationLockedSuggestionQueue{SchemaVersion: "1", PaperID: paperID, Suggestions: []CitationLockedSuggestion{{ID: "citation-locked-1", PaperID: paperID, Kind: kind, Prompt: prompt, SuggestedText: text, CitationLocks: locks, ModelName: model, ModelVersion: version, Status: StatusSuggested}}}, nil
}

func ReviewCitationLockedSuggestion(queue CitationLockedSuggestionQueue, input CitationLockedReviewInput) (CitationLockedSuggestionQueue, error) {
	if strings.TrimSpace(input.SuggestionID) == "" {
		return queue, fmt.Errorf("suggestion id is required")
	}
	if strings.TrimSpace(input.Reviewer) == "" {
		return queue, fmt.Errorf("reviewer is required")
	}
	if input.Decision != StatusAccepted && input.Decision != StatusRejected && input.Decision != StatusCorrected {
		return queue, fmt.Errorf("review decision must be accepted, rejected, or corrected")
	}
	for i := range queue.Suggestions {
		if queue.Suggestions[i].ID == input.SuggestionID {
			queue.Suggestions[i].Status = input.Decision
			queue.Suggestions[i].ReviewerDecision = CitationLockedDecision{Decision: input.Decision, Reviewer: input.Reviewer, Note: input.Note}
			return queue, nil
		}
	}
	return queue, fmt.Errorf("suggestion not found")
}

func supportedCitationLockedKind(kind CitationLockedSuggestionKind) bool {
	switch kind {
	case CitationLockedQueryExpansion, CitationLockedScreeningRationale, CitationLockedExtraction, CitationLockedReportProse:
		return true
	default:
		return false
	}
}

func (s CitationLockedSuggestion) AcceptWithoutReview() error {
	return fmt.Errorf("citation-locked LLM suggestions require reviewer approval before acceptance")
}

func validCitationLocks(supports []CitationLockedSupport) []CitationLockedSupport {
	locks := []CitationLockedSupport{}
	for _, support := range supports {
		ref := strings.TrimSpace(support.Ref)
		text := strings.TrimSpace(support.ExactText)
		if ref == "" || text == "" {
			continue
		}
		locks = append(locks, CitationLockedSupport{Ref: ref, ExactText: text})
	}
	return locks
}

func EverySuggestedSentenceCitationLocked(s CitationLockedSuggestion) bool {
	if s.Status != StatusSuggested || len(s.CitationLocks) == 0 {
		return false
	}
	for _, sentence := range strings.Split(s.SuggestedText, "\n") {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		locked := false
		for _, lock := range s.CitationLocks {
			if strings.Contains(sentence, "["+lock.Ref+"]") && strings.TrimSpace(lock.ExactText) != "" {
				locked = true
				break
			}
		}
		if !locked {
			return false
		}
	}
	return true
}

func citationLockedSuggestedText(kind CitationLockedSuggestionKind, prompt string, locks []CitationLockedSupport) string {
	prefix := map[CitationLockedSuggestionKind]string{CitationLockedQueryExpansion: "Query expansion", CitationLockedScreeningRationale: "Screening rationale", CitationLockedExtraction: "Extraction candidate", CitationLockedReportProse: "Report prose"}[kind]
	lines := []string{}
	for _, lock := range locks {
		basis := lock.ExactText
		if len(basis) > 160 {
			basis = basis[:160]
		}
		if prompt != "" {
			lines = append(lines, fmt.Sprintf("%s for %q: %s [%s]", prefix, prompt, basis, lock.Ref))
		} else {
			lines = append(lines, fmt.Sprintf("%s: %s [%s]", prefix, basis, lock.Ref))
		}
	}
	return strings.Join(lines, "\n")
}
