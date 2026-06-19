package sources

import (
	"fmt"
	"net/url"
	"strings"
)

// OpenAccessResolvePlan is a legal, provenance-first plan for resolving DOI
// full-text candidates. It intentionally does not download full text or approve
// acquisition.
type OpenAccessResolvePlan struct {
	SchemaVersion      string                      `json:"schemaVersion"`
	DOI                string                      `json:"doi"`
	Sources            []OpenAccessResolveSource   `json:"sources"`
	HumanGates         []string                    `json:"humanGates"`
	UnsupportedSources []UnsupportedFullTextSource `json:"unsupportedSources"`
}

// OpenAccessResolveSource describes one legal OA discovery source.
type OpenAccessResolveSource struct {
	ID                 string   `json:"id"`
	Label              string   `json:"label"`
	Kind               string   `json:"kind"`
	Lookup             string   `json:"lookup"`
	Signals            []string `json:"signals"`
	LicensePolicy      string   `json:"licensePolicy"`
	AcquisitionPolicy  string   `json:"acquisitionPolicy"`
	ProvenanceRequired []string `json:"provenanceRequired"`
}

// UnsupportedFullTextSource documents intentionally unsupported sources.
type UnsupportedFullTextSource struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// BuildOpenAccessResolvePlan builds a DOI-specific legal OA source plan.
func BuildOpenAccessResolvePlan(doi string) (OpenAccessResolvePlan, error) {
	doi = normalizeSourceDOI(doi)
	if doi == "" {
		return OpenAccessResolvePlan{}, fmt.Errorf("doi is required")
	}
	return OpenAccessResolvePlan{
		SchemaVersion:      "1",
		DOI:                doi,
		Sources:            legalOpenAccessResolveSources(doi),
		HumanGates:         []string{"license review before storing/distributing full text", "acquisition approval before download", "privacy review before packaging", "ambiguous license/status requires human decision"},
		UnsupportedSources: UnsupportedFullTextSources(),
	}, nil
}

// LegalOpenAccessResolveSources returns generic legal OA resolver coverage.
func LegalOpenAccessResolveSources() []OpenAccessResolveSource {
	return legalOpenAccessResolveSources("<doi>")
}

// UnsupportedFullTextSources returns sources intentionally excluded from OA workflows.
func UnsupportedFullTextSources() []UnsupportedFullTextSource {
	return []UnsupportedFullTextSource{{ID: "sci-hub", Reason: "commonly provides unauthorized copyrighted full text; use legal OA sources and human acquisition gates instead"}}
}

