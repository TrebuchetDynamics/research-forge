# Scholarly source connector backlog and terms review

ResearchForge keeps scholarly source integrations local-first and explicit about outbound data, credentials, and API terms. Normal tests must use fixtures or mock HTTP servers, not live source APIs.

## Connector backlog

### PubMed / Europe PMC

- Purpose: biomedical literature discovery, PMID/PMCID metadata, and open biomedical full-text leads.
- Terms review before implementation:
  - confirm NCBI E-utilities usage policies, rate limits, API key expectations, and attribution requirements;
  - confirm Europe PMC API terms, rate limits, license metadata availability, and full-text link constraints.
- Outbound data: query terms, field filters, pagination tokens, optional tool/email/API-key identification.
- Credentials/config: optional NCBI API key; optional contact email/tool name if required by policy.

### Semantic Scholar

- Purpose: citation-aware discovery, paper metadata, abstracts, author IDs, and citation/reference graph enrichment.
- Terms review before implementation:
  - confirm API key requirements, throttling, field availability, abstract/license constraints, and redistribution limits;
  - decide whether citation graph fields are cached or revalidated per project.
- Outbound data: query terms, paper IDs, requested fields, pagination tokens, optional API key.
- Credentials/config: optional or required Semantic Scholar API key depending on endpoint tier.

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
| arXiv | query terms, max results, pagination metadata | optional base URL/contact user agent | none by default |
| Crossref | query terms, row limits, filters | optional base URL/contact user agent | email if configured later |
| Unpaywall | DOI lookup, configured email | email required by policy | email |
| PubMed / Europe PMC | query terms, IDs, pagination tokens | optional NCBI key/contact metadata | API keys, email if configured |
| Semantic Scholar | query terms, paper IDs, requested fields | API key depending on endpoint | API key |
| NASA ADS | query terms, bibcodes/DOIs, requested fields | API token | API token |
| DOAJ / CORE | query terms, DOI/title filters, pagination tokens | CORE API key if used | API key |

## Implementation gate for new connectors

Before a connector moves from backlog to implementation:

1. add deterministic fixtures and mocked HTTP tests;
2. document request URL, query parameters, user-agent/contact behavior, retry/backoff behavior, and cache key rules;
3. normalize source output into `PaperRecord` or a dedicated domain record;
4. record request/response provenance without secrets;
5. add redaction tests for every credential-bearing configuration value.
