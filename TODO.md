# ResearchForge TODO

This is the end-to-end implementation checklist for ResearchForge. All production-code tasks must be developed with TDD: write the failing test first, make it pass, then refactor.

See also:

- [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md)
- [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)
- [ROADMAP.md](./ROADMAP.md)
- [SKILLS.md](./SKILLS.md)

## Global rules

- [ ] Use red-green-refactor for every implementation slice.
- [ ] Keep CLI and Fyne behavior backed by shared Go application services.
- [ ] Record provenance for user-visible workflow changes and external-tool/API outputs.
- [ ] Avoid live network dependencies in normal tests.
- [ ] Use only legal, deterministic, minimal test fixtures.
- [ ] Keep local clone workspaces, copyrighted PDFs, secrets, and private data out of git.
- [ ] Prefer local-first operation; make heavyweight services optional until required.
- [ ] Add ADRs only for hard-to-reverse, surprising trade-offs.

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
- [ ] Add license after owner decision.
- [x] Add `CONTEXT.md` glossary when first domain terms are finalized.
- [ ] Add `docs/adr/` and ADR index when first ADR is accepted.

## 1. Milestone 0 — Go/Fyne/CLI foundation

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
- [ ] Add `govulncheck` gate when feasible.

### 1.2 CLI skeleton

- [ ] Choose CLI framework with ADR if needed.
- [x] Add root `rforge --help`.
- [x] Add `rforge version`.
- [x] Add global flags: `--project`, `--config`, `--json`, `--log-level`.
- [x] Add consistent JSON output envelope.
- [x] Add consistent error format and exit codes.
- [ ] Add shell completion if framework supports it.
- [x] Add CLI command tests.

### 1.3 Project workspace

- [x] Define `ResearchProject` domain type.
- [x] Define project directory layout.
- [x] Add `rforge project create <path> --title <title>`.
- [x] Add `rforge project open/inspect <path>`.
- [x] Add `rforge project list`.
- [x] Write `rforge.project.toml` manifest.
- [x] Write `rforge.lock.json` lockfile.
- [x] Initialize local project data directories.
- [ ] Initialize SQLite local database.
- [ ] Add project path validation and traversal tests.
- [ ] Add archive-safe project metadata.

### 1.4 Manifest, lockfile, provenance

- [x] Define manifest schema and version.
- [x] Define lockfile schema and version.
- [x] Add manifest read/write tests.
- [x] Add lockfile read/write tests.
- [x] Add append-only provenance event log.
- [x] Add event IDs, timestamps, actor, action, target, inputs, outputs, and warnings.
- [ ] Add event replay/query helpers.
- [x] Record project-create event.
- [ ] Record CLI command provenance where relevant.
- [ ] Add deterministic test clock/ID generator.

### 1.5 Storage foundation

- [ ] Decide SQLite-first vs PostgreSQL-first; recommended SQLite-first.
- [ ] Add storage interface.
- [ ] Add SQLite implementation.
- [ ] Add migration mechanism.
- [ ] Add migration tests.
- [ ] Add database backup before migrations.
- [ ] Add storage health check.
- [ ] Prepare PostgreSQL adapter seam for later.

### 1.6 Doctor command

- [ ] Add `rforge doctor`.
- [ ] Check Go/runtime version where useful.
- [ ] Check project manifest/lockfile validity.
- [ ] Check SQLite availability.
- [ ] Check optional GROBID endpoint.
- [ ] Check optional OpenSearch endpoint.
- [ ] Check optional Qdrant endpoint.
- [ ] Check optional R/metafor.
- [ ] Output actionable JSON and human-readable results.

### 1.7 Fyne app shell

- [ ] Add Fyne dependency after build decision.
- [ ] Add `rforge ui` or separate desktop entry point.
- [ ] Add app shell.
- [ ] Add project dashboard placeholder.
- [ ] Add background job abstraction.
- [ ] Add view-model tests for dashboard state.
- [ ] Ensure no core logic lives in widgets.

## 2. Milestone 1 — Scholarly metadata and library MVP

### 2.1 Source connector framework

