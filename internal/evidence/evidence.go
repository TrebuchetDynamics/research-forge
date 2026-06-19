package evidence

import (
	"fmt"
	"strings"
)

type Field struct {
	Name string
	Type string
}
type SchemaInput struct {
	Name   string
	Fields []Field
}
type Schema struct {
	SchemaVersion string
	Name          string
	Fields        []Field
}

func NewSchema(input SchemaInput) (Schema, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Schema{}, fmt.Errorf("schema name is required")
	}
	if len(input.Fields) == 0 {
		return Schema{}, fmt.Errorf("schema fields are required")
	}
	fields := []Field{}
	for _, f := range input.Fields {
		f.Name = strings.TrimSpace(f.Name)
		f.Type = strings.TrimSpace(f.Type)
		if f.Name == "" || f.Type == "" {
			return Schema{}, fmt.Errorf("schema field name and type are required")
		}
		fields = append(fields, f)
	}
	return Schema{SchemaVersion: "1", Name: name, Fields: fields}, nil
}

type SupportKind string

const (
	SupportPassage  SupportKind = "passage"
	SupportTable    SupportKind = "table"
	SupportFigure   SupportKind = "figure"
	SupportEquation SupportKind = "equation"
	SupportDataset  SupportKind = "dataset"
	SupportCitation SupportKind = "citation"
)

type Support struct {
	Kind SupportKind
	Ref  string
}

type Status string

const (
	StatusSuggested Status = "suggested"
	StatusAccepted  Status = "accepted"
	StatusRejected  Status = "rejected"
	StatusCorrected Status = "corrected"
)

type EvidenceInput struct {
	PaperID    string
	SchemaName string
	Values     map[string]string
	Support    Support
	Status     Status
}
type CorrectionEvent struct {
	Status   Status
	Reviewer string
	Note     string
}
type EvidenceItem struct {
	PaperID     string
	SchemaName  string
	Values      map[string]string
	Support     Support
	Status      Status
	History     []CorrectionEvent
	SuggestedBy string
}

func NewEvidenceItem(input EvidenceInput) (EvidenceItem, error) {
	if strings.TrimSpace(input.PaperID) == "" {
		return EvidenceItem{}, fmt.Errorf("paper id is required")
	}
	if input.Status == "" {
		input.Status = StatusSuggested
	}
	return EvidenceItem{PaperID: input.PaperID, SchemaName: input.SchemaName, Values: input.Values, Support: input.Support, Status: input.Status}, nil
}
func (e *EvidenceItem) Transition(status Status, reviewer, note string) error {
	if status == StatusSuggested {
		return fmt.Errorf("cannot transition back to suggested")
	}
	if strings.TrimSpace(reviewer) == "" {
		return fmt.Errorf("reviewer is required")
	}
	e.Status = status
	e.History = append(e.History, CorrectionEvent{Status: status, Reviewer: reviewer, Note: note})
	return nil
}
func (e EvidenceItem) AcceptWithoutReview() error {
	return fmt.Errorf("LLM suggestions require human review before acceptance")
}

type AuditIssue struct {
	Code    string
	PaperID string
}

func Audit(items []EvidenceItem) []AuditIssue {
	var issues []AuditIssue
	for _, item := range items {
		if item.Status == StatusAccepted && (item.Support.Kind == "" || strings.TrimSpace(item.Support.Ref) == "") {
			issues = append(issues, AuditIssue{Code: "accepted_without_support", PaperID: item.PaperID})
		}
	}
	return issues
}

type LLMConfig struct {
	Provider string
	Model    string
	APIKey   string
}

func (c LLMConfig) Redacted() LLMConfig {
	if c.APIKey != "" {
		c.APIKey = "[redacted]"
	}
	return c
}

// ExtractionTarget describes a single numeric field to extract from abstract text.
type ExtractionTarget struct {
	Name       string // e.g. "value_pct"
	Unit       string // e.g. "%"
	PromptHint string // short natural-language cue for the LLM
}

// ExtractionField is one field definition in an ExtractionSchema.
type ExtractionField struct {
	Name        string
	Description string
	Required    bool
}

// ExtractionSchema describes the full set of fields an abstract extraction LLM call should populate.
type ExtractionSchema struct {
	Name     string
	Required []ExtractionField
	Optional []ExtractionField
}

// STHExtractionSchemaPreset returns the built-in extraction schema for solar-to-hydrogen
// efficiency benchmarking (ADR-0007). Required fields block analysis readiness if absent.
func STHExtractionSchemaPreset() ExtractionSchema {
	return ExtractionSchema{
		Name: "sth-efficiency",
		Required: []ExtractionField{
			{Name: "value_pct", Description: "Solar-to-hydrogen efficiency in %", Required: true},
			{Name: "device_type", Description: "pec | pv-electrolysis | particle-suspension | biohybrid", Required: true},
			{Name: "auxiliary_bias", Description: "unassisted | assisted | unknown", Required: true},
			{Name: "measurement_standard", Description: "am1.5g-100 | non-standard | unknown", Required: true},
			{Name: "verbatim_quote", Description: "Exact sentence from the abstract that reports the efficiency value", Required: true},
			{Name: "confidence", Description: "Extraction confidence 0–1", Required: true},
		},
		Optional: []ExtractionField{
			{Name: "ci_lower", Description: "Lower bound of reported confidence interval (same unit as value_pct)"},
			{Name: "ci_upper", Description: "Upper bound of reported confidence interval (same unit as value_pct)"},
			{Name: "se", Description: "Reported standard error (same unit as value_pct)"},
			{Name: "target_reaction", Description: "e.g. water-splitting, CO2-reduction"},
			{Name: "electrode_material", Description: "Primary light-absorbing electrode material"},
			{Name: "electrolyte", Description: "Electrolyte composition"},
			{Name: "illumination_intensity_mwcm2", Description: "Illumination intensity in mW/cm²"},
			{Name: "active_area_cm2", Description: "Active device area in cm²"},
		},
	}
}

// SuggestRequest carries the context for an LLM-backed evidence suggestion.
type SuggestRequest struct {
	PaperID      string
	AbstractText string          // populated for abstract extraction path
	TargetField  ExtractionTarget // populated for abstract extraction path
}

type SuggestionAdapter interface {
	Suggest(SuggestRequest) (EvidenceItem, error)
	Name() string
}
type NoopSuggestionAdapter struct{}

func (NoopSuggestionAdapter) Name() string { return "noop-llm" }
func (NoopSuggestionAdapter) Suggest(request SuggestRequest) (EvidenceItem, error) {
	return EvidenceItem{PaperID: request.PaperID, Status: StatusSuggested, SuggestedBy: "noop-llm"}, nil
}
func SuggestWithLLM(adapter SuggestionAdapter, request SuggestRequest) (EvidenceItem, error) {
	item, err := adapter.Suggest(request)
	if err != nil {
		return EvidenceItem{}, err
	}
	item.Status = StatusSuggested
	item.SuggestedBy = adapter.Name()
	return item, nil
}
