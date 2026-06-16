# Scholarly source connector backlog and terms review

ResearchForge keeps scholarly source integrations local-first and explicit about outbound data, credentials, and API terms. Normal tests must use fixtures or mock HTTP servers, not live source APIs.

## Connector backlog

### PubMed / Europe PMC

- Purpose: biomedical literature discovery, PMID/PMCID metadata, and open biomedical full-text leads.
- Status: Europe PMC search connector implemented as `rforge search --source europepmc`; PubMed/NCBI E-utilities search connector implemented as `rforge search --source pubmed`; normalized records preserve PMID, PMCID, MeSH terms, and Europe PMC full-text/OA links where source payloads provide them. Opt-in live smoke documentation and `make biomedical-live-smoke` cover the biomedical connectors.
- Terms review before further implementation:
  - confirm NCBI E-utilities usage policies, rate limits, API key expectations, and attribution requirements;
  - confirm Europe PMC API terms, rate limits, license metadata availability, and full-text link constraints before full-text acquisition.
- Outbound data: query terms, field filters, pagination tokens, optional tool/email/API-key identification.
- Credentials/config: optional NCBI API key/contact metadata through `RFORGE_PUBMED_API_KEY`, `RFORGE_PUBMED_TOOL`, and `RFORGE_PUBMED_EMAIL`; API keys are omitted from raw provenance refs.

### Semantic Scholar

- Purpose: citation-aware discovery, paper metadata, abstracts, author IDs, and citation/reference graph enrichment.
- Terms review before implementation:
  - confirm API key requirements, throttling, field availability, abstract/license constraints, and redistribution limits;
  - decide whether citation graph fields are cached or revalidated per project.
- Outbound data: query terms, paper IDs, requested fields, pagination tokens, optional API key.
- Credentials/config: optional or required Semantic Scholar API key depending on endpoint tier; `RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES` configures quota/transient retry attempts and the shared HTTP policy honors `Retry-After` on 429 responses.

### OpenAlex

- Purpose: broad scholarly metadata, concepts/domain mapping, related works, and citation graph discovery.
- Status: OpenAlex works search supports cursor pagination, resumable paginated imports via `rforge search import --resume-state state.json`, source filters via `--filter`, advanced filter flags (`--from-year`, `--to-year`, `--type`, `--open-access`, `--concept`), OA/license metadata, concept names/IDs plus topic/domain/field/subfield metadata for domain mapping, related-work OpenAlex IDs and `RelatedWorks` discovery records, author/institution entity search APIs, and citation/reference graph expansion through `rforge citations expand --source openalex`. Library imports merge duplicate DOI records across OpenAlex/Crossref/Semantic Scholar into one record while preserving additional identifiers and source refs.
- Outbound data: query terms, source filters, cursor tokens, row limits.
- Credentials/config: no API key; optional endpoint override/contact user agent.

### arXiv

- Purpose: preprint discovery and source/PDF acquisition leads.
- Status: arXiv search supports optional `--category`, preserves version/category/comment/journal-ref metadata, DOI when present, and adds PDF plus TeX source acquisition URLs. arXiv PDF/source assets can be fetched with `rforge pdf fetch-arxiv`, and TeX source files can be parsed with `rforge parse --parser tex --tex <file>`.
- Outbound data: query terms, category filters, max results, optional endpoint override.
- Credentials/config: no API key; use a contact user-agent if configured later.

### NASA ADS

- Purpose: physics and astronomy workflows, especially citation and bibliographic metadata for ADS-indexed literature.
- Terms review before implementation:
  - confirm token requirements, rate limits, acceptable cached metadata, and attribution/citation requirements;
  - validate whether abstracts or full-text links carry additional restrictions.
- Outbound data: query terms, bibcodes/DOIs, requested fields, pagination tokens, API token.
- Credentials/config: NASA ADS API token stored outside project manifests and redacted from provenance/output.

### DOAJ / CORE

- Purpose: open-access journal and repository discovery, OA status enrichment, and legal full-text leads.
- Terms review before implementation:
  - confirm DOAJ API attribution, throttling, and article metadata license;
  - confirm CORE API key requirements, rate limits, content/link redistribution constraints, and PDF URL handling.
- Outbound data: query terms, DOI/title filters, pagination tokens, optional API key.
- Credentials/config: CORE API key if required; DOAJ usually public but still subject to rate limits.

## Source-specific outbound data and credential requirements

| Source | Outbound user/project data | Credential/config | Must redact |
| --- | --- | --- | --- |
| OpenAlex | query terms, filters, pagination cursor | optional base URL/contact user agent | none by default |
| arXiv | query terms, category filters, max results, pagination metadata | optional base URL/contact user agent | none by default |
| Crossref | query terms, row limits, works filters, DOI refresh lookups, reference-list extraction | optional base URL/contact user agent | email if configured later |
| Unpaywall | DOI lookup, configured email | email required by policy | email |
| PubMed / Europe PMC | query terms, IDs, pagination tokens | optional NCBI key/contact metadata | API keys, email if configured |
| Semantic Scholar | query terms, paper IDs, requested fields | API key depending on endpoint; retry count via env | API key |
| NASA ADS | query terms, bibcodes/DOIs, requested fields | API token | API token |
| DOAJ / CORE | query terms, DOI/title filters, pagination tokens | CORE API key if used | API key |

## Implementation gate for new connectors

Before a connector moves from backlog to implementation:

1. add deterministic fixtures and mocked HTTP tests;
2. document request URL, query parameters, user-agent/contact behavior, retry/backoff behavior, and cache key rules;
3. normalize source output into `PaperRecord` or a dedicated domain record;
4. record request/response provenance without secrets;
5. add redaction tests for every credential-bearing configuration value.
