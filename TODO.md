# ResearchForge TODO

This is the end-to-end implementation checklist for ResearchForge. All production-code tasks must be developed with TDD: write the failing test first, make it pass, then refactor.

See also:

- [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md)
- [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)
- [ROADMAP.md](./ROADMAP.md)
- [SKILLS.md](./SKILLS.md)

## Global rules

- [x] Use red-green-refactor for every implementation slice.
- [x] Keep CLI and web GUI behavior backed by shared Go application services.
- [x] Record provenance for user-visible workflow changes and external-tool/API outputs.
- [x] Avoid live network dependencies in normal tests.
- [x] Use only legal, deterministic, minimal test fixtures.
- [x] Keep local clone workspaces, copyrighted PDFs, secrets, and private data out of git.
- [x] Prefer local-first operation; make heavyweight services optional until required.
- [x] Add ADRs only for hard-to-reverse, surprising trade-offs.

## 0. Repository and planning foundation

- [x] Create public GitHub repository under `TrebuchetDynamics/research-forge`.
- [x] Add product requirements document.
- [x] Add README.
- [x] Add development plan.
- [x] Add TDD-only project development skills.
- [x] Add issue templates.
- [x] Add pull request template with TDD receipt section.
- [x] Add contribution guide.
- [x] Add code of conduct if public contributions are expected.
- [x] Add license after owner decision. _(Resolved 2026-06-13: MIT, Copyright (c) 2026 Trebuchet Dynamics; approved by owner in issue #1, `make license-decision-approval-gate` reports `approved:true`; see `LICENSE`, `docs/license-decision.md`, and `docs/owner-decisions.md`.)_
- [x] Add `CONTEXT.md` glossary when first domain terms are finalized.
- [x] Add `docs/adr/` and ADR index when first ADR is accepted.
- [x] Reconcile PRD `rforge.project.yaml` example with current `rforge.project.toml` implementation via ADR or PRD update.
- [x] Standardize artificial photosynthesis as the main deterministic end-to-end test topic.

## 1. Milestone 0 — Go/CLI/Web GUI foundation

### 1.1 Go module and tooling

- [x] Add `go.mod` with module path.
- [x] Add initial package layout.
- [x] Add `cmd/rforge/main.go`.
- [x] Add build/version metadata variables.
- [x] Add `Makefile` or task runner.
- [x] Add `.gitignore` for builds, temp projects, SQLite files, and local clones.
- [x] Add formatting/lint/test commands.
- [x] Add GitHub Actions CI.
- [x] Add `go test ./...` gate.
- [x] Add `go vet ./...` gate.
- [x] Add `govulncheck` gate when feasible.

### 1.2 CLI skeleton

- [x] Choose CLI framework with ADR if needed.
- [x] Add root `rforge --help`.
- [x] Add `rforge version`.
- [x] Add global flags: `--project`, `--config`, `--json`, `--log-level`.
- [x] Add consistent JSON output envelope.
- [x] Add consistent error format and exit codes.
- [x] Add shell completion if framework supports it.
- [x] Add CLI command tests.
- [x] Add external service command group design.
- [x] Add `rforge service check <name>`.
- [x] Add `rforge service start <name>` where safe/local.
- [x] Add `rforge service stop <name>` where safe/local.

### 1.3 Project workspace

- [x] Define `ResearchProject` domain type.
- [x] Define project directory layout.
- [x] Add `rforge project create <path> --title <title>`.
- [x] Add `rforge project open/inspect <path>`.
- [x] Add `rforge project list`.
- [x] Write `rforge.project.toml` manifest.
- [x] Write `rforge.lock.json` lockfile.
- [x] Initialize local project data directories.
- [x] Initialize SQLite local database.
- [x] Add project path validation and traversal tests.
- [x] Add archive-safe project metadata.
- [x] Add repo-embedded `.researchforge` config creation when `rforge` runs inside an existing repository.
- [x] Default repo-embedded Research project path to `<repo>/research-forge/`.
- [x] Add discovery workflow for existing academic files, PDFs, notes, and research assets already present in a repository, with explicit provenance before import.

### 1.4 Manifest, lockfile, provenance

- [x] Define manifest schema and version.
- [x] Define lockfile schema and version.
- [x] Add manifest read/write tests.
- [x] Add lockfile read/write tests.
- [x] Add append-only provenance event log.
- [x] Add event IDs, timestamps, actor, action, target, inputs, outputs, and warnings.
- [x] Add event replay/query helpers.
- [x] Record project-create event.
- [x] Record CLI command provenance where relevant.
- [x] Add deterministic test clock/ID generator.

### 1.5 Storage foundation

- [x] Decide SQLite-first vs PostgreSQL-first; recommended SQLite-first.
- [x] Add storage interface.
- [x] Add SQLite implementation.
- [x] Add migration mechanism.
- [x] Add migration tests.
- [x] Add database backup before migrations.
- [x] Add storage health check.
- [x] Prepare PostgreSQL adapter seam for later.

### 1.6 Doctor command

- [x] Add `rforge doctor`.
- [x] Check Go/runtime version where useful.
- [x] Check project manifest/lockfile validity.
- [x] Check SQLite availability.
- [x] Check optional GROBID endpoint.
- [x] Check optional OpenSearch endpoint.
- [x] Check optional Qdrant endpoint.
- [x] Check optional R/metafor.
- [x] Output actionable JSON and human-readable results.

### 1.7 Local web GUI shell

- [x] Add Go + HTMX web GUI workspace/dependencies.
- [x] Add `rforge ui` local web entry point placeholder.
- [x] Add local web app shell placeholder.
- [x] Add project dashboard and generated-artifact placeholder.
- [x] Add background job abstraction.
- [x] Add view-model tests for dashboard state.
- [x] Ensure no core logic lives in browser components.

## 2. Milestone 1 — Scholarly metadata and library MVP

### 2.1 Source connector framework

- [x] Define `SourceConnector` interface.
- [x] Define `SourceQuery` domain type.
- [x] Define connector request/response provenance.
- [x] Add HTTP client with timeouts, retries, user-agent, and rate-limit behavior.
- [x] Add source response cache.
- [x] Add mocked HTTP test harness.

### 2.2 Paper library model

- [x] Define `PaperRecord`.
- [x] Add identifiers: DOI, arXiv ID, PMID, OpenAlex ID, Crossref ID, Semantic Scholar ID.
- [x] Add authors, title, abstract, year, venue, publisher, URLs, license/OA status.
- [x] Store raw source payload references.
- [x] Store provenance per source.
- [x] Add create/update/list/search library storage.
- [x] Add library CLI list command.
- [x] Add web GUI library view model.

### 2.3 OpenAlex connector

- [x] Add OpenAlex fixture responses.
- [x] Test query URL/parameters.
- [x] Normalize OpenAlex works into `PaperRecord`.
- [x] Store OpenAlex source metadata.
- [x] Add `rforge search --source openalex`.
- [x] Add pagination/limit behavior.
- [x] Add rate-limit/backoff handling.

### 2.4 arXiv connector

- [x] Add arXiv Atom fixtures.
- [x] Test query URL/parameters.
- [x] Normalize arXiv entries into `PaperRecord`.
- [x] Preserve arXiv versions and categories.
- [x] Add `rforge search --source arxiv`.

### 2.5 Crossref connector

- [x] Add Crossref fixture responses.
- [x] Test query URL/parameters.
- [x] Normalize Crossref works into `PaperRecord`.
- [x] Preserve DOI/reference metadata.
- [x] Add `rforge search --source crossref`.

### 2.6 Unpaywall connector

- [x] Add Unpaywall fixtures.
- [x] Test DOI lookup behavior.
- [x] Normalize OA status, license, best OA location, PDF URLs.
- [x] Add `rforge oa lookup <doi>`.
- [x] Ensure email/API configuration does not leak.

### 2.7 Additional scholarly source backlog

- [x] Add PubMed / Europe PMC connector backlog and terms review.
- [x] Add Semantic Scholar connector backlog and API terms review.
- [x] Add NASA ADS connector backlog for physics/astronomy workflows.
- [x] Add DOAJ / CORE connector backlog for open-access discovery.
- [x] Record source-specific outbound data and credential requirements.

### 2.8 Search strategy builder

- [x] Define saved search strategy model.
- [x] Add Boolean query construction tests.
- [x] Add synonym/concept expansion scaffold.
- [x] Add field-specific search representation.
- [x] Version search strategies in provenance.
- [x] Add watched-search schedule metadata.

### 2.9 Deduplication

- [x] Define duplicate scoring model.
- [x] Deduplicate exact DOI matches.
- [x] Deduplicate normalized arXiv IDs.
- [x] Deduplicate fuzzy title + author + year.
- [x] Merge source provenance safely.
- [x] Preserve all source identifiers.
- [x] Add duplicate review/report command.
- [x] Add manual duplicate merge command.
- [x] Add manual duplicate split command.
- [x] Add merge/split provenance events.
- [x] Add tests for false positive boundaries.

### 2.10 Imports and exports

- [x] Add BibTeX parser and fixtures.
- [x] Add RIS parser and fixtures.
- [x] Add CSV import.
- [x] Add JSON import.
- [x] Add BibTeX export golden tests.
- [x] Add RIS export golden tests.
- [x] Add CSV export golden tests.
- [x] Add JSON export golden tests.
- [x] Add `rforge import` and `rforge export`.

### 2.11 Search/library UI

- [x] Add search form view model.
- [x] Add search result table view model.
- [x] Add library table/detail view model.
- [x] Add web GUI search screen.
- [x] Add web GUI library screen.
- [x] Add loading/error/empty states.
- [x] Ensure UI calls shared services.

### 2.12 Automatic paper discovery

- [x] Define watched search domain type.
- [x] Add `rforge watch add`.
- [x] Add `rforge watch run`.
- [x] Add scheduled watched-search refresh runner semantics.
- [x] Add new-paper inbox storage.
- [x] Add `rforge inbox`.
- [x] Add `rforge fetch pdfs --open-access-only`.
- [x] Add approval workflow before automatic PDF downloads.
- [x] Record watched-search refresh provenance.

## 3. Milestone 2 — OSS repository intelligence MVP

- [x] Add `opensource/README.md`.
- [x] Add `.gitignore` rule for `opensource/clones/`.
- [x] Define `OSSRepositoryStudy` domain type.
- [x] Add repository name validation.
- [x] Add OSS registry storage.
- [x] Add `rforge oss add <owner/repo>`.
- [x] Add `rforge oss list`.
- [x] Add safe clone path resolution.
- [x] Add shallow clone command runner abstraction.
- [x] Add tests with local fake git repositories.
- [x] Add `rforge oss clone <owner/repo>`.
- [x] Add license-file detection.
- [x] Add `rforge oss license-check`.
- [x] Add study-note template.
- [x] Add `rforge oss note`.
- [x] Add topic scan metadata workflow.
- [x] Add `rforge oss scan --topic`.
- [x] Add `rforge oss report --area`.
- [x] Add OSS metadata refresh command.
- [x] Add scheduled OSS refresh metadata model.
- [x] Add stale/archived repository detection.
- [x] Add web-ready OSS dashboard view model.
- [x] Ensure external source code is not copied into production code without review.

## 4. Milestone 3 — Legal full-text, parsing, and indexing

### 4.1 Document assets and OA policy

- [x] Define `DocumentAsset`.
- [x] Add acquisition source, license, OA status, checksum, local path, and MIME type.
- [x] Add copyright/OA guard tests.
- [x] Add legal PDF URL selection from Unpaywall metadata.
- [x] Add `rforge pdf fetch --doi` with mocked HTTP tests.
- [x] Add manual local file import with local-only status.
- [x] Prevent accidental export of restricted assets.

### 4.2 GROBID parsing

- [x] Define parser adapter interface.
- [x] Add GROBID client with timeout/error handling.
- [x] Add mock TEI fixtures.
- [x] Parse title/authors/abstract/sections/references.
- [x] Generate deterministic section and passage IDs.
- [x] Record parser version/config in lockfile or event log.
- [x] Add `rforge parse --paper <id> --parser grobid`.
- [x] Add parser warning/error storage.

### 4.3 Indexing and retrieval

- [x] Define `ParsedDocument` and passage model.
- [x] Add local full-text index MVP with SQLite FTS or Bleve.
- [x] Add `rforge index rebuild`.
- [x] Add passage retrieval with paper/section/passage references.
- [x] Add `rforge retrieve --query`.
- [x] Add optional OpenSearch adapter seam.
- [x] Add optional Qdrant adapter seam.
- [x] Add embeddings adapter seam if needed.
- [x] Add web-ready PDF/section/passages view model.

### 4.4 Citation graph

- [x] Define citation graph model.
- [x] Add backward citation storage.
- [x] Add forward citation storage.
- [x] Add co-citation cluster scaffold.
- [x] Add bibliographic coupling scaffold.
- [x] Add research lineage view model.
- [x] Add citation graph export.
- [x] Add graph export format interoperability tests.
- [x] Add web GUI citation graph view model.

## 5. Milestone 4 — Screening workflow

- [x] Define screening stages: title/abstract, full text, final inclusion.
- [x] Define decisions: include, exclude, uncertain.
- [x] Define exclusion reason configuration.
- [x] Add reason validation tests.
- [x] Add reviewer attribution.
- [x] Add `rforge screen configure`.
- [x] Add `rforge screen decide`.
- [x] Add decision event history.
- [x] Add queue filtering by stage/status.
- [x] Add `rforge screen queue`.
- [x] Add conflict detection for multi-reviewer workflows.
- [x] Add uncertain queue.
- [x] Add PRISMA count generation from stored state/events.
- [x] Add `rforge prisma counts`.
- [x] Add CSV screening export/import.
- [x] Add ASReview-style active-learning prioritization scaffold.
- [x] Add web-ready screening queue and decision-panel view model.

## 6. Milestone 5 — Evidence extraction

- [x] Define extraction schema format.
- [x] Add schema validation.
- [x] Define `EvidenceItem`.
- [x] Require source support for accepted evidence.
- [x] Add support kinds: passage, table, figure, equation, dataset, citation.
- [x] Add `rforge extraction schema add`.
- [x] Add manual `rforge extract add`.
- [x] Add status transitions: suggested, accepted, rejected, corrected.
- [x] Preserve correction history.
- [x] Add evidence audit command for unsupported/weak evidence.
- [x] Add CSV/JSON/Markdown evidence export.
- [x] Add LLM suggestion adapter interface.
- [x] Add explicit LLM configuration and secret handling.
- [x] Add `rforge extract suggest`.
- [x] Ensure suggestions cannot become accepted without review.
- [x] Add web-ready evidence table and source-link view model.

## 7. Milestone 6 — Meta-analysis MVP

- [x] Define `AnalysisRun`.
- [x] Generate analysis input table from accepted evidence.
- [x] Add effect-size helper interface.
- [x] Implement first effect-size calculator.
- [x] Add known-result fixtures.
- [x] Generate R/metafor scripts.
- [x] Add opt-in R/metafor integration tests.
- [x] Capture R and package versions.
- [x] Run R/metafor through safe external command wrapper.
- [x] Store input snapshots, scripts, outputs, warnings, and checksums.
- [x] Register forest plot artifacts.
- [x] Register funnel plot artifacts where applicable.
- [x] Parse heterogeneity metrics.
- [x] Add meta-regression scaffold.
- [x] Add subgroup analysis scaffold.
- [x] Add publication bias check scaffold.
- [x] Add sensitivity-analysis scaffold.
- [x] Add `rforge analysis prepare`.
- [x] Add `rforge analysis run`.
- [x] Add `rforge analysis export`.
- [x] Add web-ready analysis setup/results view model for meta-analysis artifacts.

## 8. Milestone 7 — Report generation

- [x] Define report data model.
- [x] Add fixture project for report golden tests.
- [x] Build Markdown report skeleton.
- [x] Add citation table.
- [x] Add bibliography output.
- [x] Add evidence tables.
- [x] Add screening summary.
- [x] Add PRISMA diagram output.
- [x] Add reproducible notebook generation scaffold.
- [x] Add analysis result section.
- [x] Add forest/funnel plot references.
- [x] Add audit appendix.
- [x] Ensure report answers PRD audit questions.
- [x] Add HTML export.
- [x] Add LaTeX export scaffold.
- [x] Add `rforge report build`.
- [x] Add `rforge report audit`.
- [x] Add web-ready report/artifact browser view model.

## 9. Milestone 8 — Hardening and beta release

### 9.1 Quality and security

- [x] Add threat model document.
- [x] Add path traversal tests for project/archive/clone/document paths.
- [x] Add archive extraction safety tests.
- [x] Add external command argument safety tests.
- [x] Add API key redaction tests.
- [x] Add shareable-report redaction tests for local paths, reviewer names, and private notes.
- [x] Add per-project data-retention policy tests.
- [x] Add outbound API data-flow documentation.
- [x] Add external-tool version and container digest lockfile tests.
- [x] Add HTTP timeout tests.
- [x] Add bounded response-size tests where needed.
- [x] Add fuzz tests for import parsers.
- [x] Add race tests for background jobs.
- [x] Add dependency/license scan workflow.

### 9.2 Performance

- [x] Add benchmark datasets for 10, 1,000, and 100,000 records.
- [x] Benchmark deduplication.
- [x] Benchmark imports/exports.
- [x] Benchmark index rebuild.
- [x] Benchmark report generation.
- [x] Add cancellation tests for long jobs.
- [x] Add memory/allocation regression notes.
- [x] Ensure background jobs expose cancellation/progress for local web GUI responsiveness.

### 9.3 Documentation

- [x] Add user installation guide.
- [x] Add quickstart tutorial.
- [x] Add CLI command reference.
- [x] Add project format documentation.
- [x] Add external service setup docs.
- [x] Add CLI external service start/stop/check documentation.
- [x] Add privacy/copyright documentation.
- [x] Add developer setup guide.
- [x] Add architecture overview.
- [x] Add ADR index.
- [x] Add fixture policy documentation.
- [x] Add example open-data project.

### 9.4 Release packaging

- [x] Add cross-platform CLI build automation.
- [x] Add Go + HTMX web GUI smoke-check target.
- [x] Add checksums for artifacts.
- [x] Add SBOM/dependency metadata if feasible.
- [x] Add project archive/restore commands.
- [x] Add upgrade tests for project format.
- [x] Add release notes template.
- [x] Add install smoke test.
- [x] Prepare pre-alpha release.
- [x] Prepare alpha release.
- [x] Prepare beta release.
- [x] Prepare 1.0 release only after MVP success criteria pass.

## 10. Final MVP acceptance checklist

The MVP is complete when a researcher can:

- [x] Create a research project from the CLI.
- [x] Create/open a research project from the web GUI.
- [x] Search OpenAlex for a topic.
- [x] Search Crossref for a topic.
- [x] Search arXiv for a topic.
- [x] Deduplicate imported/searched records.
- [x] Retrieve legal open-access PDFs where available.
- [x] Parse papers with GROBID.
- [x] Search/retrieve exact passages.
- [x] Screen papers with include/exclude reasons.
- [x] Generate PRISMA counts.
- [x] Extract structured evidence into tables.
- [x] Link evidence to source passages/tables/figures/equations.
- [x] Run a basic meta-analysis.
- [x] Study and catalog relevant OSS repositories from the CLI.
- [x] View OSS repository studies in the web GUI.
- [x] View CLI-generated papers, meta-analysis outputs, PRISMA/citation diagrams, and report artifacts in the web GUI.
- [x] Export a reproducible report with citations and provenance.
- [x] Reproduce the report from stored manifest, lockfile, provenance, and project data.

## Current completion audit

See [docs/todo-completion-audit.md](docs/todo-completion-audit.md) for the current prompt-to-artifact checklist, remaining owner/license blocker, Go + HTMX implementation tracker, and validation evidence for this TODO list.

## 11. Continuous validation and post-1.0 backlog

The MVP checklist above is complete. This section tracks follow-on validation and future enhancements. Items in this backlog section are post-1.0 work and are intentionally exempt from the owner-decision coverage audit (see `isTodoBacklogHeading` in `internal/cli`), so they do not need a tracking decision/issue to remain unchecked.

### Completed validation hardening

- [x] Add full research-pipeline CLI e2e: create → import → deduplicate → screen → PRISMA → extract evidence → meta-analysis → report, offline and deterministic (`internal/cli/full_pipeline_e2e_test.go`).
- [x] Add report reproducibility e2e asserting byte-identical rebuilds from stored project state.
- [x] Add import/export round-trip e2e preserving identifiers across formats and projects.
- [x] Add project archive/restore round-trip e2e preserving manifest and library.

### Ordered Meta-analysis spine implementation plan

This is the canonical build order for the post-1.0 super-tool work. The thematic OSS-derived backlogs below remain the evidence/source pool, but implementation should proceed through this sequence so ResearchForge reaches the first done artifact: a Reproducible review package for meta-analysis authors.

#### Phase 0 — Product blueprint and acceptance gates

- [x] Write the ResearchForge super-tool blueprint: modules, CLI command families, HTMX pages, storage files/tables, provenance events, validation gates, and OSS adapter dispositions (`docs/meta-analysis-spine-blueprint.md`).
- [x] Write the phased Meta-analysis spine roadmap and explicitly defer broad research-cockpit-only features until the Reproducible review package is audit/replay safe (`docs/meta-analysis-spine-roadmap.md`).
- [x] Define the `rforge forge` state machine: question draft, source plan, import plan, dedupe review, full-text acquisition, parser arbitration, indexing, screening, extraction, analysis, package/export, archive, and reopen/resume (`docs/rforge-forge-state-machine.md`).
- [x] Define Reproducible review package acceptance criteria: required files, manifests, checksums, redaction report, replay command, audit report, and failure modes (`docs/reproducible-review-package.md`).
- [x] Add an acceptance-test matrix mapping every phase to unit tests, CLI e2e tests, handler tests, Playwright paths, screenshot coverage, provenance assertions, and package replay checks (`docs/meta-analysis-spine-acceptance-matrix.md`).

#### Phase 1 — Question, protocol, and source plan

- [x] Add a research-question compiler for PICO/PECO/SPIDER/freeform questions that drafts source-specific query plans, inclusion/exclusion criteria, extraction schema seeds, and reviewer prompts without auto-accepting claims (`internal/protocol/compiler.go`, `rforge protocol compile`).
- [x] Add a source-planning cockpit and CLI plan preview for OpenAlex, Semantic Scholar, Crossref, arXiv, PubMed/Europe PMC, NASA ADS, DOAJ/CORE, Unpaywall, Zotero/JabRef, and local imports (`rforge protocol plan-sources`, `/sources`).
- [x] Add connector capability registry records for supported entities, rate limits, auth needs, live-smoke status, license/shareability policy, cacheability, and provenance fields (`protocol.DefaultConnectorCapabilityRegistry`, `rforge protocol capabilities`).
- [x] Add API drift/live-smoke snapshot storage and dashboard alerts for all source connectors (`protocol.ConnectorLiveSmokeSnapshot`, `rforge protocol live-smoke-snapshot`, `/connectors`).
- [x] Add query-expansion suggestion records from KeyBERT/SciSpaCy/LLM assistants, requiring source text links and reviewer approval before a source plan changes (`protocol.DraftQueryExpansionSuggestions`, `protocol.ApplyApprovedQueryExpansions`, `rforge protocol suggest-expansions`).

#### Phase 2 — Library import, identity, and deduplication

- [x] Add Zotero/JabRef reference-manager fidelity work: collections/groups, tags, notes, annotations, citation keys, BibTeX/BibLaTeX cleanup diffs, and linked-file privacy checks (`ImportZoteroRDF`, `ImportBibTeX`, `ReferenceManagerFidelityReport`).
- [x] Add a reference-manager interchange fidelity matrix across BibTeX, RIS, CSL-JSON, Zotero RDF, Better BibTeX citation keys, tags, notes, collections, and redacted attachments (`BuildReferenceManagerInterchangeMatrix`, `rforge library reference-manager-matrix`).
- [x] Add source-fusion identity resolution for DOI, arXiv, PMID, PMCID, OpenAlex, Semantic Scholar, Crossref, Zotero, and ADS bibcode identifiers with explainable match rules (`ResolveIdentityClusters`, `rforge library identity-resolve`).
- [x] Add reversible merge/split decisions and conflict records for identity clusters (`IdentityDecision`, `IdentityConflictRecord`, `rforge library identity-decision`, `rforge library identity-conflicts`).
- [x] Add a revtools-inspired visual dedupe/cluster review screen with exportable decision history and PRISMA/audit provenance (`/dedupe`, `rforge library identity-decision log`, `data/identity-decisions.jsonl`).

#### Phase 3 — Legal full-text acquisition

- [x] Add DOAJ/CORE OA discovery adapters and compare full-text candidates from Unpaywall, DOAJ, CORE, PubMed/Europe PMC/PMC, arXiv, and local files (`NewDOAJConnector`, `NewCOREConnector`, `CompareOpenAccessCandidates`, `rforge oa candidates`).
- [x] Add legal acquisition queues with OA/license status, source URL, expected local path, restricted/shareable flags, and explicit reviewer approval before download or archive inclusion (`LegalAcquisitionQueue`, `rforge oa acquisition-queue`, `rforge oa acquisition-approve`).
- [x] Add privacy/licensing review gates for imported attachments, notes, annotations, local paths, copyrighted PDFs, and shareable reports (`PrivacyLicensingReview`, `rforge oa privacy-review`, `rforge oa privacy-approve`).
- [x] Add PMCID/PMID linking, structured biomedical full-text import, supplementary-file discovery, and biomedical live drift smoke tests (`LinkPMCIDPMID`, `ImportStructuredBiomedicalFullText`, `NewBiomedicalLiveDriftSmokeSnapshot`, `rforge pdf import-biomedical`).

#### Phase 4 — Parser arbitration and reference normalization

- [x] Add parser-output license/provenance manifests for GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse-style metadata, and Anystyle outputs (`ParserRunManifest`, `DefaultParserOutputPolicies`, `rforge parse manifest-policies`).
- [x] Add a multi-engine parser arbitration layer that scores parser output per field, compares raw text/offsets/warnings, and records why one output was accepted (`ArbitrateParserOutputs`, `rforge parse arbitrate`).
- [x] Extend parsed-document models with stable offsets, layered annotations, citation spans, parser confidence, and multi-parser reconciliation outputs (`EnrichParsedDocumentModel`, `ParserReconciliationOutput`, parser adapters, arbitration reports).
- [x] Add reviewer-persistent parsed-reference adjudication for accept/correct/reject/defer decisions (`ReferenceAdjudication`, `rforge parse adjudicate-ref`, `rforge parse adjudicated-refs`).
- [x] Normalize parsed references against Crossref, OpenAlex, Semantic Scholar, and ADS while preserving raw strings, confidence, provenance, and ambiguity queues (`NormalizeParsedReferences`, `NewNASAADSConnector`, `rforge parse normalize-refs --source ads`).
- [x] Import full bibliography-to-citation-graph edges and link citation spans back to passages and report evidence (`ImportParsedBibliography`, `rforge citations import-bibliography`).

#### Phase 5 — Retrieval, graph, and domain-map layer

- [x] Add OpenSearch mapping-version lockfiles, bulk indexing with partial-failure provenance, highlighted passage results, and opt-in OpenSearch integration tests (`OpenSearchMappingVersion`, `OpenSearchBulkReport`, `TestOptInOpenSearchIntegration`).
- [x] Add Qdrant adapter hardening: embedding-provider registry, compliance profiles, model/dimension locks, payload privacy, vector-index invalidation, and opt-in Qdrant integration tests (`DefaultEmbeddingProviderRegistry`, `QdrantRebuildReport`, `data/qdrant.vector.lock.json`, `TestOptInQdrantIntegration`).
- [x] Add calibrated hybrid retrieval tuning files with lexical/vector/backend weights, evaluation scores, selected configuration, and query-set checksums (`CalibrateHybridRetrieval`, `rforge retrieve tune-hybrid`).
- [x] Add retrieval benchmarks comparing SQLite FTS, OpenSearch, Qdrant vector search, and hybrid ranking on deterministic passage-query fixtures (`RunRetrievalBenchmark`, `rforge retrieve benchmark`).
- [x] Add BERTopic-style domain-map artifacts with representative papers/passages, reviewer-edited labels, topic merge/split history, model settings, and citation graph links (`BuildDomainMapArtifact`, `rforge citations domain-map`).
- [x] Add accessible/no-JS citation/domain graph views: filtered node tables, edge lists, graph summaries, keyboard navigation, and exportable graph reports (`BuildAccessibleGraphView`, `rforge citations accessible-view`).

#### Phase 6 — Screening and review assistance

- [x] Persist ASReview-style active-learning runs with input hashes, seed decisions, ranking method, ranked output, reviewer progress, stopping diagnostics, and adjudication state (`BuildActiveLearningRun`, `rforge screen active-run`).
- [x] Add balanced exploration/exploitation ranking policies, richer recall/effort simulation, and sensitivity diagnostics (`PrioritizeBalancedRecords`, `SimulateRecallEffort`, `ActiveLearningSensitivityDiagnostics`, `rforge screen sensitivity`).
- [x] Add reviewer assignment, conflict/adjudication panels, uncertain queues, and exportable audit bundles for screening decisions (`AssignReviewers`, `BuildConflictAdjudicationPanel`, `UncertainQueue`, `BuildScreeningAuditBundle`, `rforge screen assign|panel|audit-bundle`).
- [x] Add RobotReviewer-inspired risk-of-bias schema templates and evidence-suggestion queues with exact support text, uncertainty, model/version metadata, and reviewer decisions (`DefaultRiskOfBiasSchemaTemplates`, `DraftRiskOfBiasSuggestionQueue`, `ReviewRiskOfBiasSuggestion`, `rforge evidence risk-bias-*`).
- [x] Add HTMX screening cockpit views for active-learning queues, uncertainty/exploration flags, progress metrics, stopping diagnostics, and audit-bundle links (`/screening`, `BuildScreeningCockpitState`, `NewScreeningCockpitHandler`).

#### Phase 7 — Evidence extraction and gap analysis

- [x] Add an evidence extraction grid linking every field to passage/table/figure/equation support, parser offsets, PDF view, reviewer status, correction history, and downstream analysis inclusion (`BuildExtractionGrid`, `rforge evidence grid`).
- [x] Add scientific entity extraction suggestions with passage offsets, abbreviation resolution, entity-link candidates, confidence, model provenance, and reviewer decisions (`DraftScientificEntitySuggestions`, `ReviewScientificEntitySuggestion`, `rforge evidence entity-suggest|entity-review`).
- [x] Add LLM-assisted but citation-locked extraction/report-prose suggestions that remain unaccepted until reviewer approval (`DraftCitationLockedSuggestions`, `ReviewCitationLockedSuggestion`, `rforge evidence citation-suggest|citation-review`).
- [x] Add an evidence gap analyzer for missing outcomes, missing comparators, unsupported claims, incomplete full-text acquisition, and analysis-input readiness (`AnalyzeEvidenceGaps`, `rforge evidence gaps`).
- [x] Add per-passage provenance and parser/version/source-offset links in generated reports and package audits (`PassageProvenance`, `BuildPassageProvenanceFromParsedDocuments`, `rforge report build --parsed`).

#### Phase 8 — Statistical analysis and method comparison

- [x] Add additional effect-size calculators beyond standardized mean difference, log odds ratio, and risk ratio (`MeanDifference`, `RiskDifference`, `FisherZCorrelation`; `rforge analysis prepare --effect mean-difference|risk-difference|fisher-z-correlation`).
- [x] Improve subgroup analysis and meta-regression UX beyond direct CLI value entry (`ModeratorPreviewFromEvidence`, `SubgroupValuesFromEvidence`, `MetaRegressionValuesFromEvidence`, `rforge analysis moderators`, `--from-evidence`).
- [x] Add influence diagnostics, richer sensitivity artifacts, and publication-bias tests beyond Egger-style regression (`InfluenceDiagnostics`, richer `LeaveOneOut`, `BeggRankCorrelation`, `rforge analysis influence`, `publication-bias --method begg`).
- [x] Add publication-ready analysis artifact manifests for forest/funnel plots, plot settings, checksums, R/metafor scripts, engine versions, warnings, and report embedding metadata (`AnalysisArtifactManifest`, `NewAnalysisArtifactManifest`, `run1-artifact-manifest.json`).
- [x] Add PyMARE-style secondary engine comparison reports against metafor fixtures, including environment locks, model-setting parity, warning capture, output deltas, and disagreement handling (`EngineComparisonReport`, `BuildPyMAREFixtureResult`, `rforge analysis engine-compare`).
- [x] Add a method-comparison workbench for parser choices, retrieval backends, screening rankers, effect-size models, and publication-bias diagnostics (`MethodComparisonWorkbench`, `DefaultMethodComparisonWorkbench`, `rforge analysis method-workbench`).

#### Phase 9 — Report, traceability, and Reproducible review package

- [x] Add citation-to-evidence trace views from every report claim back to effect-size rows, accepted evidence, passages, parser outputs, PDFs, reference-manager items, source API records, and raw request/response metadata (`CitationEvidenceTraceView`, `BuildCitationEvidenceTraceView`, `rforge report trace`).
- [x] Add a claim traceability panel that blocks final export for unresolved or weakly supported generated paragraphs, tables, or figures (`ClaimTraceabilityPanel`, `GuardFinalReportExport`, `rforge report claim-panel|final-export`).
- [x] Add the Reproducible review package format with project manifest, lockfiles, source query plans, dedupe decisions, parser manifests, screening audit, extraction schema, accepted evidence, analysis artifacts, report outputs, redaction policy, and checksums (`reviewpkg.Manifest`, `reviewpkg.Create`, `rforge package create`).
- [x] Add package replay/audit commands that verify all checksums, lockfiles, analysis inputs, report outputs, redactions, and provenance links (`reviewpkg.Audit`, `reviewpkg.Replay`, `rforge package audit|replay`).
- [x] Add package archive/restore compatibility tests that prove a package can be moved, reopened, audited, and replayed without private local state (`reviewpkg.Archive`, `reviewpkg.Restore`, `rforge package archive|restore`).

#### Phase 10 — HTMX cockpit and one-command Forge workflow

- [x] Add the Forge home timeline showing active project, current state, provenance events, blocked review gates, background jobs, and next safe actions with CLI-equivalent commands (`/forge`, `BuildForgeHomeState`, `NewForgeHomeHandler`).
- [x] Add HTMX workbenches for source planning, import/dedupe, legal acquisition, parser arbitration, retrieval tuning, screening, evidence extraction, meta-analysis, report traceability, research map, connector health, and reproducibility/export (`/workbenches`, generic workbench routes, CLI-equivalent commands).
- [x] Add dashboard information architecture: routes, partial endpoints, view models, no-JS fallbacks, background jobs, and ownership boundaries (`/architecture`, `DashboardInformationArchitecture`, `BuildDashboardInformationArchitecture`).
- [x] Add dashboard permissions/privacy model for local-only paths, copyrighted PDFs, reviewer notes, credentials, embeddings, cache files, and shareable report fields (`/privacy`, `DashboardPrivacyModel`, `BuildDashboardPrivacyModel`).
- [x] Add the one-command `rforge forge` guided workflow with review gates at every irreversible scientific or data-sharing decision (`internal/forge`, `rforge forge init/status/next/approve/reopen/replay`, persisted state, blocked gates, transition provenance).
- [x] Expand Playwright and screenshot coverage for all Meta-analysis spine cockpit screens and no-JS fallbacks (`internal/webui/playwright_spine_e2e_test.go`, expanded JS-disabled screenshots).

#### Phase 11 — Broader research cockpit expansion after the spine

- [x] Add project knowledge graph queries that merge collections/tags, concepts, citations, parsed references, evidence, screening decisions, analysis runs, report claims, and provenance events (`internal/knowledge`, `rforge knowledge query`).
- [x] Add live research-map cockpit features for concept maps, citation neighborhoods, retrieval clusters, evidence coverage, and snapshot export (`/map`, `/map/snapshot.json`, `BuildResearchMapCockpitState`).
- [x] Add lab-notebook timeline views for all human and automated workflow events (`/notebook`, `/notebook/snapshot.json`, `BuildLabNotebookTimelineState`).
- [x] Add OSS-inventory-to-roadmap reports that group `nextSlice` entries by area and detect TODO coverage gaps for new inventory notes (`BuildInventoryRoadmapReport`, `rforge oss inventory-roadmap`).
- [x] Add cross-tool benchmarks for discovery recall, dedupe precision, parser accuracy, reference normalization, retrieval quality, screening effort savings, and report/package reproducibility (`internal/benchmarks`, `rforge benchmark cross-tool`).

### Post-1.0 backlog

- [x] Make `import` resilient to duplicate identifiers: skip in-file/in-store duplicates and no-identifier records and report them, instead of aborting the whole import and leaving partial state (parsers skip+count unstorable records, `Store.ImportRecords` skips+reports duplicates; merging stays in `duplicate merge`). See `docs/superpowers/specs/2026-06-13-import-duplicate-handling-design.md`.
- [x] Add a web GUI end-to-end test that builds view models from a CLI-generated project and serves them through the `internal/webui` handlers, tying CLI-produced artifacts to the cockpit pages (`internal/webui.BuildLibraryViewModel`/`BuildArtifactDashboardState` read the project's library, screening, and analysis state; e2e serves the library and artifacts handlers via httptest in `internal/webui/builders_test.go`).
- [x] Add opt-in live-service smoke tests for source connectors (OpenAlex, arXiv, Crossref, Unpaywall) behind an environment guard, mirroring the external e2e pattern (`TestOptInLiveSourceConnectorSmoke` runs live queries under `RFORGE_RUN_LIVE_SOURCE_SMOKE=1`, per-connector subtests, Unpaywall gated on `RFORGE_UNPAYWALL_EMAIL`; skips cleanly otherwise — `internal/sources/live_smoke_test.go`).
- [x] Add an opt-in GROBID parse e2e against a real GROBID endpoint behind an environment guard (`TestOptInGROBIDRealEndpointParse` parses a real PDF via a live GROBID server under `RFORGE_GROBID_E2E_URL`/`RFORGE_GROBID_E2E_PDF` and skips cleanly otherwise; `internal/parsing/grobid_e2e_test.go` also covers the unreachable-endpoint error path locally).
- [x] Add an opt-in R/metafor `analysis run` e2e using a real Rscript to complement the deterministic `FakeRunner` path (`internal/analysis.RscriptRunner` executes generated metafor scripts via `Rscript`; `TestOptInRMetaforIntegration` runs real R/metafor under `RFORGE_RUN_R_METAFOR_INTEGRATION=1` and skips cleanly when `Rscript` is absent).
- [x] Add a multi-reviewer screening e2e exercising conflict detection and the uncertain queue through the CLI (`internal/cli/screening_e2e_test.go`; added the enabling `rforge screen conflicts --stage <stage>` command over the existing `MemoryStore.Conflicts`).
- [x] Add OSS inventory governance commands over the inventory manifest: `rforge oss inventory-report [--area]` (deterministic Markdown ecosystem report), `rforge oss inventory-drift` (manifest-vs-note metadata drift + unreferenced-note detection), `rforge oss inventory-refresh --source github [--base-url]` (in-place stars/forks/archived/pushedAt/licenseSPDX refresh via a GitHub-compatible API with mocked-HTTP tests), and `rforge oss inventory-policy [--stale-after 18mo] [--now]` (archived/missing-license/copyleft-disposition/stale governance checks). See `internal/oss/inventory_{report,drift,refresh,policy}.go` and tests.
- [x] Add a PaperMage JSON parser adapter wired into `rforge parse --paper <id> --parser papermage --papermage <file>`, normalizing PaperMage output into `ParsedDocument` (`internal/parsing/papermage.go` + test).
- [x] Extend OpenAlex search with `--from-year/--to-year/--type/--open-access/--concept` filters and `--resume-state` paging, add `rforge library refresh-crossref`, and add a `crossref` direction to `citations expand` (`internal/sources/openalex.go`, `internal/cli/{source,library}_commands.go`).
- [x] Make cross-source `import` merge duplicate-identifier records and union their source refs/identifiers instead of dropping them (`internal/library/import_cross_source_dedupe_test.go`, `internal/library/store.go`).
- [x] Serve a real local Go + HTMX research cockpit from `rforge ui` instead of the placeholder status command: `webui.NewRouter` wires an `http.Server` with `--addr`/`RFORGE_UI_ADDR` (default `:8080`) so multiple dashboards run concurrently on different ports, one per research folder; `rforge --json ui` reports the resolved address/project/routes without binding (`internal/webui/server.go`, `internal/cli/ui_command.go`). Static assets are embedded.
- [x] Read papers in the browser: `/papers` and `/papers/{id}` render parsed sections/passages from `<project>/parsed/` next to a natively rendered project-local PDF (`/papers/{id}/pdf`) when present under `<project>/documents/`, falling back to parsed text only; paper ids are allow-list validated against path traversal (`internal/webui/papers.go`).
- [x] Surface meta-analysis detail (I²/τ²/Q heterogeneity + forest/funnel plot availability + warnings) on `/artifacts` from `<project>/analysis/<run>-result.json` (`internal/webui/builders.go`, `internal/webui/webui.go`).
- [x] Render the knowledge/citation graph on `/artifacts` as a deterministic server-side SVG built from `<project>/data/citation-graph.json`, with nodes linking to their `/papers/{id}` reading page and a seam for vendored interactive JS later (`internal/webui/builders.go`, `internal/webui/webui.go`).
- [x] Add an in-browser project switcher (`/projects/switch`, `/projects/active`) that repoints the active research folder at runtime, complementing per-instance `--addr` ports (`internal/webui/server.go`).
- [x] Add an interactive citation/knowledge graph: a `/artifacts/graph.json` data endpoint and a vendored, dependency-free, offline `citation-graph.js` (embedded under `internal/webui/static/`) that fetches it and renders an SVG with pan (drag), zoom (wheel / `+`/`-`), and click-through (node → its `/papers/{id}` page), progressively enhancing the server-rendered SVG fallback so JS-disabled clients keep the static graph (`internal/webui/graph_data.go`, `internal/webui/static/citation-graph.js`).
- [x] Add an opt-in Playwright browser e2e (`internal/webui/playwright_e2e_test.go`, build tag `playwright_e2e`) that boots the real dashboard router and drives headless Chromium through the shell nav, papers list, parsed full-text detail, and the JS-enhanced interactive citation graph (clicking a node navigates to its paper page). Excluded from the default `go build/vet/test ./...` gate by the build tag, gated by `RFORGE_RUN_PLAYWRIGHT=1`, and skips cleanly when the Playwright driver/browsers are absent. Run via `make web-gui-e2e`.
- [x] Vendor htmx locally (`internal/webui/static/htmx.min.js`, served from embedded assets) and load it from `/assets/htmx.min.js` instead of an `unpkg.com` CDN URL carrying an invalid SRI `integrity` hash that blocked htmx from executing at all — restoring offline, local-first operation of every HTMX flow (search, library/artifacts refresh, project create/open, project switcher).
- [x] Expand the Playwright e2e into subtests covering the native PDF embed on the paper page, interactive-graph keyboard zoom (`+` updates the viewport transform), the in-browser HTMX project switcher (switch folder → library reflects it), and the no-JS static-SVG graph fallback (interactive enhancement absent without JavaScript). Added Go unit tests asserting the shell loads vendored htmx (no CDN/SRI) and the asset is served (`internal/webui/assets_test.go`).
- [x] Add a screenshot regression test: a deterministic, default-gate-unit-tested pixel-diff helper (`internal/webui/imagediff.go`, `imagediff_test.go`) plus an opt-in Playwright test (`internal/webui/screenshot_e2e_test.go`, build tag `playwright_e2e`) that captures the shell, papers list, paper detail, and artifacts pages in a JS-disabled, fixed-viewport context and compares them against committed golden PNGs (`internal/webui/testdata/screenshots/`) with an anti-aliasing tolerance. Goldens are generated against the pinned Chromium via `make web-gui-screenshots-update`; `make web-gui-e2e` runs all browser tests including the comparison. Failure artifacts (`*.actual.png`) are git-ignored.
- [x] Add a dedicated GitHub Actions workflow for the browser e2e (`.github/workflows/playwright-e2e.yml`): installs Go and the pinned Chromium via the `playwright-go` CLI (`install --with-deps`, browser cache keyed on the pinned version) and runs the functional `TestPlaywrightDashboard` e2e under the `playwright_e2e` build tag with `RFORGE_RUN_PLAYWRIGHT=1`. The default `go test ./...` CI job stays browser-free (the e2e is build-tagged out). The screenshot regression is intentionally excluded from CI because its goldens are pinned to the generating machine's font rendering, so it remains a local tool.

### Open-source inventory feature-gap backlog

These follow-on tasks come from the committed open-source study notes in `opensource/inventory/` and should be implemented adapter-first, with fixture/fake-backed tests before any live integrations.

- [x] Add Zotero RDF import/export with collection hierarchy, Better BibTeX citation-key preservation, annotation mapping, and privacy-redacted attachment metadata (`ImportZoteroRDF`, `ExportZoteroRDF`, `opensource/inventory/zotero.md`).
- [x] Persist ASReview-style active-learning prioritization runs with dataset/input hashes, seed decisions, ranking method, ranked output, reviewer progress metrics, stopping diagnostics, and adjudication state (`BuildActiveLearningRun`, `rforge screen active-run`, `opensource/inventory/asreview.md`).
- [x] Add a parser quality/confidence report that compares GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse-style metadata, and Anystyle reference output without auto-accepting conflicting fields (`BuildParserQualityReport`, `rforge parse quality`, `opensource/inventory/{grobid,s2orc-doc2json,papermage,cermine,science-parse,anystyle}.md`).
- [x] Normalize parsed bibliography references against Crossref, OpenAlex, and Semantic Scholar while preserving raw reference text, parser provenance, match confidence, and reviewer-visible ambiguity queues (`NormalizeParsedReferencesAcrossConnectors`, parser name/version on matches, `opensource/inventory/{grobid,s2orc-doc2json,anystyle}.md`).
- [x] Import full bibliography-to-citation-graph edges from parsed documents and link citation-span offsets back to passages and report evidence (`ImportParsedBibliographies`, `rforge citations import-bibliography --parsed-dir`, `opensource/inventory/{grobid,s2orc-doc2json}.md`).
- [x] Add per-passage provenance and parser/version/source-offset links in generated reports so evidence claims remain traceable after parser upgrades (`BuildPassageProvenanceFromParsedDocuments`, report passage provenance table, `opensource/inventory/{grobid,papermage}.md`).
- [x] Extend the PaperMage/S2ORC-style parsed-document model with layered annotations, stable offsets, citation spans, and multi-parser reconciliation outputs (`EnrichParsedDocumentModel`, `PaperMageJSONParser`, `S2ORCJSONParser`, `ArbitrateParserOutputs`, `opensource/inventory/{papermage,s2orc-doc2json}.md`).
- [x] Add a CERMINE adapter seam and fallback orchestration policy, including a review UI for conflicting parser outputs (`CERMINEXMLParser`, `DefaultParserFallbackPolicy`, `/parsing`, `opensource/inventory/cermine.md`).
- [x] Add stale-parser maintenance/risk scoring and comparative benchmark fixtures for parser candidates before enabling historical fallbacks such as Science Parse (`BuildParserMaintenanceRiskReport`, `rforge parse maintenance-risk`, `opensource/inventory/science-parse.md`).
- [x] Add OpenSearch mapping-version lockfile records, bulk indexing with partial-failure provenance, highlighted passage search results, and an opt-in OpenSearch integration test (`OpenSearchMappingVersion`, `RebuildWithReport`, highlights, `TestOptInOpenSearchIntegration`, `opensource/inventory/opensearch.md`).
- [x] Add a Qdrant adapter implementation with production embedding-provider presets, payload privacy/version locking, calibrated hybrid ranking controls, and an opt-in Qdrant integration test (`QdrantIndex`, `DefaultEmbeddingProviderRegistry`, `CalibrateHybridRetrieval`, `TestOptInQdrantIntegration`, `opensource/inventory/qdrant.md`).
- [x] Add richer OpenAlex cursor-based multi-page import runs with frozen query/cursor state, concept/domain-map import, higher-level filter presets, and opt-in live smoke coverage for those workflows (`OpenAlexFilterPreset`, cursor resume state, concept metadata, live smoke, `opensource/inventory/openalex.md`).
- [x] Add Semantic Scholar quota-aware resumable graph expansion run files plus richer live smoke coverage for API drift and field restrictions (`SemanticScholarGraphRun`, `rforge citations expand --run-state`, live smoke coverage, `opensource/inventory/semantic-scholar.md`).
- [x] Add richer interactive graph exploration beyond the artifact preview: filtering, neighborhood expansion, provenance overlays, and keyboard-accessible alternatives (`/map` filter/neighborhood/provenance controls, keyboard alternatives, `opensource/inventory/semantic-scholar.md`).
- [x] Add additional metafor-backed effect-size calculators, influence diagnostics, richer publication-bias tests, publication-ready plot styling, and a separate Bayesian-engine path beyond the current normal-approximation scaffold (`GridBayesianEngine`, styled SVG plots, additional calculators/diagnostics, `opensource/inventory/metafor.md`).
- [x] Add a reference-manager interchange fidelity matrix that round-trips BibTeX, RIS, CSL-JSON, Zotero RDF, Better BibTeX citation keys, tags, notes, collections, and redacted attachment metadata with per-field loss reports (`BuildReferenceManagerRoundTripMatrix`, `rforge library reference-manager-matrix`, `opensource/inventory/zotero.md`).
- [x] Add a privacy/licensing review gate for imported reference-manager attachments, notes, and annotations before they can be copied into project archives or shareable reports (`oa privacy-review`, `guardReferenceManagerPrivacyGate`, `opensource/inventory/zotero.md`).
- [x] Add an ASReview-style audit export bundle containing frozen screening dataset, seed labels, ranking iterations, reviewer actions, stopping diagnostics, random seeds, and model/version metadata (`ScreeningAuditBundle`, `rforge screen audit-bundle`, `opensource/inventory/asreview.md`).
- [x] Add biomedical full-text workflows for PubMed/Europe PMC/PMC open-access records: PMCID/PMID linking, OA license capture, structured full-text import, supplementary-file discovery, and opt-in live drift smoke tests (`LinkPMCIDPMID`, `ImportStructuredBiomedicalFullText`, `make biomedical-live-smoke`, `opensource/inventory/README.md`, `docs/source-connectors.md`).
- [x] Add a source-API drift dashboard that records live-smoke snapshots for OpenAlex, Semantic Scholar, PubMed, Europe PMC, Crossref, arXiv, and Unpaywall, flags response-shape changes, and links failures to connector provenance (`BuildSourceAPIDriftDashboard`, `rforge protocol live-smoke-dashboard`, `opensource/inventory/README.md`).
- [x] Add retrieval evaluation benchmarks comparing SQLite FTS, OpenSearch, Qdrant vector search, and hybrid ranking on deterministic passage-query fixtures with reproducibility and privacy notes (`RunRetrievalBenchmark`, `rforge retrieve benchmark`, `opensource/inventory/{opensearch,qdrant}.md`).
- [x] Add an OSS-inventory-to-roadmap report that groups `nextSlice` entries by area, detects TODO coverage gaps for new inventory notes, and suggests adapter/test/live-smoke implementation slices without marking them complete (`BuildInventoryRoadmapReport`, `rforge oss inventory-roadmap`, `opensource/inventory/manifest.json`).
- [x] Add reviewer-persistent parsed-reference adjudication: reviewers can accept, correct, reject, or defer Anystyle/GROBID/S2ORC reference matches, with decision provenance and exportable ambiguity queues (`ReferenceAdjudication`, `ExportReferenceAmbiguityQueue`, `rforge parse adjudicated-refs --ambiguity-out`, `opensource/inventory/anystyle.md`).
- [x] Add embedding-provider compliance profiles that document what text leaves the machine, required consent/config, model version locks, dimensionality, retention policy, and redaction behavior before Qdrant/HTTP embedding indexing runs (`DefaultEmbeddingProviderRegistry`, `ValidateEmbeddingProviderCompliance`, `rforge index embedding-providers`, `opensource/inventory/qdrant.md`).
- [x] Add calibrated hybrid retrieval tuning files that store lexical/vector/backend weights, evaluation scores, selected configuration, and query-set checksums for reproducible retrieval comparisons (`CalibrateHybridRetrieval`, `rforge retrieve tune-hybrid`, `opensource/inventory/{opensearch,qdrant}.md`).
- [x] Add OpenAlex author/institution/concept disambiguation review queues so imported domain maps and entity searches do not silently merge ambiguous people, institutions, or concepts (`BuildOpenAlexDisambiguationQueue`, `SearchConcepts`, `rforge search --source openalex --entity concepts`, `opensource/inventory/openalex.md`).
- [x] Add Semantic Scholar/OpenAlex graph expansion budget controls: max depth, max nodes, max API calls, retry budget, resume cursor, and dry-run estimate before live citation expansion (`GraphExpansionBudget`, `rforge citations expand --dry-run`, `opensource/inventory/{semantic-scholar,openalex}.md`).
- [x] Add publication-ready analysis artifact manifests that bundle forest/funnel SVGs, plot settings, checksums, R/metafor script, engine versions, warnings, and report embedding metadata (`AnalysisArtifactManifest`, `rforge analysis run`, `opensource/inventory/metafor.md`).
- [x] Add parser-output license/provenance manifests for external parsers so generated TEI/JSON/reference files record parser source, version, command, input checksum, output checksum, license constraints, and shareability (`ParserRunManifest`, `rforge parse manifest-policies`, `opensource/inventory/{grobid,s2orc-doc2json,cermine,anystyle}.md`).
- [x] Add accessibility and no-JS review views for large citation/domain graphs: tabular edge lists, filtered node tables, keyboard navigation, graph summaries, and exportable graph reports alongside interactive SVGs (`BuildAccessibleGraphView`, `rforge citations accessible-view`, `opensource/inventory/{semantic-scholar,openalex}.md`).

### ResearchForge super-tool synthesis backlog

These tasks combine the open-source inventory into one end-to-end ResearchForge experience: a local, auditable research operating system that still keeps every external tool behind provenance-aware adapters.

- [x] Add a unified research-workflow orchestrator that can run discovery → import → dedupe → legal full-text fetch → parse → index → screen → extract → analyze → report as a resumable DAG with checkpoints, inputs/outputs, provenance, and restart safety (`DefaultWorkflowDAG`, `RunWorkflowDAG`, `rforge forge run-dag`, `opensource/inventory/{openalex,semantic-scholar,zotero,grobid,asreview,metafor}.md`).
- [x] Add a project knowledge graph that merges Zotero collections/tags, OpenAlex concepts, Semantic Scholar citation edges, parsed references, evidence items, screening decisions, analysis runs, and report claims into one queryable local graph (`BuildProjectKnowledgeGraph`, `rforge knowledge query`, `opensource/inventory/{zotero,openalex,semantic-scholar,grobid,s2orc-doc2json,papermage}.md`).
- [x] Add a research-question compiler that turns a PICO/PECO/SPIDER/freeform question into source-specific query plans, inclusion/exclusion criteria drafts, extraction schema drafts, and provenance-tagged review prompts without auto-accepting scientific claims (`internal/protocol/compiler.go`, `rforge protocol compile`, `opensource/inventory/{openalex,semantic-scholar,asreview}.md`).
- [x] Add a citation-to-evidence trace view where every report claim links backward to effect-size rows, accepted evidence, passages, parser outputs, PDFs, reference-manager items, source API records, and raw request/response metadata (`BuildCitationEvidenceTraceView`, `rforge report trace`, `opensource/inventory/{zotero,grobid,papermage,semantic-scholar,metafor}.md`).
- [x] Add a multi-engine parser arbitration layer that scores GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse-style metadata, and Anystyle output per field, routes conflicts to review, and records why one output was selected (`ArbitrateParserOutputs`, `rforge parse arbitrate --parsed`, `opensource/inventory/{grobid,s2orc-doc2json,papermage,cermine,science-parse,anystyle}.md`).
- [x] Add a source-fusion identity resolver that merges DOI/arXiv/PMID/PMCID/OpenAlex/Semantic Scholar/Crossref/Zotero IDs using explainable match rules, conflict records, and reversible merge/split operations (`ResolveIdentityClusters`, `DetectIdentityConflicts`, `ApplyIdentityDecision`, `rforge library identity-decision apply`, `opensource/inventory/{zotero,openalex,semantic-scholar,anystyle}.md`).
- [x] Add a live research map cockpit that combines concept maps, citation neighborhoods, screening priority, parser quality, retrieval hits, and evidence coverage into one local web dashboard with no-JS fallbacks and exportable audit snapshots (`BuildResearchMapCockpitState`, `/map`, `/map/snapshot.json`, `opensource/inventory/{openalex,semantic-scholar,asreview,qdrant,opensearch}.md`).
- [x] Add an evidence gap analyzer that cross-checks the research question, screened-in papers, parsed passages, extracted evidence fields, and meta-analysis inputs to identify missing outcomes, missing comparators, unsupported claims, and studies needing full-text acquisition (`AnalyzeEvidenceGaps`, `rforge evidence gaps --question --screened-in --parsed-paper`, `opensource/inventory/{asreview,grobid,papermage,metafor}.md`).
- [x] Add a Reproducible review package format as the first "done" artifact for the Meta-analysis spine, bundling project manifest, lockfiles, source query plans, dedupe decisions, parser manifests, screening audit, extraction schema, accepted evidence, analysis artifacts, report outputs, redaction policy, and checksums for archival or peer review (`reviewpkg.Create`, `packageRole`, `replay.sh`, `audit-report.json`, `CONTEXT.md`, `opensource/inventory/{zotero,asreview,grobid,metafor}.md`).
- [x] Add a connector capability registry that describes each source/tool adapter's supported entities, rate limits, auth needs, live-smoke status, license/shareability policy, cacheability, and supported provenance fields (`protocol.DefaultConnectorCapabilityRegistry`, `rforge protocol capabilities`, `opensource/inventory/manifest.json`).
- [x] Add a cross-tool benchmark suite that measures discovery recall, dedupe precision, parser field accuracy, reference-normalization accuracy, retrieval quality, screening effort savings, and report reproducibility on deterministic fixture projects (`BuildCrossToolBenchmarkReport`, `BenchmarkFixture`, `rforge benchmark cross-tool`, `opensource/inventory/{openalex,semantic-scholar,grobid,anystyle,asreview,opensearch,qdrant}.md`).
- [x] Add an LLM-assisted but citation-locked synthesis layer that can suggest query expansions, screening rationales, extraction candidates, and report prose only when every suggested sentence is linked to exact source passages/evidence and marked unaccepted until reviewer approval (`DraftCitationLockedSuggestions`, `EverySuggestedSentenceCitationLocked`, `rforge evidence citation-suggest --kind query_expansion|screening_rationale|extraction|report_prose`, `RESEARCH-FORGE-PRD.md`, `opensource/inventory/{grobid,papermage,asreview}.md`).
- [x] Add a lab-notebook timeline that records all human and automated workflow events across imports, source refreshes, parser runs, reviewer decisions, extraction edits, analysis reruns, and report builds as a browsable provenance journal (`BuildLabNotebookTimelineState`, `/notebook`, `/notebook/snapshot.json`, `opensource/inventory/{zotero,asreview,metafor}.md`).
- [x] Add a method-comparison workbench where users can compare parser choices, retrieval backends, screening rankers, effect-size models, and publication-bias diagnostics side-by-side before selecting the method locked into the final report (`DefaultMethodComparisonWorkbench`, `CompareWithSelection`, `rforge analysis method-workbench --select --reviewer --reason`, `opensource/inventory/{grobid,cermine,opensearch,qdrant,asreview,metafor}.md`).
- [x] Add a one-command `rforge forge` guided workflow that creates or opens a project, asks for the research question and source/tool choices, previews privacy/legal implications, then drives the unified orchestrator with review gates at every irreversible scientific or data-sharing decision (`forge.Init`, `rforge forge init --sources --tools`, `rforge forge run-dag`, `RESEARCH-FORGE-PRD.md`, `opensource/inventory/README.md`).

### Super-tool planning and HTMX dashboard requirements backlog

These planning tasks turn the super-tool synthesis into concrete product slices and local Go + HTMX cockpit requirements. The dashboard should remain a review/control surface over shared Go services, not a separate source of scientific truth.

- [x] Write a ResearchForge super-tool blueprint that maps each open-source-inspired capability to a first-class ResearchForge module, CLI command family, HTMX page, storage tables/files, provenance events, and validation gate, prioritizing the Meta-analysis spine for meta-analysis authors before broad general-purpose research-cockpit expansion (`docs/meta-analysis-spine-blueprint.md`, `RESEARCH-FORGE-PRD.md`, `opensource/inventory/README.md`, `docs/web-gui-plan.md`).
- [x] Add a phased super-tool roadmap that makes the Meta-analysis spine the first product path: source plan → import/dedupe → legal full text → parser arbitration → screening → evidence extraction → statistics → Reproducible review package, then layers broader knowledge-cockpit features after that package is audit/replay safe (`docs/meta-analysis-spine-roadmap.md`, `RESEARCH-FORGE-PRD.md`, `TODO.md`).
- [x] Define an end-to-end project state machine for `rforge forge`: question draft, source plan, import plan, dedupe review, full-text acquisition, parser arbitration, indexing, screening, extraction, analysis, report, archive, and reopen/resume states (`docs/rforge-forge-state-machine.md`, `RESEARCH-FORGE-PRD.md`).
- [x] Add HTMX dashboard requirement: a Forge home timeline showing active project, current workflow state, latest provenance events, blocked review gates, background jobs, and next safe actions with CLI-equivalent commands shown for every button (`BuildForgeHomeState`, `/forge`, `/forge/refresh`, `docs/web-gui-plan.md`).
- [x] Add HTMX dashboard requirement: a source-planning cockpit for OpenAlex, Semantic Scholar, Crossref, arXiv, PubMed/Europe PMC, Unpaywall, Zotero, and local imports, including rate-limit/auth/privacy warnings and dry-run result estimates before network calls (`/sources`, `docs/web-gui-plan.md`, `opensource/inventory/{openalex,semantic-scholar,zotero}.md`).
- [x] Add HTMX dashboard requirement: an import/deduplication workbench that shows identity clusters, conflicting source fields, Zotero collection/tag context, citation-key preservation, merge/split history, and reversible decisions (`BuildDedupeReviewState`, `/dedupe`, `opensource/inventory/{zotero,openalex,semantic-scholar}.md`).
- [x] Add HTMX dashboard requirement: a legal full-text acquisition queue showing OA/license status, source URL, expected stored path, restricted/shareable flags, and explicit reviewer approval before downloading or archiving documents (`BuildAcquisitionQueueState`, `/acquisition`, `RESEARCH-FORGE-PRD.md`).
- [x] Add HTMX dashboard requirement: a parser arbitration screen comparing GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse-style metadata, and Anystyle outputs field-by-field with confidence, raw text, offsets, warnings, and accept/correct/reject controls (`BuildParserConflictReviewState`, `/parsing`, `opensource/inventory/{grobid,s2orc-doc2json,papermage,cermine,science-parse,anystyle}.md`).
- [x] Add HTMX dashboard requirement: a retrieval tuning screen comparing SQLite FTS, OpenSearch, Qdrant vector, and hybrid results for the same query, with passage provenance, ranking explanations, embedding privacy status, and benchmark scores (`BuildRetrievalTuningState`, `/retrieve`, `opensource/inventory/{opensearch,qdrant}.md`).
- [x] Add HTMX dashboard requirement: an ASReview-inspired screening cockpit with active-learning queue, uncertainty/exploration flags, reviewer assignment, conflict/adjudication panels, recall/effort curves, stopping diagnostics, and exportable audit bundle links (`BuildScreeningCockpitState`, `/screening`, `opensource/inventory/asreview.md`).
- [x] Add HTMX dashboard requirement: an evidence extraction grid linking each field to source passage/table/figure/equation, parser offset, PDF view, reviewer status, correction history, and downstream analysis inclusion status (`BuildEvidenceGridState`, `/evidence`, `opensource/inventory/{grobid,papermage,metafor}.md`).
- [x] Add HTMX dashboard requirement: a meta-analysis workbench showing prepared effect-size inputs, model choices, metafor script, warnings, heterogeneity, sensitivity/influence diagnostics, forest/funnel artifacts, and publication-ready artifact manifests (`BuildAnalysisWorkbenchState`, `/analysis`, `opensource/inventory/metafor.md`).
- [x] Add HTMX dashboard requirement: a claim traceability panel in the report builder where every generated paragraph/table/figure references accepted evidence and unresolved or weakly supported claims block final export (`BuildReportClaimPanelState`, `/report`, `RESEARCH-FORGE-PRD.md`).
- [x] Add HTMX dashboard requirement: a research map view that unifies citation graph, OpenAlex concepts, Zotero collections/tags, screening status, retrieval clusters, and evidence coverage, with filters, keyboard navigation, no-JS tables, and snapshot export (`BuildResearchMapCockpitState`, `/map`, `/map/snapshot.json`, `opensource/inventory/{openalex,semantic-scholar,zotero,asreview}.md`).
- [x] Add HTMX dashboard requirement: a connector health/control center showing live-smoke history, API drift alerts, cache status, credential redaction checks, rate-limit budgets, and adapter capability registry coverage (`/connectors`, `opensource/inventory/manifest.json`).
- [x] Add HTMX dashboard requirement: a reproducibility/export center that previews the Reproducible review package contents, redaction results, checksums, lockfiles, external-tool versions, parser manifests, analysis artifacts, report outputs, and reviewer decision logs before package creation (`BuildPackageExportCenterState`, `/package`, `CONTEXT.md`, `opensource/inventory/{zotero,grobid,asreview,metafor}.md`).
- [x] Add planning artifact: a dashboard information architecture diagram listing routes, partial endpoints, view models, no-JS fallbacks, background jobs, and ownership boundaries for every Forge cockpit screen (`BuildDashboardInformationArchitecture`, `/architecture`, `docs/web-gui-plan.md`).
- [x] Add planning artifact: a permissions/privacy model for the dashboard that classifies local-only paths, copyrighted PDFs, reviewer notes, credentials, embeddings, cache files, and shareable report fields (`BuildDashboardPrivacyModel`, `/privacy`, `docs/web-gui-plan.md`, `docs/privacy-copyright.md`).
- [x] Add planning artifact: an acceptance-test matrix mapping each super-tool workflow state and HTMX dashboard screen to unit tests, handler tests, Playwright paths, screenshot coverage, CLI parity checks, and provenance assertions (`docs/meta-analysis-spine-acceptance-matrix.md`, `docs/web-gui-plan.md`).

### Expanded OSS inventory study backlog

These tasks come from the expanded OSS inventory and deepen ResearchForge into a super-tool across reference management, source discovery, NLP, screening, retrieval, and statistics.

- [x] Add JabRef-inspired BibTeX/BibLaTeX quality reports covering citation-key collisions, groups/saved searches, field cleanup diffs, linked-file privacy, and reviewer-approved normalization (`BuildJabRefQualityReport`, `rforge library jabref-quality`, `opensource/inventory/jabref.md`).
- [x] Add RobotReviewer-inspired risk-of-bias and evidence-suggestion workflows where every automated judgment cites exact support text, uncertainty, model/version metadata, and reviewer accept/correct/reject state (`EveryRiskOfBiasJudgmentAuditable`, `rforge evidence risk-bias-*`, `opensource/inventory/robotreviewer.md`).
- [x] Add revtools-inspired visual clustering for duplicate review and screening triage, with exportable cluster decisions and PRISMA/audit provenance (`/dedupe`, `BuildDedupeReviewState`, `opensource/inventory/revtools.md`).
- [x] Add PyMARE-style secondary meta-analysis engine comparison reports against metafor fixtures, including environment locks, model-setting parity, warning capture, and output deltas (`CompareAnalysisEngines`, `rforge analysis engine-compare`, `opensource/inventory/pymare.md`).
- [x] Add SentenceTransformers-style embedding model registry entries for local/remote providers, dimensions, license notes, text-egress policy, vector-index invalidation, and retrieval benchmark compatibility (`DefaultEmbeddingProviderRegistry`, `rforge index embedding-providers`, `opensource/inventory/sentence-transformers.md`).
- [x] Add BERTopic-inspired topic/domain map artifacts with representative papers/passages, reviewer-edited labels, topic merge/split history, model settings, and citation-graph links (`BuildDomainMapArtifact`, `rforge citations domain-map`, `opensource/inventory/bertopic.md`).
- [ ] Add SciSpaCy-inspired scientific entity extraction suggestions with passage offsets, abbreviation resolution, entity-link candidates, confidence, model provenance, and reviewer decisions (`opensource/inventory/scispacy.md`).
- [ ] Add KeyBERT-inspired keyword/query-expansion suggestions linked to abstracts/passages, with diversity scoring, reviewer approval, and before/after search-plan provenance (`opensource/inventory/keybert.md`).
- [ ] Add NASA ADS connector planning and implementation slices for bibcode/DOI search, physics/astronomy metadata normalization, citation expansion, token redaction, and opt-in live smoke tests (`opensource/inventory/nasa-ads.md`).
- [ ] Add DOAJ/CORE open-access discovery slices for license-aware full-text candidate queues, source URL provenance, attribution/rate-limit handling, and reviewer-approved acquisition (`opensource/inventory/doaj-core.md`).
