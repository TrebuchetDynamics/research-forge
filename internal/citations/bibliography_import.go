package citations

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type BibliographyImportReport struct {
	SchemaVersion     string             `json:"schemaVersion"`
	PaperID           string             `json:"paperId"`
	EdgeCount         int                `json:"edgeCount"`
	Edges             []BibliographyEdge `json:"edges"`
	CitationSpanLinks []CitationSpanLink `json:"citationSpanLinks"`
	EvidenceLinks     []EvidenceLink     `json:"evidenceLinks"`
	Graph             *Graph             `json:"-"`
}

type BibliographyEdge struct {
	SourceID       string `json:"sourceId"`
	TargetID       string `json:"targetId"`
	ReferenceIndex int    `json:"referenceIndex"`
	Raw            string `json:"raw,omitempty"`
	Title          string `json:"title,omitempty"`
	DOI            string `json:"doi,omitempty"`
}

type CitationSpanLink struct {
	CitationSpanID string             `json:"citationSpanId"`
	PassageID      string             `json:"passageId"`
	ReferenceIndex int                `json:"referenceIndex"`
	TargetID       string             `json:"targetId"`
	SpanText       string             `json:"spanText"`
	Offset         parsing.TextOffset `json:"offset"`
}

type EvidenceLink struct {
	PaperID             string `json:"paperId"`
	EvidenceIndex       int    `json:"evidenceIndex"`
	EvidenceSupportRef  string `json:"evidenceSupportRef"`
	CitationSpanID      string `json:"citationSpanId"`
	CitationTargetID    string `json:"citationTargetId"`
	CitationReferenceID int    `json:"citationReferenceIndex"`
}

func ImportParsedBibliography(doc parsing.ParsedDocument, items []evidence.EvidenceItem) BibliographyImportReport {
	graph := NewGraph()
	report := BibliographyImportReport{SchemaVersion: "1", PaperID: doc.PaperID, Graph: graph}
	targets := map[int]string{}
	for i, ref := range doc.References {
		target := referenceTargetID(i, ref)
		targets[i] = target
		graph.AddCitation(doc.PaperID, target)
		report.Edges = append(report.Edges, BibliographyEdge{SourceID: doc.PaperID, TargetID: target, ReferenceIndex: i, Raw: ref.Raw, Title: ref.Title, DOI: ref.DOI})
	}
	for _, span := range doc.CitationSpans {
		target := targets[span.ReferenceIndex]
		if target == "" {
			continue
		}
		report.CitationSpanLinks = append(report.CitationSpanLinks, CitationSpanLink{CitationSpanID: span.ID, PassageID: span.PassageID, ReferenceIndex: span.ReferenceIndex, TargetID: target, SpanText: span.Text, Offset: span.Offset})
	}
	report.EvidenceLinks = linkEvidenceToCitationSpans(doc.PaperID, items, report.CitationSpanLinks)
	sort.Slice(report.Edges, func(i, j int) bool { return report.Edges[i].TargetID < report.Edges[j].TargetID })
	sort.Slice(report.CitationSpanLinks, func(i, j int) bool {
		return report.CitationSpanLinks[i].CitationSpanID < report.CitationSpanLinks[j].CitationSpanID
	})
	report.EdgeCount = len(report.Edges)
	return report
}

func linkEvidenceToCitationSpans(paperID string, items []evidence.EvidenceItem, spans []CitationSpanLink) []EvidenceLink {
	byPassage := map[string][]CitationSpanLink{}
	for _, span := range spans {
		byPassage[span.PassageID] = append(byPassage[span.PassageID], span)
	}
	links := []EvidenceLink{}
	for i, item := range items {
		if item.PaperID != paperID || item.Support.Kind != evidence.SupportPassage || strings.TrimSpace(item.Support.Ref) == "" {
			continue
		}
		for _, span := range byPassage[item.Support.Ref] {
			links = append(links, EvidenceLink{PaperID: item.PaperID, EvidenceIndex: i, EvidenceSupportRef: item.Support.Ref, CitationSpanID: span.CitationSpanID, CitationTargetID: span.TargetID, CitationReferenceID: span.ReferenceIndex})
		}
	}
	return links
}

func referenceTargetID(index int, ref parsing.Reference) string {
	if doi := strings.TrimSpace(strings.ToLower(ref.DOI)); doi != "" {
		return "doi:" + doi
	}
	if raw := strings.TrimSpace(ref.Raw); raw != "" {
		return "rawref:" + stableSlug(raw)
	}
	if title := strings.TrimSpace(ref.Title); title != "" {
		return "title:" + stableSlug(title)
	}
	return fmt.Sprintf("ref:%d", index)
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func stableSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlug.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 64 {
		return value[:64]
	}
	return value
}
