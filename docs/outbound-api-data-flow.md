# Outbound API data flow

Local-first storage does not mean offline search. Queries and identifiers go to
selected source APIs, and results are stored locally. Normal tests use fixtures or
mock HTTP. Live calls are opt-in.

## Selected metadata integrations

This table covers common implemented paths, not the complete connector inventory.
`RawRef` is a local request reference, not a sanitized copy of every response.
[`PaperRecords`](../internal/sources/core/sources.go) copies it to
`SourceRef.RawPayloadRef` on normalized records.

| Integration / implementation | Outbound request fields | Credentials / identification | Retained reference |
| --- | --- | --- | --- |
| [OpenAlex](../internal/sources/adapters/openalex.go) | Works: `search`, `filter`, `per-page`, optional `cursor`. Entity search sends terms/limits. Work/graph lookups send work IDs or DOI filters. | Optional `RFORGE_OPENALEX_API_KEY` as `Authorization: Bearer …`. | Search fields or lookup/graph IDs and limits in `RawRef`. The configured key is absent from the request URL and `RawRef`. |
| [arXiv](../internal/sources/adapters/arxiv.go) | `search_query` combines terms and optional category, with `max_results`; ID lookup sends `id_list` and `max_results`. | No connector-specific credential or contact/email option in this CLI path. | Request parameters in `RawRef`. |
| [Crossref](../internal/sources/adapters/crossref.go) | Search sends `query`, `rows`, translated `filter` when applicable, and optional `cursor`; DOI lookup/reference retrieval sends the DOI in the path. | No connector-specific credential or contact/email option in this CLI path. | Search parameters or DOI/reference path in `RawRef`. |
| [Semantic Scholar](../internal/sources/adapters/semantic_scholar.go) | Search: `query`, `limit`, `fields`, optional `publicationDateOrYear` and `offset`. Graph: paper identifier and references/citations path, `limit`, `fields`. | Optional `RFORGE_SEMANTIC_SCHOLAR_API_KEY` in `x-api-key`. | Search query, limit, filter and offset, or graph identifier/direction/limit in `RawRef`; not the configured key or `fields` selection. |
| [PubMed](../internal/sources/adapters/pubmed.go) | ESearch sends `db=pubmed`, `term`, `retmax`, `retmode=json`. ESummary and EFetch send returned IDs, database and JSON/XML format. | Optional `RFORGE_PUBMED_API_KEY`, `RFORGE_PUBMED_TOOL`, `RFORGE_PUBMED_EMAIL` become URL parameters `api_key`, `tool`, `email` on these requests. | ESearch parameters in `RawRef`: `api_key` omitted, configured `email` and `tool` retained. |
| [Europe PMC](../internal/sources/adapters/europepmc.go) | `query` includes terms and any appended filter; `pageSize`, `format=json`, `resultType=core`, `cursorMark` (default `*`). | No connector-specific credential in this CLI path. | These request fields, including query/filter and cursor, in `RawRef`. |
| [Unpaywall](../internal/sources/adapters/unpaywall.go) | DOI in `/v2/…` path and required `email` URL parameter. | Configured email is required; CLI reads `RFORGE_UNPAYWALL_EMAIL`. | `SourceRef.RawPayloadRef` contains the DOI path without the configured email. This does not remove email from the outbound request. |

### Endpoints and existing checks

[CLI source configuration](../internal/cli/source_commands.go) selects default
provider base URLs or reads `RFORGE_OPENALEX_URL`, `RFORGE_ARXIV_URL`,
`RFORGE_CROSSREF_URL`, `RFORGE_SEMANTIC_SCHOLAR_URL`, `RFORGE_PUBMED_URL`,
`RFORGE_EUROPEPMC_URL` and `RFORGE_UNPAYWALL_URL` for the respective integrations.
`searchConnector`, `defaultSourceHTTPClient`, `defaultArXivHTTPClient` and
`defaultSemanticScholarHTTPClient` configure these search paths; `executeOA`
configures Unpaywall lookup. The CLI clients send `User-Agent: ResearchForge/dev`,
not an automatically supplied contact email.
The [source transport](../internal/sources/transport/http.go) builds URLs from the
configured base, path and query, then adds configured headers.

Existing offline tests cover these boundaries:

- [OpenAlex tests](../internal/sources/adapters/openalex_test.go): `TestOpenAlexConnectorOptionalAPIKey` checks anonymous/Bearer requests and unchanged URL/`RawRef`.
- [PubMed tests](../internal/sources/adapters/pubmed_test.go): `TestPubMedSearchAddsNCBIIdentificationWithoutLeakingAPIKeyToRawRef` checks outbound identification and retained email/tool.
- [Unpaywall tests](../internal/sources/adapters/unpaywall_test.go): successful lookup checks outbound email and the email-free reference; the missing-email case checks rejection.
- [Europe PMC](../internal/sources/adapters/europepmc_test.go), [arXiv](../internal/sources/adapters/arxiv_test.go) and [Crossref tests](../internal/sources/adapters/crossref_test.go) check request parameters and normalization.
- [CLI mock search](../internal/cli/cli_test.go) (`TestExecuteSearchSemanticScholarJSONWithMockHTTP`) checks `x-api-key`; [Semantic Scholar graph tests](../internal/sources/adapters/semantic_scholar_graph_test.go) check identifier paths and graph references.

## Other selected integrations

| Integration / implementation | Data crossing the boundary | Credentials / sharing concern |
| --- | --- | --- |
| [GROBID](../internal/parsing/adapters/grobid.go) | `Parse` posts PDF bytes as multipart `input` to the configured base URL plus `/api/processFulltextDocument`. The endpoint can be remote: there is no loopback restriction. | The adapter configures no authentication. PDF privacy/licensing review still applies. [Mock parser tests](../internal/parsing/adapters/grobid_test.go) cover the configured path, not remote-service qualification. |
| [R/metafor](../internal/analysis/core/runner.go) | Generated local input table/script passed to a local `Rscript` process. | No credentials by default. Local paths and analysis artifacts still need sharing review. |
| [OSS git clone](../internal/oss/clone.go) | Repository URL passed to `git clone`, which can contact the remote. | Git credentials can come from user configuration. Inspect credential-bearing URLs before sharing. |

## Provenance is not package redaction

Omitting a configured credential from a normal reference is not blanket log/error
sanitization. The source transport returns some underlying errors directly.
Do not assume every error, response or local artifact is redacted.
[Search batch output](../internal/cli/source_commands.go) includes results,
manifests, checkpoints, failures and per-source `raw/` files. Those raw search
artifacts are not shareable review packages (`writeSearchBatchRaw` writes DOI/title
rows, not complete HTTP responses).

Package filtering/redaction is a separate
[implementation](../internal/reviewpkg/package.go) and
[package contract](reproducible-review-package.md#redaction-requirements).
Inspect the package's redaction report and audit before sharing. Keep existing
[privacy/licensing requirements](threat-model.md) and
[human research/export gates](../internal/automation/policy.go), including human
privacy review approval before packaging and human approval for package export.
Metadata retrieval and provenance alone do not approve full-text acquisition,
scientific conclusions or redistribution.
