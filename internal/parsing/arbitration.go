package parsing

import "strings"

type ParserArbitrationReport struct {
	SchemaVersion         string                       `json:"schemaVersion"`
	PaperID               string                       `json:"paperId"`
	FieldScores           map[string][]FieldScore      `json:"fieldScores"`
	Comparisons           []FieldComparison            `json:"comparisons"`
	WarningComparison     []ParserWarnings             `json:"warningComparison"`
	ReconciliationOutputs []ParserReconciliationOutput `json:"reconciliationOutputs"`
	Decision              ParserArbitrationDecision    `json:"decision"`
}

type FieldScore struct {
	ParserName string  `json:"parserName"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

type FieldComparison struct {
	Field        string             `json:"field"`
	ParserValues []ParserFieldValue `json:"parserValues"`
}

type ParserFieldValue struct {
	ParserName string     `json:"parserName"`
	RawText    string     `json:"rawText"`
	Offset     TextOffset `json:"offset"`
}

type TextOffset struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type ParserWarnings struct {
	ParserName string   `json:"parserName"`
	Warnings   []string `json:"warnings"`
}

type ParserArbitrationDecision struct {
	AcceptedParser           string `json:"acceptedParser"`
	Reason                   string `json:"reason"`
	Reviewer                 string `json:"reviewer,omitempty"`
	ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
}

type ArbitrationDecisionInput struct {
	AcceptedParser string
	Reason         string
	Reviewer       string
}

func ArbitrateParserOutputs(docs []ParsedDocument, input ArbitrationDecisionInput) ParserArbitrationReport {
	report := ParserArbitrationReport{SchemaVersion: "1", FieldScores: map[string][]FieldScore{}, Decision: ParserArbitrationDecision{AcceptedParser: strings.TrimSpace(input.AcceptedParser), Reason: strings.TrimSpace(input.Reason), Reviewer: strings.TrimSpace(input.Reviewer), ReviewerApprovalRequired: true}}
	if len(docs) == 0 {
		return report
	}
	report.PaperID = docs[0].PaperID
	fields := []string{"title", "abstract", "sections", "references"}
	for _, field := range fields {
		comparison := FieldComparison{Field: field}
		for _, doc := range docs {
			raw := fieldRawText(doc, field)
			report.FieldScores[field] = append(report.FieldScores[field], FieldScore{ParserName: doc.ParserName, Score: scoreField(doc, field, raw), Reason: fieldScoreReason(field, raw, doc.Warnings)})
			comparison.ParserValues = append(comparison.ParserValues, ParserFieldValue{ParserName: doc.ParserName, RawText: raw, Offset: TextOffset{Start: 0, End: len(raw)}})
		}
		report.Comparisons = append(report.Comparisons, comparison)
	}
	for _, doc := range docs {
		report.WarningComparison = append(report.WarningComparison, ParserWarnings{ParserName: doc.ParserName, Warnings: append([]string{}, doc.Warnings...)})
	}
	if report.Decision.AcceptedParser == "" {
		report.Decision.AcceptedParser = bestParserByFieldScores(report.FieldScores)
	}
	if report.Decision.Reason == "" {
		report.Decision.Reason = "highest aggregate field score; reviewer must confirm before accepting parser output"
	}
	report.ReconciliationOutputs = buildReconciliationOutputs(report.Comparisons, docs, report.Decision)
	return report
}

func fieldRawText(doc ParsedDocument, field string) string {
	switch field {
	case "title":
		return strings.TrimSpace(doc.Title)
	case "abstract":
		return strings.TrimSpace(doc.Abstract)
	case "sections":
		parts := []string{}
		for _, section := range doc.Sections {
			parts = append(parts, section.Title)
			for _, passage := range section.Passages {
				parts = append(parts, passage.Text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	case "references":
		parts := []string{}
		for _, ref := range doc.References {
			parts = append(parts, firstNonEmptyArbitration(ref.Title, ref.Raw, ref.DOI))
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func scoreField(doc ParsedDocument, field, raw string) float64 {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	score := 1.0
	if len(doc.Warnings) > 0 {
		score -= 0.1 * float64(len(doc.Warnings))
	}
	if score < 0 {
		return 0
	}
	return score
}

func fieldScoreReason(field, raw string, warnings []string) string {
	if strings.TrimSpace(raw) == "" {
		return field + " missing"
	}
	if len(warnings) > 0 {
		return field + " present with parser warnings"
	}
	return field + " present without parser warnings"
}

func bestParserByFieldScores(scores map[string][]FieldScore) string {
	totals := map[string]float64{}
	for _, fieldScores := range scores {
		for _, score := range fieldScores {
			totals[score.ParserName] += score.Score
		}
	}
	best := ""
	bestScore := -1.0
	for parser, score := range totals {
		if score > bestScore {
			best, bestScore = parser, score
		}
	}
	return best
}

func buildReconciliationOutputs(comparisons []FieldComparison, docs []ParsedDocument, decision ParserArbitrationDecision) []ParserReconciliationOutput {
	outputs := []ParserReconciliationOutput{}
	warnings := map[string][]string{}
	for _, doc := range docs {
		warnings[doc.ParserName] = append([]string{}, doc.Warnings...)
	}
	for _, comparison := range comparisons {
		output := ParserReconciliationOutput{Field: comparison.Field, AcceptedParser: decision.AcceptedParser, Reason: decision.Reason}
		for _, value := range comparison.ParserValues {
			alt := ParserReconciliationAlt{ParserName: value.ParserName, Value: value.RawText, Offset: value.Offset, Warnings: warnings[value.ParserName]}
			if value.ParserName == decision.AcceptedParser {
				output.AcceptedValue = value.RawText
			}
			output.Alternatives = append(output.Alternatives, alt)
		}
		outputs = append(outputs, output)
	}
	return outputs
}

func firstNonEmptyArbitration(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
