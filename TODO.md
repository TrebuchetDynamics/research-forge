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

### Post-1.0 backlog

- [x] Make `import` resilient to duplicate identifiers: skip in-file/in-store duplicates and no-identifier records and report them, instead of aborting the whole import and leaving partial state (parsers skip+count unstorable records, `Store.ImportRecords` skips+reports duplicates; merging stays in `duplicate merge`). See `docs/superpowers/specs/2026-06-13-import-duplicate-handling-design.md`.
- [x] Add a web GUI end-to-end test that builds view models from a CLI-generated project and serves them through the `internal/webui` handlers, tying CLI-produced artifacts to the cockpit pages (`internal/webui.BuildLibraryViewModel`/`BuildArtifactDashboardState` read the project's library, screening, and analysis state; e2e serves the library and artifacts handlers via httptest in `internal/webui/builders_test.go`).
- [ ] Add opt-in live-service smoke tests for source connectors (OpenAlex, arXiv, Crossref, Unpaywall) behind an environment guard, mirroring the external e2e pattern.
- [ ] Add an opt-in GROBID parse e2e against a real GROBID endpoint behind an environment guard.
- [ ] Add an opt-in R/metafor `analysis run` e2e using a real Rscript to complement the deterministic `FakeRunner` path.
- [x] Add a multi-reviewer screening e2e exercising conflict detection and the uncertain queue through the CLI (`internal/cli/screening_e2e_test.go`; added the enabling `rforge screen conflicts --stage <stage>` command over the existing `MemoryStore.Conflicts`).
