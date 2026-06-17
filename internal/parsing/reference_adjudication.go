package parsing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ReferenceAdjudicationSchemaVersion = "1"

type ReferenceAdjudication struct {
	SchemaVersion  string              `json:"schemaVersion"`
	ID             string              `json:"id"`
	PaperID        string              `json:"paperId"`
	ReferenceIndex int                 `json:"referenceIndex"`
	Decision       string              `json:"decision"`
	Reviewer       string              `json:"reviewer"`
	Reason         string              `json:"reason"`
	Original       Reference           `json:"original"`
	Correction     ReferenceCorrection `json:"correction,omitempty"`
	CreatedAt      string              `json:"createdAt"`
}

type ReferenceCorrection struct {
	Title string `json:"title,omitempty"`
	DOI   string `json:"doi,omitempty"`
	Raw   string `json:"raw,omitempty"`
}

type ReferenceAdjudicationReport struct {
	PaperID    string                     `json:"paperId"`
	Items      []AdjudicatedReferenceItem `json:"items"`
	Accepted   int                        `json:"accepted"`
	Corrected  int                        `json:"corrected"`
	Rejected   int                        `json:"rejected"`
	Deferred   int                        `json:"deferred"`
	Unreviewed int                        `json:"unreviewed"`
}

type AdjudicatedReferenceItem struct {
	Index     int                    `json:"index"`
	Status    string                 `json:"status"`
	Reference Reference              `json:"reference"`
	Decision  *ReferenceAdjudication `json:"decision,omitempty"`
}

func NewReferenceAdjudication(doc ParsedDocument, index int, decision, reviewer, reason string, correction ReferenceCorrection) (ReferenceAdjudication, error) {
	decision = strings.TrimSpace(strings.ToLower(decision))
	if !ValidReferenceAdjudicationDecision(decision) {
		return ReferenceAdjudication{}, fmt.Errorf("invalid reference adjudication decision %q", decision)
	}
	if index < 0 || index >= len(doc.References) {
		return ReferenceAdjudication{}, fmt.Errorf("reference index %d out of range", index)
	}
	if strings.TrimSpace(reviewer) == "" {
		return ReferenceAdjudication{}, fmt.Errorf("reviewer is required")
	}
	if strings.TrimSpace(reason) == "" {
		return ReferenceAdjudication{}, fmt.Errorf("reason is required")
	}
	if decision == "correct" && strings.TrimSpace(correction.Title) == "" && strings.TrimSpace(correction.DOI) == "" && strings.TrimSpace(correction.Raw) == "" {
		return ReferenceAdjudication{}, fmt.Errorf("correction requires title, doi, or raw")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return ReferenceAdjudication{SchemaVersion: ReferenceAdjudicationSchemaVersion, ID: fmt.Sprintf("%s-ref-%d-%s", doc.PaperID, index, strings.ReplaceAll(now, ":", "")), PaperID: doc.PaperID, ReferenceIndex: index, Decision: decision, Reviewer: strings.TrimSpace(reviewer), Reason: strings.TrimSpace(reason), Original: doc.References[index], Correction: correction, CreatedAt: now}, nil
}

func ValidReferenceAdjudicationDecision(decision string) bool {
	switch decision {
	case "accept", "correct", "reject", "defer":
		return true
	default:
		return false
	}
}

func AppendReferenceAdjudication(path string, record ReferenceAdjudication) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	return err
}

func LoadReferenceAdjudications(path string) ([]ReferenceAdjudication, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	records := []ReferenceAdjudication{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record ReferenceAdjudication
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, scanner.Err()
}

func ApplyReferenceAdjudications(doc ParsedDocument, records []ReferenceAdjudication) ReferenceAdjudicationReport {
	latest := map[int]ReferenceAdjudication{}
	for _, record := range records {
		if record.PaperID == doc.PaperID && record.ReferenceIndex >= 0 && record.ReferenceIndex < len(doc.References) {
			latest[record.ReferenceIndex] = record
		}
	}
	report := ReferenceAdjudicationReport{PaperID: doc.PaperID}
	for i, ref := range doc.References {
		item := AdjudicatedReferenceItem{Index: i, Status: "unreviewed", Reference: ref}
		if record, ok := latest[i]; ok {
			item.Decision = &record
			switch record.Decision {
			case "accept":
				item.Status = "accepted"
				report.Accepted++
			case "correct":
				item.Status = "corrected"
				item.Reference = applyReferenceCorrection(ref, record.Correction)
				report.Corrected++
			case "reject":
				item.Status = "rejected"
				report.Rejected++
			case "defer":
				item.Status = "deferred"
				report.Deferred++
			}
		} else {
			report.Unreviewed++
		}
		report.Items = append(report.Items, item)
	}
	return report
}

func applyReferenceCorrection(ref Reference, correction ReferenceCorrection) Reference {
	if strings.TrimSpace(correction.Title) != "" {
		ref.Title = strings.TrimSpace(correction.Title)
	}
	if strings.TrimSpace(correction.DOI) != "" {
		ref.DOI = strings.TrimSpace(correction.DOI)
	}
	if strings.TrimSpace(correction.Raw) != "" {
		ref.Raw = strings.TrimSpace(correction.Raw)
	}
	return ref
}
