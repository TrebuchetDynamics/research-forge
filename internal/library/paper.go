package library

import (
	"fmt"
	"strings"
)

// Identifiers are normalized scholarly IDs for a PaperRecord.
type Identifiers struct {
	DOI               string
	ArXivID           string
	PMID              string
	PMCID             string
	OpenAlexID        string
	CrossrefID        string
	SemanticScholarID string
}

// Author is one normalized paper author.
type Author struct {
	Given  string
	Family string
	ORCID  string
}

// SourceRef records source-specific payload Provenance for a PaperRecord.
type SourceRef struct {
	Source        string
	RawPayloadRef string
	RetrievedAt   string
	Metadata      map[string]string
}

// PaperRecordInput is caller-provided scholarly metadata before normalization.
type PaperRecordInput struct {
	Title         string
	Identifiers   Identifiers
	Authors       []Author
	Abstract      string
	Year          int
	Venue         string
	Publisher     string
	URLs          []string
	License       string
	OpenAccess    bool
	SourcePayload string
	SourceRefs    []SourceRef
}

// PaperRecord is a normalized scholarly metadata entry for a paper or preprint.
type PaperRecord struct {
	Title         string
	Identifiers   Identifiers
	Authors       []Author
	Abstract      string
	Year          int
	Venue         string
	Publisher     string
	URLs          []string
	License       string
	OpenAccess    bool
	SourcePayload string
	SourceRefs    []SourceRef
}

// NewPaperRecord validates and normalizes a paper metadata record.
func NewPaperRecord(input PaperRecordInput) (PaperRecord, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return PaperRecord{}, fmt.Errorf("paper title is required")
	}
	identifiers := normalizeIdentifiers(input.Identifiers)
	if !identifiers.any() {
		return PaperRecord{}, fmt.Errorf("at least one paper identifier is required")
	}
	return PaperRecord{
		Title:         title,
		Identifiers:   identifiers,
		Authors:       normalizeAuthors(input.Authors),
		Abstract:      strings.TrimSpace(input.Abstract),
		Year:          input.Year,
		Venue:         strings.TrimSpace(input.Venue),
		Publisher:     strings.TrimSpace(input.Publisher),
		URLs:          normalizeStrings(input.URLs),
		License:       strings.TrimSpace(input.License),
		OpenAccess:    input.OpenAccess,
		SourcePayload: strings.TrimSpace(input.SourcePayload),
		SourceRefs:    normalizeSourceRefs(input.SourceRefs),
	}, nil
}

func normalizeSourceRefs(refs []SourceRef) []SourceRef {
	out := make([]SourceRef, 0, len(refs))
	for _, ref := range refs {
		ref.Source = strings.TrimSpace(ref.Source)
		ref.RawPayloadRef = strings.TrimSpace(ref.RawPayloadRef)
		ref.RetrievedAt = strings.TrimSpace(ref.RetrievedAt)
		ref.Metadata = normalizeMetadata(ref.Metadata)
		if ref.Source == "" && ref.RawPayloadRef == "" && ref.RetrievedAt == "" && len(ref.Metadata) == 0 {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func normalizeMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	normalized := map[string]string{}
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			normalized[key] = value
		}
	}
	return normalized
}

func normalizeAuthors(authors []Author) []Author {
	out := make([]Author, 0, len(authors))
	for _, author := range authors {
		author.Given = strings.TrimSpace(author.Given)
		author.Family = strings.TrimSpace(author.Family)
		author.ORCID = strings.TrimPrefix(strings.TrimSpace(author.ORCID), "https://orcid.org/")
		if author.Given == "" && author.Family == "" && author.ORCID == "" {
			continue
		}
		out = append(out, author)
	}
	return out
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeIdentifiers(ids Identifiers) Identifiers {
	ids.DOI = normalizeDOI(ids.DOI)
	ids.CrossrefID = normalizeDOI(ids.CrossrefID)
	ids.ArXivID = strings.TrimPrefix(strings.TrimSpace(ids.ArXivID), "arXiv:")
	ids.PMID = strings.TrimSpace(ids.PMID)
	ids.PMCID = normalizePMCID(ids.PMCID)
	ids.OpenAlexID = strings.TrimPrefix(strings.TrimSpace(ids.OpenAlexID), "https://openalex.org/")
	ids.SemanticScholarID = strings.TrimSpace(ids.SemanticScholarID)
	return ids
}

func normalizePMCID(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" || strings.HasPrefix(value, "PMC") {
		return value
	}
	return "PMC" + value
}

func normalizeDOI(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://doi.org/")
	value = strings.TrimPrefix(value, "http://doi.org/")
	value = strings.TrimPrefix(value, "doi:")
	return strings.TrimSpace(value)
}

func (ids Identifiers) any() bool {
	return ids.DOI != "" || ids.ArXivID != "" || ids.PMID != "" || ids.PMCID != "" || ids.OpenAlexID != "" || ids.CrossrefID != "" || ids.SemanticScholarID != ""
}
