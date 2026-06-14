package parsing

// ReferenceReviewItem identifies a parsed reference that needs human review.
type ReferenceReviewItem struct {
	Index      int     `json:"index"`
	Title      string  `json:"title,omitempty"`
	DOI        string  `json:"doi,omitempty"`
	Raw        string  `json:"raw,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Reason     string  `json:"reason"`
}

// ReferenceReviewReport summarizes ambiguous parsed references for manual review.
type ReferenceReviewReport struct {
	PaperID   string                `json:"paperId"`
	Threshold float64               `json:"threshold"`
	Items     []ReferenceReviewItem `json:"items"`
}

// AmbiguousReferences flags references with missing identifiers/titles or low parser confidence.
func AmbiguousReferences(doc ParsedDocument, threshold float64) ReferenceReviewReport {
	if threshold <= 0 {
		threshold = 0.75
	}
	report := ReferenceReviewReport{PaperID: doc.PaperID, Threshold: threshold}
	for i, ref := range doc.References {
		reason := ""
		switch {
		case ref.Title == "" && ref.DOI == "":
			reason = "missing_title_and_doi"
		case ref.Confidence > 0 && ref.Confidence < threshold:
			reason = "low_confidence"
		case ref.Raw != "" && ref.Title == "":
			reason = "raw_only"
		}
		if reason == "" {
			continue
		}
		report.Items = append(report.Items, ReferenceReviewItem{Index: i, Title: ref.Title, DOI: ref.DOI, Raw: ref.Raw, Confidence: ref.Confidence, Reason: reason})
	}
	return report
}
