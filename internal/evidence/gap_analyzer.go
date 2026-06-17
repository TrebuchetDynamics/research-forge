package evidence

import "strings"

type EvidenceGapAnalysisInput struct {
	ResearchQuestion          string
	ScreenedInPaperIDs        []string
	ParsedPassagePaperIDs     []string
	Items                     []EvidenceItem
	Claims                    []CitationLockedSuggestion
	RequiredOutcomes          []string
	RequiredComparators       []string
	FullTextRequiredPaperIDs  []string
	AvailableFullTextPaperIDs []string
	AnalysisIncludedPaperIDs  []string
}

type EvidenceGapReport struct {
	SchemaVersion    string        `json:"schemaVersion"`
	ReadyForAnalysis bool          `json:"readyForAnalysis"`
	Gaps             []EvidenceGap `json:"gaps"`
}

type EvidenceGap struct {
	Code    string `json:"code"`
	PaperID string `json:"paperId,omitempty"`
	Field   string `json:"field,omitempty"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

func AnalyzeEvidenceGaps(input EvidenceGapAnalysisInput) EvidenceGapReport {
	report := EvidenceGapReport{SchemaVersion: "1", ReadyForAnalysis: true}
	accepted := acceptedEvidence(input.Items)
	outcomes := valuesByField(accepted, "outcome")
	comparators := valuesByField(accepted, "comparator")
	acceptedPaperIDs := map[string]bool{}
	for _, item := range accepted {
		acceptedPaperIDs[item.PaperID] = true
	}
	parsedPassages := set(input.ParsedPassagePaperIDs)
	for _, paperID := range input.ScreenedInPaperIDs {
		paperID = strings.TrimSpace(paperID)
		if paperID == "" {
			continue
		}
		if !acceptedPaperIDs[paperID] {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "screened_in_missing_evidence", PaperID: paperID, Message: "screened-in study has no accepted extracted evidence"})
		}
		if len(parsedPassages) > 0 && !parsedPassages[paperID] {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "screened_in_missing_parsed_passages", PaperID: paperID, Message: "screened-in study has no parsed passages for evidence support"})
		}
	}
	for _, term := range researchQuestionTerms(input.ResearchQuestion) {
		if !containsTermInEvidence(accepted, term) {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "question_term_missing_evidence", Value: term, Message: "research-question term is not represented in accepted evidence values"})
		}
	}
	for _, outcome := range input.RequiredOutcomes {
		if !containsFold(outcomes, outcome) {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "missing_outcome", Field: "outcome", Value: strings.TrimSpace(outcome), Message: "required outcome has no accepted evidence"})
		}
	}
	for _, comparator := range input.RequiredComparators {
		if !containsFold(comparators, comparator) {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "missing_comparator", Field: "comparator", Value: strings.TrimSpace(comparator), Message: "required comparator has no accepted evidence"})
		}
	}
	for _, item := range accepted {
		if item.Support.Kind == "" || strings.TrimSpace(item.Support.Ref) == "" {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "unsupported_claim", PaperID: item.PaperID, Message: "accepted evidence is missing source support"})
		}
	}
	for _, claim := range input.Claims {
		if claim.Status != StatusAccepted || len(validCitationLocks(claim.CitationLocks)) == 0 {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "unsupported_claim", PaperID: claim.PaperID, Value: claim.ID, Message: "claim/prose suggestion is not reviewer-accepted with citation locks"})
		}
	}
	availableFullText := set(input.AvailableFullTextPaperIDs)
	for _, paperID := range input.FullTextRequiredPaperIDs {
		paperID = strings.TrimSpace(paperID)
		if paperID != "" && !availableFullText[paperID] {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "incomplete_full_text", PaperID: paperID, Message: "required full text has not been acquired/imported"})
		}
	}
	analysisIncluded := set(input.AnalysisIncludedPaperIDs)
	for paperID := range acceptedPaperIDs {
		if !analysisIncluded[paperID] {
			report.Gaps = append(report.Gaps, EvidenceGap{Code: "analysis_input_not_ready", PaperID: paperID, Message: "accepted evidence is not present in analysis inputs"})
		}
	}
	if len(report.Gaps) > 0 {
		report.ReadyForAnalysis = false
	}
	return report
}

func acceptedEvidence(items []EvidenceItem) []EvidenceItem {
	out := []EvidenceItem{}
	for _, item := range items {
		if item.Status == StatusAccepted {
			out = append(out, item)
		}
	}
	return out
}

func valuesByField(items []EvidenceItem, field string) []string {
	values := []string{}
	for _, item := range items {
		if value := strings.TrimSpace(item.Values[field]); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func containsFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return true
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func researchQuestionTerms(question string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, token := range strings.FieldsFunc(strings.ToLower(question), func(r rune) bool { return r < 'a' || r > 'z' }) {
		if len(token) < 6 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func containsTermInEvidence(items []EvidenceItem, term string) bool {
	for _, item := range items {
		for _, value := range item.Values {
			if strings.Contains(strings.ToLower(value), term) {
				return true
			}
		}
	}
	return false
}

func set(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}