- [ ] Define `SourceConnector` interface.
- [ ] Define `SourceQuery` domain type.
- [ ] Define connector request/response provenance.
- [ ] Add HTTP client with timeouts, retries, user-agent, and rate-limit behavior.
- [ ] Add source response cache.
- [ ] Add mocked HTTP test harness.

### 2.2 Paper library model

- [ ] Define `PaperRecord`.
- [ ] Add identifiers: DOI, arXiv ID, PMID, OpenAlex ID, Crossref ID, Semantic Scholar ID.
- [ ] Add authors, title, abstract, year, venue, publisher, URLs, license/OA status.
- [ ] Store raw source payload references.
- [ ] Store provenance per source.
- [ ] Add create/update/list/search library storage.
- [ ] Add library CLI list command.
- [ ] Add Fyne library view model.

### 2.3 OpenAlex connector

- [ ] Add OpenAlex fixture responses.
- [ ] Test query URL/parameters.
- [ ] Normalize OpenAlex works into `PaperRecord`.
- [ ] Store OpenAlex source metadata.
- [ ] Add `rforge search --source openalex`.
- [ ] Add pagination/limit behavior.
- [ ] Add rate-limit/backoff handling.

### 2.4 arXiv connector

- [ ] Add arXiv Atom fixtures.
- [ ] Test query URL/parameters.
- [ ] Normalize arXiv entries into `PaperRecord`.
- [ ] Preserve arXiv versions and categories.
- [ ] Add `rforge search --source arxiv`.

### 2.5 Crossref connector

- [ ] Add Crossref fixture responses.
- [ ] Test query URL/parameters.
- [ ] Normalize Crossref works into `PaperRecord`.
- [ ] Preserve DOI/reference metadata.
- [ ] Add `rforge search --source crossref`.

### 2.6 Unpaywall connector

- [ ] Add Unpaywall fixtures.
- [ ] Test DOI lookup behavior.
- [ ] Normalize OA status, license, best OA location, PDF URLs.
- [ ] Add `rforge oa lookup <doi>`.
- [ ] Ensure email/API configuration does not leak.

### 2.7 Deduplication

- [ ] Define duplicate scoring model.
- [ ] Deduplicate exact DOI matches.
- [ ] Deduplicate normalized arXiv IDs.
- [ ] Deduplicate fuzzy title + author + year.
- [ ] Merge source provenance safely.
- [ ] Preserve all source identifiers.
- [ ] Add duplicate review/report command.
- [ ] Add tests for false positive boundaries.

### 2.8 Imports and exports

- [ ] Add BibTeX parser and fixtures.
- [ ] Add RIS parser and fixtures.
- [ ] Add CSV import.
- [ ] Add JSON import.
- [ ] Add BibTeX export golden tests.
- [ ] Add RIS export golden tests.
- [ ] Add CSV export golden tests.
- [ ] Add JSON export golden tests.
- [ ] Add `rforge import` and `rforge export`.

### 2.9 Search/library UI

- [ ] Add search form view model.
- [ ] Add search result table view model.
- [ ] Add library table/detail view model.
- [ ] Add Fyne search screen.
- [ ] Add Fyne library screen.
- [ ] Add loading/error/empty states.
- [ ] Ensure UI calls shared services.

## 3. Milestone 2 — OSS repository intelligence MVP

- [ ] Add `opensource/README.md`.
- [ ] Add `.gitignore` rule for `opensource/clones/`.
- [ ] Define `OSSRepositoryStudy` domain type.
- [ ] Add repository name validation.
- [ ] Add OSS registry storage.
- [ ] Add `rforge oss add <owner/repo>`.
- [ ] Add `rforge oss list`.
- [ ] Add safe clone path resolution.
- [ ] Add shallow clone command runner abstraction.
- [ ] Add tests with local fake git repositories.
- [ ] Add `rforge oss clone <owner/repo>`.
- [ ] Add license-file detection.
- [ ] Add `rforge oss license-check`.
- [ ] Add study-note template.
- [ ] Add `rforge oss note`.
- [ ] Add topic scan metadata workflow.
- [ ] Add `rforge oss scan --topic`.
- [ ] Add `rforge oss report --area`.
- [ ] Add Fyne OSS dashboard view model and screen.
- [ ] Ensure external source code is not copied into production code without review.