func legalOpenAccessResolveSources(doi string) []OpenAccessResolveSource {
	qdoi := url.QueryEscape(doi)
	pdoi := url.PathEscape(doi)
	return []OpenAccessResolveSource{
		legalOASource("unpaywall", "Unpaywall", "doi-oa-resolver", "https://api.unpaywall.org/v2/"+pdoi+"?email=<contact>", []string{"is_oa", "oa_status", "best_oa_location", "url_for_pdf", "license", "host_type"}, "license and host_type gate PDF acquisition", "metadata lookup only; PDF download requires acquisition approval", []string{"doi", "oa_status", "license", "best_oa_location", "raw_ref"}),
		legalOASource("openalex", "OpenAlex OA locations", "scholarly-metadata", "https://api.openalex.org/works?filter=doi:"+qdoi, []string{"is_oa", "oa_status", "primary_location", "landing_page_url", "license"}, "primary_location license gates acquisition", "metadata lookup only; prefer publisher/repository OA URLs with license", []string{"doi", "openalex_id", "oa_status", "license", "raw_ref"}),
		legalOASource("europepmc", "Europe PMC", "biomedical-oa", "https://www.ebi.ac.uk/europepmc/webservices/rest/search?query=DOI:"+qdoi+"&format=json", []string{"pmid", "pmcid", "fullTextUrlList", "license", "open_access"}, "PMC/Europe PMC license fields gate acquisition", "biomedical full text only when OA/license metadata permits", []string{"doi", "pmid", "pmcid", "license", "raw_ref"}),
		legalOASource("pmc", "PubMed Central (PMC)", "biomedical-fulltext", "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pmc&term="+qdoi, []string{"pmcid", "jats xml availability", "license"}, "PMC article license gates redistribution", "JATS/PDF acquisition requires approval and license capture", []string{"doi", "pmcid", "license", "raw_ref"}),
		legalOASource("arxiv", "arXiv", "preprint-server", "https://export.arxiv.org/api/query?search_query=doi:"+qdoi, []string{"arxiv_id", "pdf", "versions", "categories"}, "arXiv license/non-exclusive terms captured per record", "preprint PDF/source acquisition still records checksum and license", []string{"doi", "arxiv_id", "version", "raw_ref"}),
		legalOASource("biorxiv-medrxiv", "bioRxiv / medRxiv", "preprint-server", "https://api.biorxiv.org/details/<server>/<date>/<date>", []string{"preprint_doi", "server", "category", "license"}, "preprint license captured per article", "date-range APIs filtered by DOI/query; acquisition requires approval", []string{"doi", "server", "license", "raw_ref"}),
		legalOASource("chemrxiv", "ChemRxiv", "preprint-server", "https://chemrxiv.org/engage/chemrxiv/public-api/v1/items?term="+qdoi, []string{"doi", "item_url", "license", "category"}, "ChemRxiv license field gates redistribution", "preprint acquisition requires approval", []string{"doi", "chemrxiv_id", "license", "raw_ref"}),
		legalOASource("doaj", "DOAJ", "oa-journal-index", "https://doaj.org/api/search/articles/"+pdoi, []string{"article links", "journal", "license"}, "DOAJ license metadata gates acquisition", "link candidate only until reviewed", []string{"doi", "license", "full_text_url", "raw_ref"}),
		legalOASource("core", "CORE", "repository-aggregator", "https://api.core.ac.uk/v3/search/works?q="+qdoi, []string{"repository", "downloadUrl", "license", "publisher"}, "repository license/shareability review required", "candidate URL only until human approval", []string{"doi", "repository", "license", "download_url", "raw_ref"}),
		legalOASource("semantic-scholar", "Semantic Scholar", "metadata-oa-hints", "https://api.semanticscholar.org/graph/v1/paper/DOI:"+pdoi, []string{"openAccessPdf", "externalIds", "venue"}, "openAccessPdf is a hint; license must be confirmed", "do not download from hints without license/acquisition review", []string{"doi", "s2_id", "openAccessPdf", "raw_ref"}),
		legalOASource("crossref", "Crossref license links", "metadata-oa-hints", "https://api.crossref.org/works/"+pdoi, []string{"license", "link", "content-version", "URL"}, "publisher license links gate acquisition", "metadata/license lookup only", []string{"doi", "license", "link", "raw_ref"}),
		legalOASource("openlibrary", "Open Library / Internet Archive", "book-archive", "https://openlibrary.org/search.json?q="+qdoi, []string{"ebook_access", "ia identifiers", "public scan availability"}, "only ebook_access=public or explicit open license can be treated as OA", "borrow/controlled-digital-lending items are not OA acquisition candidates", []string{"query", "work_id", "ebook_access", "raw_ref"}),
		legalOASource("software-heritage", "Software Heritage", "software-archive", "https://archive.softwareheritage.org/browse/search/?q="+qdoi, []string{"origin urls", "archive status", "SWHID"}, "source-code license must be reviewed at origin", "archive/reference metadata only; no dependency/import approval", []string{"origin", "swhid", "visit_status", "raw_ref"}),
	}
}

func legalOASource(id, label, kind, lookup string, signals []string, licensePolicy, acquisitionPolicy string, provenance []string) OpenAccessResolveSource {
	return OpenAccessResolveSource{ID: id, Label: label, Kind: kind, Lookup: strings.TrimSpace(lookup), Signals: signals, LicensePolicy: licensePolicy, AcquisitionPolicy: acquisitionPolicy, ProvenanceRequired: provenance}
}
