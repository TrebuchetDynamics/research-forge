package protocol

import "strings"

type SourcePlan struct {
	SchemaVersion            string                     `json:"schemaVersion"`
	Question                 string                     `json:"question"`
	Framework                Framework                  `json:"framework"`
	Sources                  []SourcePlanEntry          `json:"sources"`
	Warnings                 []string                   `json:"warnings"`
	QueryExpansionProvenance []QueryExpansionProvenance `json:"queryExpansionProvenance,omitempty"`
}

type SourcePlanEntry struct {
	Source                    string   `json:"source"`
	Label                     string   `json:"label"`
	SourceKind                string   `json:"sourceKind"`
	SupportedEntities         []string `json:"supportedEntities"`
	Query                     string   `json:"query"`
	DryRunEstimate            string   `json:"dryRunEstimate"`
	RateLimitPolicy           string   `json:"rateLimitPolicy"`
	AuthRequirement           string   `json:"authRequirement"`
	LiveSmokeStatus           string   `json:"liveSmokeStatus"`
	LicenseShareabilityPolicy string   `json:"licenseShareabilityPolicy"`
	Cacheability              string   `json:"cacheability"`
	ProvenanceFields          []string `json:"provenanceFields"`
	PrivacyWarning            string   `json:"privacyWarning"`
	ReviewerApprovalRequired  bool     `json:"reviewerApprovalRequired"`
	CLICommand                string   `json:"cliCommand"`
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
	registry := DefaultConnectorCapabilityRegistry()
	entries := []SourcePlanEntry{
		discoverySource(registry, "openalex", queryFor("openalex"), "Public API; broad graph imports require cursor/rate provenance."),
		discoverySource(registry, "semantic-scholar", queryFor("semantic-scholar"), "API key must be redacted; field/rate restrictions apply."),
		discoverySource(registry, "crossref", queryFor("crossref"), "Publisher metadata can be incomplete; preserve raw source refs."),
		discoverySource(registry, "arxiv", queryFor("arxiv"), "Preprint metadata and versions require explicit provenance."),
		discoverySource(registry, "pubmed", queryFor("pubmed"), "Biomedical queries may expose search intent to NCBI services."),
		discoverySource(registry, "europepmc", queryFor("europepmc"), "Full-text/license availability must be checked before acquisition."),
		discoverySource(registry, "nasa-ads", queryFor("openalex"), "ADS tokens must be redacted; bibcode metadata is source-specific."),
		discoverySource(registry, "doaj", queryFor("openalex"), "OA journal metadata does not by itself permit full-text redistribution."),
		discoverySource(registry, "core", queryFor("openalex"), "CORE full-text links require license/shareability review."),
		lookupSource(registry, "unpaywall", "DOI lookup for imported records", "Use only after DOI-bearing records exist; PDF URLs still need legal acquisition approval."),
		referenceManagerSource(registry, "zotero", "rforge import csl-json|zotero-rdf <file>", "Zotero exports may include private notes/attachment paths; redact before package export."),
		referenceManagerSource(registry, "jabref", "rforge import bibtex <file>", "BibTeX linked-file fields can contain private local paths."),
		localSource(registry, "local", "rforge project discover/import <path>", "Local PDFs/notes remain private until explicitly approved for acquisition/package inclusion."),
	}
	return SourcePlan{SchemaVersion: "1", Question: plan.Question, Framework: plan.Framework, Sources: entries, Warnings: []string{"Draft source plan: reviewer approval required before network calls, imports, downloads, or package inclusion."}}
}

func discoverySource(registry ConnectorCapabilityRegistry, id, query, warning string) SourcePlanEntry {
	capability := mustCapability(registry, id)
	return sourceFromCapability(capability, query, "preview query only; live counts require explicit source execution", warning, "rforge search --source "+id+" --query "+shellQuote(query))
}

func lookupSource(registry ConnectorCapabilityRegistry, id, query, warning string) SourcePlanEntry {
	capability := mustCapability(registry, id)
	return sourceFromCapability(capability, query, "lookup count equals DOI-bearing imported records", warning, "rforge oa lookup <doi>")
}

func referenceManagerSource(registry ConnectorCapabilityRegistry, id, command, warning string) SourcePlanEntry {
	capability := mustCapability(registry, id)
	return sourceFromCapability(capability, "", "preview import diff before mutating library", warning, command)
}

func localSource(registry ConnectorCapabilityRegistry, id, command, warning string) SourcePlanEntry {
	capability := mustCapability(registry, id)
	return sourceFromCapability(capability, "", "discover local academic files without importing until approved", warning, command)
}

func sourceFromCapability(capability ConnectorCapability, query, dryRun, warning, command string) SourcePlanEntry {
	return SourcePlanEntry{Source: capability.ID, Label: capability.Label, SourceKind: capability.Kind, SupportedEntities: capability.SupportedEntities, Query: query, DryRunEstimate: dryRun, RateLimitPolicy: capability.RateLimitPolicy, AuthRequirement: capability.AuthNeeds, LiveSmokeStatus: capability.LiveSmokeStatus, LicenseShareabilityPolicy: capability.LicenseShareabilityPolicy, Cacheability: capability.Cacheability, ProvenanceFields: capability.ProvenanceFields, PrivacyWarning: warning, ReviewerApprovalRequired: true, CLICommand: command}
}

func mustCapability(registry ConnectorCapabilityRegistry, id string) ConnectorCapability {
	capability, ok := registry.ByID(id)
	if !ok {
		return ConnectorCapability{ID: id, Label: id, Kind: "unknown", AuthNeeds: "unknown", RateLimitPolicy: "unknown", LiveSmokeStatus: "unknown", LicenseShareabilityPolicy: "unknown", Cacheability: "unknown", ProvenanceFields: []string{"source"}}
	}
	return capability
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