## 4. Milestone 3 — Legal full-text, parsing, and indexing

### 4.1 Document assets and OA policy

- [ ] Define `DocumentAsset`.
- [ ] Add acquisition source, license, OA status, checksum, local path, and MIME type.
- [ ] Add copyright/OA guard tests.
- [ ] Add legal PDF URL selection from Unpaywall metadata.
- [ ] Add `rforge pdf fetch --doi` with mocked HTTP tests.
- [ ] Add manual local file import with local-only status.
- [ ] Prevent accidental export of restricted assets.

### 4.2 GROBID parsing

- [ ] Define parser adapter interface.
- [ ] Add GROBID client with timeout/error handling.
- [ ] Add mock TEI fixtures.
- [ ] Parse title/authors/abstract/sections/references.
- [ ] Generate deterministic section and passage IDs.
- [ ] Record parser version/config in lockfile or event log.
- [ ] Add `rforge parse --paper <id> --parser grobid`.
- [ ] Add parser warning/error storage.

### 4.3 Indexing and retrieval

- [ ] Define `ParsedDocument` and passage model.
- [ ] Add local full-text index MVP with SQLite FTS or Bleve.
- [ ] Add `rforge index rebuild`.
- [ ] Add passage retrieval with paper/section/passage references.
- [ ] Add `rforge retrieve --query`.
- [ ] Add optional OpenSearch adapter seam.
- [ ] Add optional Qdrant adapter seam.
- [ ] Add embeddings adapter seam if needed.
- [ ] Add Fyne PDF/section/passages view model and screen.

## 5. Milestone 4 — Screening workflow

- [ ] Define screening stages: title/abstract, full text, final inclusion.
- [ ] Define decisions: include, exclude, uncertain.
- [ ] Define exclusion reason configuration.
- [ ] Add reason validation tests.
- [ ] Add reviewer attribution.
- [ ] Add `rforge screen configure`.
- [ ] Add `rforge screen decide`.
- [ ] Add decision event history.
- [ ] Add queue filtering by stage/status.
- [ ] Add `rforge screen queue`.
- [ ] Add conflict detection for multi-reviewer workflows.
- [ ] Add uncertain queue.
- [ ] Add PRISMA count generation from stored state/events.
- [ ] Add `rforge prisma counts`.
- [ ] Add CSV screening export/import.
- [ ] Add ASReview-style active-learning prioritization scaffold.
- [ ] Add Fyne screening queue and decision panel.

## 6. Milestone 5 — Evidence extraction

- [ ] Define extraction schema format.
- [ ] Add schema validation.
- [ ] Define `EvidenceItem`.
- [ ] Require source support for accepted evidence.
- [ ] Add support kinds: passage, table, figure, equation, dataset, citation.
- [ ] Add `rforge extraction schema add`.
- [ ] Add manual `rforge extract add`.
- [ ] Add status transitions: suggested, accepted, rejected, corrected.
- [ ] Preserve correction history.
- [ ] Add evidence audit command for unsupported/weak evidence.
- [ ] Add CSV/JSON/Markdown evidence export.
- [ ] Add LLM suggestion adapter interface.
- [ ] Add explicit LLM configuration and secret handling.
- [ ] Add `rforge extract suggest`.
- [ ] Ensure suggestions cannot become accepted without review.
- [ ] Add Fyne evidence table and source-link view.

## 7. Milestone 6 — Meta-analysis MVP

- [ ] Define `AnalysisRun`.
- [ ] Generate analysis input table from accepted evidence.
- [ ] Add effect-size helper interface.
- [ ] Implement first effect-size calculator.
- [ ] Add known-result fixtures.
- [ ] Generate R/metafor scripts.
- [ ] Add opt-in R/metafor integration tests.
- [ ] Capture R and package versions.
- [ ] Run R/metafor through safe external command wrapper.
- [ ] Store input snapshots, scripts, outputs, warnings, and checksums.
- [ ] Register forest plot artifacts.
- [ ] Register funnel plot artifacts where applicable.
- [ ] Parse heterogeneity metrics.
- [ ] Add sensitivity-analysis scaffold.
- [ ] Add `rforge analysis prepare`.
- [ ] Add `rforge analysis run`.
- [ ] Add `rforge analysis export`.
- [ ] Add Fyne analysis setup/results view.

