package protocol

import "strings"

type SourcePlan struct {
	SchemaVersion string            `json:"schemaVersion"`
	Question      string            `json:"question"`
	Framework     Framework         `json:"framework"`
	Sources       []SourcePlanEntry `json:"sources"`
	Warnings      []string          `json:"warnings"`
}

type SourcePlanEntry struct {
	Source                   string `json:"source"`
	Label                    string `json:"label"`
	SourceKind               string `json:"sourceKind"`
	Query                    string `json:"query"`
	DryRunEstimate           string `json:"dryRunEstimate"`
	AuthRequirement          string `json:"authRequirement"`
	PrivacyWarning           string `json:"privacyWarning"`
	ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
	CLICommand               string `json:"cliCommand"`
}

func CompileSourcePlanFromQuestion(input QuestionInput) (SourcePlan, error) {
	plan, err := CompileQuestion(input)
	if err != nil {
		return SourcePlan{}, err
	}
	return CompileSourcePlan(plan), nil
}

func CompileSourcePlan(plan CompiledQuestionPlan) SourcePlan {
	queryFor := func(source string) string {
		if q, ok := plan.SourceQueries[source]; ok {
			return q.Query
		}
		if q, ok := plan.SourceQueries["openalex"]; ok {
			return q.Query
		}
		return plan.Question
	}
	entries := []SourcePlanEntry{
		discoverySource("openalex", "OpenAlex", queryFor("openalex"), "none", "Public API; broad graph imports require cursor/rate provenance."),
		discoverySource("semantic-scholar", "Semantic Scholar", queryFor("semantic-scholar"), "optional RFORGE_SEMANTIC_SCHOLAR_API_KEY", "API key must be redacted; field/rate restrictions apply."),
		discoverySource("crossref", "Crossref", queryFor("crossref"), "none", "Publisher metadata can be incomplete; preserve raw source refs."),
		discoverySource("arxiv", "arXiv", queryFor("arxiv"), "none", "Preprint metadata and versions require explicit provenance."),
		discoverySource("pubmed", "PubMed", queryFor("pubmed"), "optional NCBI API key/email", "Biomedical queries may expose search intent to NCBI services."),
		discoverySource("europepmc", "Europe PMC", queryFor("europepmc"), "none", "Full-text/license availability must be checked before acquisition."),
		discoverySource("nasa-ads", "NASA ADS", queryFor("openalex"), "NASA ADS token required for live API", "ADS tokens must be redacted; bibcode metadata is source-specific."),
		discoverySource("doaj", "DOAJ", queryFor("openalex"), "none", "OA journal metadata does not by itself permit full-text redistribution."),
		discoverySource("core", "CORE", queryFor("openalex"), "CORE API key may be required", "CORE full-text links require license/shareability review."),
		lookupSource("unpaywall", "Unpaywall", "DOI lookup for imported records", "email may be required", "Use only after DOI-bearing records exist; PDF URLs still need legal acquisition approval."),
		referenceManagerSource("zotero", "Zotero", "rforge import csl-json|zotero-rdf <file>", "Zotero exports may include private notes/attachment paths; redact before package export."),
		referenceManagerSource("jabref", "JabRef", "rforge import bibtex <file>", "BibTeX linked-file fields can contain private local paths."),
		localSource("local", "Local files", "rforge project discover/import <path>", "Local PDFs/notes remain private until explicitly approved for acquisition/package inclusion."),
	}
	return SourcePlan{SchemaVersion: "1", Question: plan.Question, Framework: plan.Framework, Sources: entries, Warnings: []string{"Draft source plan: reviewer approval required before network calls, imports, downloads, or package inclusion."}}
}

func discoverySource(id, label, query, auth, warning string) SourcePlanEntry {
	return SourcePlanEntry{Source: id, Label: label, SourceKind: "scholarly-source", Query: query, DryRunEstimate: "preview query only; live counts require explicit source execution", AuthRequirement: auth, PrivacyWarning: warning, ReviewerApprovalRequired: true, CLICommand: "rforge search --source " + id + " --query " + shellQuote(query)}
}

func lookupSource(id, label, query, auth, warning string) SourcePlanEntry {
	return SourcePlanEntry{Source: id, Label: label, SourceKind: "oa-lookup", Query: query, DryRunEstimate: "lookup count equals DOI-bearing imported records", AuthRequirement: auth, PrivacyWarning: warning, ReviewerApprovalRequired: true, CLICommand: "rforge oa lookup <doi>"}
}

func referenceManagerSource(id, label, command, warning string) SourcePlanEntry {
	return SourcePlanEntry{Source: id, Label: label, SourceKind: "reference-manager", DryRunEstimate: "preview import diff before mutating library", AuthRequirement: "local export file", PrivacyWarning: warning, ReviewerApprovalRequired: true, CLICommand: command}
}

func localSource(id, label, command, warning string) SourcePlanEntry {
	return SourcePlanEntry{Source: id, Label: label, SourceKind: "local-import", DryRunEstimate: "discover local academic files without importing until approved", AuthRequirement: "local filesystem access", PrivacyWarning: warning, ReviewerApprovalRequired: true, CLICommand: command}
}

func (p SourcePlan) BySource(source string) (SourcePlanEntry, bool) {
	for _, entry := range p.Sources {
		if entry.Source == source {
			return entry, true
		}
	}
	return SourcePlanEntry{}, false
}

func (p SourcePlan) MustSource(source string) SourcePlanEntry {
	entry, _ := p.BySource(source)
	return entry
}

func shellQuote(value string) string {
	value = strings.ReplaceAll(value, `'`, `'"'"'`)
	return `'` + value + `'`
}
