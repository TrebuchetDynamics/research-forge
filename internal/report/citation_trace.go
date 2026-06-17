package report

import (
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type CitationEvidenceTraceInput struct {
	Claims          []evidence.CitationLockedSuggestion
	EvidenceItems   []evidence.EvidenceItem
	AnalysisRun     analysis.AnalysisRun
	ParsedDocuments []parsing.ParsedDocument
	LibraryRecords  []library.PaperRecord
	PDFBaseURL      string
}

type CitationEvidenceTraceView struct {
	SchemaVersion string           `json:"schemaVersion"`
	Claims        []ClaimTraceView `json:"claims"`
}

type ClaimTraceView struct {
	ClaimID               string                  `json:"claimId"`
	PaperID               string                  `json:"paperId"`
	ClaimText             string                  `json:"claimText"`
	EffectSizeRows        []analysis.InputRow     `json:"effectSizeRows"`
	AcceptedEvidence      []evidence.EvidenceItem `json:"acceptedEvidence"`
	Passages              []TracePassage          `json:"passages"`
	ParserOutputs         []string                `json:"parserOutputs"`
	PDFViewURL            string                  `json:"pdfViewUrl,omitempty"`
	ReferenceManagerItems []string                `json:"referenceManagerItems,omitempty"`
	SourceAPIRecords      []string                `json:"sourceApiRecords,omitempty"`
	RawRequestResponse    map[string]string       `json:"rawRequestResponse,omitempty"`
}

type TracePassage struct {
	PassageID     string             `json:"passageId"`
	Text          string             `json:"text"`
	ParserName    string             `json:"parserName"`
	ParserVersion string             `json:"parserVersion"`
	Offset        parsing.TextOffset `json:"offset"`
}

func BuildCitationEvidenceTraceView(input CitationEvidenceTraceInput) CitationEvidenceTraceView {
	view := CitationEvidenceTraceView{SchemaVersion: "1"}
	for _, claim := range input.Claims {
		row := ClaimTraceView{ClaimID: claim.ID, PaperID: claim.PaperID, ClaimText: claim.SuggestedText, RawRequestResponse: map[string]string{}}
		for _, effect := range input.AnalysisRun.InputRows {
			if effect.PaperID == claim.PaperID {
				row.EffectSizeRows = append(row.EffectSizeRows, effect)
			}
		}
		for _, item := range input.EvidenceItems {
			if item.PaperID == claim.PaperID && item.Status == evidence.StatusAccepted {
				row.AcceptedEvidence = append(row.AcceptedEvidence, item)
			}
		}
		refs := claimRefs(claim, row.AcceptedEvidence)
		for _, doc := range input.ParsedDocuments {
			if doc.PaperID != claim.PaperID {
				continue
			}
			if doc.ParserName != "" {
				row.ParserOutputs = append(row.ParserOutputs, doc.ParserName+":"+doc.ParserVersion)
			}
			for _, section := range doc.Sections {
				for _, passage := range section.Passages {
					if refs[passage.ID] {
						row.Passages = append(row.Passages, TracePassage{PassageID: passage.ID, Text: passage.Text, ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, Offset: passage.Offset})
					}
				}
			}
		}
		if base := strings.TrimRight(input.PDFBaseURL, "/"); base != "" {
			row.PDFViewURL = base + "/" + claim.PaperID + "/pdf"
			for ref := range refs {
				row.PDFViewURL += "#" + ref
				break
			}
		}
		for _, record := range input.LibraryRecords {
			if !recordMatchesPaper(record, claim.PaperID) {
				continue
			}
			if record.Identifiers.ZoteroItemKey != "" {
				row.ReferenceManagerItems = append(row.ReferenceManagerItems, "zotero:"+record.Identifiers.ZoteroItemKey)
			}
			for _, sourceRef := range record.SourceRefs {
				if sourceRef.Source != "" {
					row.SourceAPIRecords = append(row.SourceAPIRecords, sourceRef.Source+":"+sourceRef.RawPayloadRef)
				}
				for key, value := range sourceRef.Metadata {
					row.RawRequestResponse[key] = value
				}
			}
		}
		if len(row.RawRequestResponse) == 0 {
			row.RawRequestResponse = nil
		}
		view.Claims = append(view.Claims, row)
	}
	return view
}

func claimRefs(claim evidence.CitationLockedSuggestion, items []evidence.EvidenceItem) map[string]bool {
	refs := map[string]bool{}
	for _, lock := range claim.CitationLocks {
		if strings.TrimSpace(lock.Ref) != "" {
			refs[lock.Ref] = true
		}
	}
	for _, item := range items {
		if strings.TrimSpace(item.Support.Ref) != "" {
			refs[item.Support.Ref] = true
		}
	}
	return refs
}

func recordMatchesPaper(record library.PaperRecord, paperID string) bool {
	ids := record.Identifiers
	return paperID != "" && (paperID == ids.DOI || paperID == ids.PMID || paperID == ids.PMCID || paperID == ids.ArXivID || paperID == ids.OpenAlexID || paperID == ids.ZoteroItemKey || paperID == record.Title)
}