## 8. Milestone 7 — Report generation

- [ ] Define report data model.
- [ ] Add fixture project for report golden tests.
- [ ] Build Markdown report skeleton.
- [ ] Add citation table.
- [ ] Add bibliography output.
- [ ] Add evidence tables.
- [ ] Add screening summary.
- [ ] Add PRISMA diagram output.
- [ ] Add analysis result section.
- [ ] Add forest/funnel plot references.
- [ ] Add audit appendix.
- [ ] Ensure report answers PRD audit questions.
- [ ] Add HTML export.
- [ ] Add LaTeX export scaffold.
- [ ] Add `rforge report build`.
- [ ] Add `rforge report audit`.
- [ ] Add Fyne report builder/export flow.

## 9. Milestone 8 — Hardening and beta release

### 9.1 Quality and security

- [ ] Add threat model document.
- [ ] Add path traversal tests for project/archive/clone/document paths.
- [ ] Add archive extraction safety tests.
- [ ] Add external command argument safety tests.
- [ ] Add API key redaction tests.
- [ ] Add HTTP timeout tests.
- [ ] Add bounded response-size tests where needed.
- [ ] Add fuzz tests for import parsers.
- [ ] Add race tests for background jobs.
- [ ] Add dependency/license scan workflow.

### 9.2 Performance

- [ ] Add benchmark datasets for 10, 1,000, and 100,000 records.
- [ ] Benchmark deduplication.
- [ ] Benchmark imports/exports.
- [ ] Benchmark index rebuild.
- [ ] Benchmark report generation.
- [ ] Add cancellation tests for long jobs.
- [ ] Add memory/allocation regression notes.
- [ ] Ensure Fyne UI stays responsive during jobs.

### 9.3 Documentation

- [ ] Add user installation guide.
- [ ] Add quickstart tutorial.
- [ ] Add CLI command reference.
- [ ] Add project format documentation.
- [ ] Add external service setup docs.
- [ ] Add privacy/copyright documentation.
- [ ] Add developer setup guide.
- [ ] Add architecture overview.
- [ ] Add ADR index.
- [ ] Add fixture policy documentation.
- [ ] Add example open-data project.

### 9.4 Release packaging

- [ ] Add cross-platform CLI build automation.
- [ ] Add Fyne package smoke checks.
- [ ] Add checksums for artifacts.
- [ ] Add SBOM/dependency metadata if feasible.
- [ ] Add project archive/restore commands.
- [ ] Add upgrade tests for project format.
- [ ] Add release notes template.
- [ ] Add install smoke test.
- [ ] Prepare pre-alpha release.
- [ ] Prepare alpha release.
- [ ] Prepare beta release.
- [ ] Prepare 1.0 release only after MVP success criteria pass.

## 10. Final MVP acceptance checklist

The MVP is complete when a researcher can:

- [ ] Create a research project from the CLI.
- [ ] Create/open a research project from the Fyne UI.
- [ ] Search OpenAlex for a topic.
- [ ] Search Crossref for a topic.
- [ ] Search arXiv for a topic.
- [ ] Deduplicate imported/searched records.
- [ ] Retrieve legal open-access PDFs where available.
- [ ] Parse papers with GROBID.
- [ ] Search/retrieve exact passages.
- [ ] Screen papers with include/exclude reasons.
- [ ] Generate PRISMA counts.
- [ ] Extract structured evidence into tables.
- [ ] Link evidence to source passages/tables/figures/equations.
- [ ] Run a basic meta-analysis.
- [ ] Study and catalog relevant OSS repositories from the CLI.
- [ ] View OSS repository studies in Fyne.
- [ ] Export a reproducible report with citations and provenance.
- [ ] Reproduce the report from stored manifest, lockfile, provenance, and project data.
