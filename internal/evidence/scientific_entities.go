package evidence

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type EntitySuggestionStatus string

const (
	EntitySuggested EntitySuggestionStatus = "suggested"
	EntityAccepted  EntitySuggestionStatus = "accepted"
	EntityRejected  EntitySuggestionStatus = "rejected"
	EntityCorrected EntitySuggestionStatus = "corrected"
)

type ScientificEntitySuggestionRequest struct {
	PaperID      string            `json:"paperId"`
	Passages     []parsing.Passage `json:"passages"`
	ModelName    string            `json:"modelName"`
	ModelVersion string            `json:"modelVersion"`
}

type ScientificEntitySuggestionQueue struct {
	SchemaVersion string                       `json:"schemaVersion"`
	PaperID       string                       `json:"paperId"`
	Suggestions   []ScientificEntitySuggestion `json:"suggestions"`
}

type ScientificEntitySuggestion struct {
	ID                   string                   `json:"id"`
	PaperID              string                   `json:"paperId"`
	PassageID            string                   `json:"passageId"`
	Mention              string                   `json:"mention"`
	Offset               parsing.TextOffset       `json:"offset"`
	Abbreviation         AbbreviationResolution   `json:"abbreviation,omitempty"`
	EntityLinkCandidates []EntityLinkCandidate    `json:"entityLinkCandidates"`
	Confidence           float64                  `json:"confidence"`
	ModelName            string                   `json:"modelName"`
	ModelVersion         string                   `json:"modelVersion"`
	Status               EntitySuggestionStatus   `json:"status"`
	ReviewerDecision     ScientificEntityDecision `json:"reviewerDecision,omitempty"`
}

type AbbreviationResolution struct {
	ShortForm string `json:"shortForm,omitempty"`
	LongForm  string `json:"longForm,omitempty"`
}

type EntityLinkCandidate struct {
	Source     string  `json:"source"`
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

type ScientificEntityDecision struct {
	Decision EntitySuggestionStatus `json:"decision"`
	Reviewer string                 `json:"reviewer"`
	Note     string                 `json:"note,omitempty"`
}

type ScientificEntityReviewInput struct {
	SuggestionID string
	Decision     EntitySuggestionStatus
	Reviewer     string
	Note         string
}

var abbreviationPattern = regexp.MustCompile(`([A-Z][A-Za-z][A-Za-z\- ]{2,80})\s+\(([A-Z][A-Z0-9]{1,10})\)`)

func DraftScientificEntitySuggestions(request ScientificEntitySuggestionRequest) ScientificEntitySuggestionQueue {
	model := strings.TrimSpace(request.ModelName)
	if model == "" {
		model = "scispacy-inspired-rules"
	}
	version := strings.TrimSpace(request.ModelVersion)
	if version == "" {
		version = "fixture-v1"
	}
	queue := ScientificEntitySuggestionQueue{SchemaVersion: "1", PaperID: strings.TrimSpace(request.PaperID)}
	seen := map[string]bool{}
	for _, passage := range request.Passages {
		for _, match := range abbreviationPattern.FindAllStringSubmatchIndex(passage.Text, -1) {
			longForm := strings.TrimSpace(passage.Text[match[2]:match[3]])
			shortForm := strings.TrimSpace(passage.Text[match[4]:match[5]])
			start := passage.Offset.Start + match[4]
			end := passage.Offset.Start + match[5]
			addEntitySuggestion(&queue, seen, request.PaperID, passage.ID, shortForm, parsing.TextOffset{Start: start, End: end}, AbbreviationResolution{ShortForm: shortForm, LongForm: longForm}, model, version, 0.92)
		}
		for _, mention := range capitalizedMentions(passage.Text) {
			start := passage.Offset.Start + mention.start
			end := passage.Offset.Start + mention.end
			addEntitySuggestion(&queue, seen, request.PaperID, passage.ID, mention.text, parsing.TextOffset{Start: start, End: end}, AbbreviationResolution{}, model, version, 0.72)
		}
	}
	sort.Slice(queue.Suggestions, func(i, j int) bool { return queue.Suggestions[i].ID < queue.Suggestions[j].ID })
	return queue
}

func ReviewScientificEntitySuggestion(queue ScientificEntitySuggestionQueue, input ScientificEntityReviewInput) (ScientificEntitySuggestionQueue, error) {
	if strings.TrimSpace(input.SuggestionID) == "" {
		return queue, fmt.Errorf("suggestion id is required")
	}
	if strings.TrimSpace(input.Reviewer) == "" {
		return queue, fmt.Errorf("reviewer is required")
	}
	if input.Decision != EntityAccepted && input.Decision != EntityRejected && input.Decision != EntityCorrected {
		return queue, fmt.Errorf("review decision must be accepted, rejected, or corrected")
	}
	for i := range queue.Suggestions {
		if queue.Suggestions[i].ID == input.SuggestionID {
			queue.Suggestions[i].Status = input.Decision
			queue.Suggestions[i].ReviewerDecision = ScientificEntityDecision{Decision: input.Decision, Reviewer: input.Reviewer, Note: input.Note}
			return queue, nil
		}
	}
	return queue, fmt.Errorf("suggestion not found")
}

func addEntitySuggestion(queue *ScientificEntitySuggestionQueue, seen map[string]bool, paperID, passageID, mention string, offset parsing.TextOffset, abbreviation AbbreviationResolution, model, version string, confidence float64) {
	mention = strings.TrimSpace(mention)
	if mention == "" || len(mention) < 2 {
		return
	}
	key := passageID + "\x00" + mention + "\x00" + fmt.Sprint(offset.Start)
	if seen[key] {
		return
	}
	seen[key] = true
	id := fmt.Sprintf("entity-%s-%d", safeEntityID(passageID), len(queue.Suggestions)+1)
	queue.Suggestions = append(queue.Suggestions, ScientificEntitySuggestion{ID: id, PaperID: paperID, PassageID: passageID, Mention: mention, Offset: offset, Abbreviation: abbreviation, EntityLinkCandidates: entityCandidates(mention, confidence), Confidence: confidence, ModelName: model, ModelVersion: version, Status: EntitySuggested})
}

type entityMention struct {
	text       string
	start, end int
}

func capitalizedMentions(text string) []entityMention {
	mentions := []entityMention{}
	start := -1
	for i, r := range text {
		if unicode.IsLetter(r) && unicode.IsUpper(r) {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			if i-start > 2 {
				mentions = append(mentions, entityMention{text: text[start:i], start: start, end: i})
			}
			start = -1
		}
	}
	if start >= 0 && len(text)-start > 2 {
		mentions = append(mentions, entityMention{text: text[start:], start: start, end: len(text)})
	}
	return mentions
}

func entityCandidates(mention string, confidence float64) []EntityLinkCandidate {
	id := strings.ToUpper(strings.NewReplacer(" ", "_", "-", "_").Replace(mention))
	return []EntityLinkCandidate{{Source: "local-scientific-entity-index", ID: "LOCAL:" + id, Label: mention, Confidence: confidence}}
}

func safeEntityID(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "passage"
	}
	return out
}
