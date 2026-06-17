package protocol

// ConnectorCapabilityRegistry describes source/tool adapter behavior needed for
// source planning, review gates, package manifests, and dashboard warnings.
type ConnectorCapabilityRegistry struct {
	SchemaVersion string                `json:"schemaVersion"`
	Connectors    []ConnectorCapability `json:"connectors"`
}

type ConnectorCapability struct {
	ID                        string   `json:"id"`
	Label                     string   `json:"label"`
	Kind                      string   `json:"kind"`
	SupportedEntities         []string `json:"supportedEntities"`
	RateLimitPolicy           string   `json:"rateLimitPolicy"`
	AuthNeeds                 string   `json:"authNeeds"`
	LiveSmokeStatus           string   `json:"liveSmokeStatus"`
	LicenseShareabilityPolicy string   `json:"licenseShareabilityPolicy"`
	Cacheability              string   `json:"cacheability"`
	ProvenanceFields          []string `json:"provenanceFields"`
}

func DefaultConnectorCapabilityRegistry() ConnectorCapabilityRegistry {
	return ConnectorCapabilityRegistry{SchemaVersion: "1", Connectors: []ConnectorCapability{
		{ID: "openalex", Label: "OpenAlex", Kind: "scholarly-source", SupportedEntities: []string{"works", "authors", "institutions", "concepts", "references", "citations"}, RateLimitPolicy: "public polite API; cursor/resume state required for multi-page imports", AuthNeeds: "none", LiveSmokeStatus: "covered by source-live-smoke", LicenseShareabilityPolicy: "open metadata; preserve source IDs and request provenance", Cacheability: "cache query responses with request parameters and cursor state", ProvenanceFields: []string{"source", "query", "filters", "cursor", "work_id", "raw_ref"}},
		{ID: "semantic-scholar", Label: "Semantic Scholar", Kind: "scholarly-source", SupportedEntities: []string{"papers", "authors", "references", "citations"}, RateLimitPolicy: "quota/rate limited; Retry-After honored by shared HTTP backoff", AuthNeeds: "optional RFORGE_SEMANTIC_SCHOLAR_API_KEY", LiveSmokeStatus: "covered by semantic-scholar-live-smoke", LicenseShareabilityPolicy: "API terms and field restrictions apply; cache only documented fields", Cacheability: "cache normalized metadata and graph refs with field list", ProvenanceFields: []string{"source", "query", "paper_id", "direction", "depth", "fields", "raw_ref"}},
		{ID: "crossref", Label: "Crossref", Kind: "scholarly-source", SupportedEntities: []string{"works", "doi", "references"}, RateLimitPolicy: "public polite API; request parameters must be recorded", AuthNeeds: "none", LiveSmokeStatus: "covered by source-live-smoke", LicenseShareabilityPolicy: "metadata terms vary by publisher; preserve DOI/source refs", Cacheability: "cache DOI/work lookups with request URL", ProvenanceFields: []string{"source", "query", "doi", "rows", "raw_ref"}},
		{ID: "arxiv", Label: "arXiv", Kind: "scholarly-source", SupportedEntities: []string{"preprints", "categories", "versions", "pdf", "source"}, RateLimitPolicy: "public API; polite pauses required for bulk workflows", AuthNeeds: "none", LiveSmokeStatus: "covered by source-live-smoke", LicenseShareabilityPolicy: "preprint metadata open; full-text/source license still captured per asset", Cacheability: "cache Atom responses and fetched asset checksums", ProvenanceFields: []string{"source", "query", "category", "arxiv_id", "version", "raw_ref"}},
		{ID: "pubmed", Label: "PubMed", Kind: "scholarly-source", SupportedEntities: []string{"articles", "pmid", "mesh", "biomedical metadata"}, RateLimitPolicy: "NCBI E-utilities policy; API key/email may raise limits", AuthNeeds: "optional NCBI API key/email", LiveSmokeStatus: "covered by biomedical-live-smoke", LicenseShareabilityPolicy: "metadata public; full text requires PMC/OA license checks", Cacheability: "cache query IDs and normalized metadata", ProvenanceFields: []string{"source", "query", "pmid", "retstart", "retmax", "raw_ref"}},
		{ID: "europepmc", Label: "Europe PMC", Kind: "scholarly-source", SupportedEntities: []string{"articles", "pmid", "pmcid", "full-text links", "license metadata"}, RateLimitPolicy: "public API; pagination/request params recorded", AuthNeeds: "none", LiveSmokeStatus: "covered by biomedical-live-smoke", LicenseShareabilityPolicy: "OA/full-text license fields must gate acquisition", Cacheability: "cache result pages and license metadata", ProvenanceFields: []string{"source", "query", "pmid", "pmcid", "page", "raw_ref"}},
		{ID: "nasa-ads", Label: "NASA ADS", Kind: "scholarly-source", SupportedEntities: []string{"bibcodes", "works", "authors", "references", "citations"}, RateLimitPolicy: "ADS API quota/rate limits; token usage must be recorded without secrets", AuthNeeds: "NASA ADS token required for live API", LiveSmokeStatus: "planned opt-in live smoke", LicenseShareabilityPolicy: "ADS terms apply; preserve bibcodes and requested fields", Cacheability: "cache normalized metadata and bibcode graph refs", ProvenanceFields: []string{"source", "query", "bibcode", "fields", "raw_ref"}},
		{ID: "doaj", Label: "DOAJ", Kind: "open-access-discovery", SupportedEntities: []string{"journals", "articles", "licenses", "oa candidates"}, RateLimitPolicy: "public API; attribution and request params recorded", AuthNeeds: "none", LiveSmokeStatus: "planned opt-in live smoke", LicenseShareabilityPolicy: "OA metadata is not automatic redistribution permission; record license", Cacheability: "cache metadata/license candidates", ProvenanceFields: []string{"source", "query", "doi", "license", "raw_ref"}},
		{ID: "core", Label: "CORE", Kind: "open-access-discovery", SupportedEntities: []string{"articles", "repositories", "full-text candidates", "licenses"}, RateLimitPolicy: "CORE API limits; key may be required", AuthNeeds: "CORE API key may be required", LiveSmokeStatus: "planned opt-in live smoke", LicenseShareabilityPolicy: "full-text links require explicit license/shareability review", Cacheability: "cache candidate metadata, not private payloads", ProvenanceFields: []string{"source", "query", "doi", "repository", "license", "raw_ref"}},
		{ID: "unpaywall", Label: "Unpaywall", Kind: "oa-lookup", SupportedEntities: []string{"doi", "oa status", "best oa location", "pdf url", "license"}, RateLimitPolicy: "email/contact expected for polite usage", AuthNeeds: "RFORGE_UNPAYWALL_EMAIL for live smoke", LiveSmokeStatus: "covered by source-live-smoke when email configured", LicenseShareabilityPolicy: "license and host type gate PDF acquisition", Cacheability: "cache DOI lookups and OA location metadata", ProvenanceFields: []string{"source", "doi", "oa_status", "license", "raw_ref"}},
		{ID: "zotero", Label: "Zotero", Kind: "reference-manager", SupportedEntities: []string{"items", "collections", "tags", "notes", "annotations", "attachments"}, RateLimitPolicy: "local export/import; no network by default", AuthNeeds: "local export file", LiveSmokeStatus: "not applicable; fixture round-trip tests", LicenseShareabilityPolicy: "notes/attachments/local paths require privacy/license review", Cacheability: "store imported normalized records and source metadata", ProvenanceFields: []string{"source", "format", "item_id", "collection", "attachment_redaction"}},
		{ID: "jabref", Label: "JabRef", Kind: "reference-manager", SupportedEntities: []string{"bibtex", "biblatex", "groups", "citation keys", "linked files"}, RateLimitPolicy: "local import/export; no network by default", AuthNeeds: "local BibTeX/BibLaTeX file", LiveSmokeStatus: "not applicable; fixture round-trip tests", LicenseShareabilityPolicy: "linked-file fields may expose private paths", Cacheability: "store normalized records and cleanup diff provenance", ProvenanceFields: []string{"source", "format", "citation_key", "group", "linked_file_redaction"}},
		{ID: "local", Label: "Local files", Kind: "local-import", SupportedEntities: []string{"pdf", "xml", "jats", "html", "text", "notes"}, RateLimitPolicy: "local filesystem only", AuthNeeds: "local path access", LiveSmokeStatus: "not applicable; local fixtures", LicenseShareabilityPolicy: "local files private until acquisition/shareability approved", Cacheability: "record checksums and local-only/redaction status", ProvenanceFields: []string{"source", "path", "checksum", "mime_type", "shareability"}},
	}}
}

func (r ConnectorCapabilityRegistry) ByID(id string) (ConnectorCapability, bool) {
	for _, capability := range r.Connectors {
		if capability.ID == id {
			return capability, true
		}
	}
	return ConnectorCapability{}, false
}
