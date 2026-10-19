// Package claims defines storage-independent claim/evidence records and separate
// verification facets. No facet or combination of facets asserts scientific truth.
package claims

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const SchemaVersion = "0"

// Claim is an assertion and its cited evidence, not a storage or transport envelope.
// WorkflowStatus preserves a legacy review state; it does not set any facet.
type Claim struct {
	SchemaVersion  string              `json:"schemaVersion"`
	ID             string              `json:"id"`
	Text           string              `json:"text"`
	WorkflowStatus string              `json:"workflowStatus,omitempty"`
	Evidence       []EvidenceReference `json:"evidence"`
	Facets         VerificationFacets  `json:"facets"`
}

// Identifier names a source within a scheme (doi, arxiv, pmid, or local, for example).
// Local identifiers must remain scoped to their originating research project.
type Identifier struct {
	Scheme string `json:"scheme"`
	Value  string `json:"value"`
}

type EvidenceRelation string

const (
	RelationSupports    EvidenceRelation = "supports"
	RelationContradicts EvidenceRelation = "contradicts"
	RelationContext     EvidenceRelation = "context"
)

// EvidenceReference binds a quote to a source passage. ContentHash is the
// SHA-256 of the complete UTF-8 passage text, formatted as sha256:<lowercase hex>.
// Missing hashes/offsets mean unavailable, never a successful integrity check.
type EvidenceReference struct {
	Identifier     Identifier       `json:"identifier"`
	PassageID      string           `json:"passageId,omitempty"`
	Relation       EvidenceRelation `json:"relation"`
	ExactText      string           `json:"exactText,omitempty"`
	ContentHash    string           `json:"contentHash,omitempty"`
	Offsets        *TextOffsets     `json:"offsets,omitempty"`
	SourceMetadata *SourceMetadata  `json:"sourceMetadata,omitempty"`
}

// SourceMetadata contains optional expected bibliographic values. A resolved
// identifier alone cannot override a disagreement with these values.
type SourceMetadata struct {
	Title string `json:"title,omitempty"`
	Year  int    `json:"year,omitempty"`
}

// TextOffsets is a half-open UTF-8 byte range [Start, End) in the parser's
// document text stream, not in a PDF or an independently normalized quote.
// Parser name/version and the passage hash must accompany a reusable location.
type TextOffsets struct {
	Start         int    `json:"start"`
	End           int    `json:"end"`
	ParserName    string `json:"parserName"`
	ParserVersion string `json:"parserVersion"`
}

// VerificationFacets never collapses the five properties into an overall state.
// A nil assessment means not checked; it has no invented checker or timestamp.
// Each non-nil assessment covers the claim and all of its evidence references.
type VerificationFacets struct {
	SourceIdentity   *Assessment      `json:"sourceIdentity"`
	ContentIntegrity *Assessment      `json:"contentIntegrity"`
	PassageMatch     *Assessment      `json:"passageMatch"`
	SemanticSupport  *SemanticSupport `json:"semanticSupport"`
	HumanApproval    *HumanApproval   `json:"humanApproval"`
}

type State string

const (
	StatePassed       State = "passed"
	StateFailed       State = "failed"
	StateInconclusive State = "inconclusive"
)

type CheckerKind string

const (
	CheckerEngine CheckerKind = "engine"
	CheckerAgent  CheckerKind = "agent"
	CheckerHuman  CheckerKind = "human"
)

// Checker identifies the actor, tool/agent version and, for semantic support,
// model identity (including its version in Model when available).
// Kind is attribution, not a computed provenance level or proof of an engine run.
type Checker struct {
	Kind    CheckerKind `json:"kind"`
	ID      string      `json:"id"`
	Version string      `json:"version,omitempty"`
	Model   string      `json:"model,omitempty"`
}

// Assessment records one facet's outcome at an explicit RFC3339 timestamp.
// Passed means only that the named property met the checker's criterion.
type Assessment struct {
	State     State     `json:"state"`
	Checker   Checker   `json:"checker"`
	Timestamp time.Time `json:"timestamp"`
}

// SemanticSupport is always agent-reported. Its private value prevents a Go
// literal, assignment, or JSON decode from installing an engine checker.
// The zero value is uninitialized; use nil for an unchecked facet.
type SemanticSupport struct {
	assessment Assessment
}

func NewSemanticSupport(value Assessment) (SemanticSupport, error) {
	if err := validateAssessment(value, CheckerAgent); err != nil {
		return SemanticSupport{}, err
	}
	if strings.TrimSpace(value.Checker.Model) == "" {
		return SemanticSupport{}, fmt.Errorf("semantic support requires model identity")
	}
	return SemanticSupport{assessment: value}, nil
}

// Assessment returns a copy; changing it cannot change the recorded attribution.
func (s SemanticSupport) Assessment() Assessment { return s.assessment }

func (s SemanticSupport) MarshalJSON() ([]byte, error) {
	if _, err := NewSemanticSupport(s.assessment); err != nil {
		return nil, err
	}
	return json.Marshal(s.assessment)
}

func (s *SemanticSupport) UnmarshalJSON(data []byte) error {
	var value Assessment
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	next, err := NewSemanticSupport(value)
	if err != nil {
		return err
	}
	*s = next
	return nil
}

// HumanApproval records a named reviewer's decision, not an automated decision.
// The zero value is uninitialized; use nil for an unchecked facet.
type HumanApproval struct {
	assessment Assessment
}

func NewHumanApproval(value Assessment) (HumanApproval, error) {
	if err := validateAssessment(value, CheckerHuman); err != nil {
		return HumanApproval{}, err
	}
	return HumanApproval{assessment: value}, nil
}

// Assessment returns a copy of the reviewer's assessment.
func (h HumanApproval) Assessment() Assessment { return h.assessment }

func (h HumanApproval) MarshalJSON() ([]byte, error) {
	if _, err := NewHumanApproval(h.assessment); err != nil {
		return nil, err
	}
	return json.Marshal(h.assessment)
}

func (h *HumanApproval) UnmarshalJSON(data []byte) error {
	var value Assessment
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	next, err := NewHumanApproval(value)
	if err != nil {
		return err
	}
	*h = next
	return nil
}

func validateAssessment(value Assessment, kind CheckerKind) error {
	if value.Checker.Kind != kind || strings.TrimSpace(value.Checker.ID) == "" {
		return fmt.Errorf("assessment requires a named %s checker", kind)
	}
	switch value.State {
	case StatePassed, StateFailed, StateInconclusive:
	default:
		return fmt.Errorf("unsupported assessment state %q", value.State)
	}
	if value.Timestamp.IsZero() {
		return fmt.Errorf("assessment requires a timestamp")
	}
	return nil
}
