# ResearchForge Roadmap

ResearchForge will be built as a sequence of test-first vertical milestones. Each milestone must preserve the core principle:

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

Decision-gated roadmap items are tracked through `rforge decisions`, `docs/owner-decisions.md`, and `docs/remaining-todo-audit.md`. Fyne desktop delivery has been re-scoped to a Go + HTMX local research cockpit by ADR 0006. Dependency-free view models are available for cockpit pages and guided local actions.

## Release stages

| Stage | Purpose | Expected outcome |
|---|---|---|
| Pre-alpha | Prove project, CLI, provenance, and metadata-library foundation | Developers can create projects and search/store scholarly metadata |
| Alpha | Complete end-to-end review workflow on fixtures/open data | Users can search, dedupe, screen, extract evidence, and build reports on controlled data |
| Beta | Real-world open-access parsing and meta-analysis | Researchers can run a full MVP workflow with legal OA papers and auditable analysis |
| 1.0 | Stable reproducible research engine | Project format, CLI workflows, web GUI, docs, and release process are stable enough for broad use |

## Milestone 0 — Foundation

**Goal:** establish the Go application foundation, `rforge` CLI, web GUI shell, local project model, storage, provenance, and validation pipeline.

**Major deliverables:**

- Go module and repository structure
- CLI skeleton and JSON output convention
- Project create/open/list commands
- Project manifest and workflow lockfile
- SQLite local storage foundation
- Append-only provenance event log
- `rforge doctor`
- local web GUI launcher, dashboard placeholder, and generated-artifact placeholder
- CI for formatting, tests, vetting, and vulnerability checks

**Exit criteria:**

- `rforge project create` creates a usable local project.
- Project contains manifest, lockfile, database, and event log.
- `rforge doctor --json` reports local readiness.
- `rforge ui` starts a local web server and opens a project placeholder.

## Milestone 1 — Scholarly metadata and library MVP

**Goal:** search scholarly metadata sources, normalize records, deduplicate, and export a research library.

**Major deliverables:**

- Source connector interface
- OpenAlex connector
- arXiv connector
- Crossref connector
- Unpaywall OA lookup
- Normalized `PaperRecord` library
- DOI/arXiv/title-author-year deduplication
- BibTeX/RIS/CSV/JSON import and export
- CLI search/import/export/library commands
- web GUI search and library screens

**Exit criteria:**

- User can search OpenAlex and arXiv from CLI.
- Records are stored and deduplicated with provenance preserved.
- User can export a library to BibTeX/CSV/JSON.

## Milestone 2 — OSS repository intelligence MVP

**Goal:** safely study open-source research tooling and maintain an auditable repository catalog.

**Major deliverables:**

- Gitignored `opensource/clones/` workspace
- OSS repository registry
- `rforge oss add/list/clone/license-check/note/report`
- License and risk metadata
- Study-note templates
- web GUI OSS dashboard

**Exit criteria:**

- User can catalog and shallow-clone repositories without committing clones.
- Repository notes, licenses, risks, and integration findings are committed as metadata.
- web GUI can display studied repositories.

## Milestone 3 — Full-text acquisition, parsing, and retrieval

**Goal:** retrieve legal open-access documents, parse them, and search exact passages.

**Major deliverables:**

- `DocumentAsset` model with OA/license/checksum metadata
- Legal OA PDF acquisition through Unpaywall/source links
- Manual local-only document import
- GROBID parser adapter
- Parsed sections, references, tables, and passages
- Local full-text index MVP
- Optional OpenSearch/Qdrant adapter seams
- `rforge pdf fetch`, `rforge parse`, `rforge index`, `rforge retrieve`
- web GUI PDF/section/passages view

**Exit criteria:**

- User can fetch legal OA PDFs where available.
- User can parse a paper with GROBID.
- Retrieval results include exact paper, section, and passage provenance.

## Milestone 4 — Screening workflow

**Goal:** support systematic-review screening with auditable decisions and PRISMA counts.

**Major deliverables:**

- Screening stages and decisions
- Configurable exclusion reasons
- Reviewer attribution
- Decision history
- Queue filtering and conflict/uncertain queues
- PRISMA counts from stored state/events
- CSV screening import/export
- Active-learning prioritization scaffold
- web GUI screening queue and decision panel

**Exit criteria:**

- User can include/exclude/mark uncertain with reasons.
- Decisions are attributable and auditable.
- PRISMA counts can be regenerated.

## Milestone 5 — Evidence extraction

**Goal:** extract structured evidence linked to exact source material.

**Major deliverables:**

- Extraction schema format
- Evidence item model
- Manual evidence extraction
- Source-support requirement
- Validation states: suggested, accepted, rejected, corrected
- LLM suggestion adapter behind explicit configuration
- Evidence export to CSV/JSON/Markdown
- web GUI evidence table and source-link view

**Exit criteria:**

- Accepted evidence cannot exist without source support.
- LLM suggestions require human review before acceptance.
- Evidence tables export with source links and provenance.

## Milestone 6 — Meta-analysis MVP

**Goal:** run a basic reproducible meta-analysis from accepted evidence.

**Major deliverables:**

- Analysis input snapshots
- Effect-size calculator interface
- First effect-size calculator
- R/metafor adapter
- Generated scripts/notebooks
- Forest/funnel plot artifact registration
- Heterogeneity metrics
- Sensitivity-analysis scaffold
- CLI analysis commands
- web GUI analysis setup/results view for CLI-generated meta-analysis artifacts

**Exit criteria:**

- User can run a basic meta-analysis from an evidence table.
- Inputs, scripts, versions, outputs, and warnings are stored.
- Results can be reproduced from project artifacts.

## Milestone 7 — Reproducible reports

**Goal:** generate auditable research reports with citations, evidence, screening, analysis, and provenance.

**Major deliverables:**

- Markdown report generator
- HTML export
- LaTeX export scaffold
- Citation and bibliography tables
- Evidence tables
- PRISMA diagram/summary
- Analysis result sections
- Audit appendix
- web GUI report/artifact browser for CLI-generated reports, PRISMA diagrams, citation diagrams, and analysis plots

**Exit criteria:**

Report can answer:

- What did we search?
- Where did each paper come from?
- Why was each paper included or excluded?
- What exact source supports each extracted claim?
- What statistical model was run?
- Can another researcher reproduce the result?

## Milestone 8 — Hardening, docs, packaging, and beta

**Goal:** make the MVP usable by early researchers across platforms.

**Major deliverables:**

- Threat model and security hardening
- Path/archive/secret/external-command safety tests
- Performance benchmarks
- User and developer documentation
- Example open-data project
- Project archive/restore
- Cross-platform CLI builds
- web GUI local-server/static-build smoke tests after stack selection
- Checksums and release notes

**Exit criteria:**

- New user can install ResearchForge.
- New user can complete the MVP workflow on open data.
- Project can be archived/restored.
- Release artifacts exclude secrets, local clones, and restricted assets.

## Cross-cutting roadmap tracks

### TDD and fixtures

- Build deterministic fixtures before external-service-heavy code.
- Keep all normal tests offline.
- Use mocked APIs, generated PDFs, TEI fixtures, fake git repos, and golden reports.

### Provenance and reproducibility

- Every external query, parser run, extraction, screening decision, analysis, and report build must be logged.
- Lockfiles must capture tool versions and relevant parameters.
- Reports must include audit appendices.

### Privacy and copyright

- Default to local-first operation.
- Redact secrets and local private paths from shareable output.
- Fetch only legal OA full text automatically.
- Keep restricted assets out of git and shareable archives.

### CLI/UI parity

- Implement behavior in shared Go services.
- Validate via CLI first where practical.
- Add local web GUI view models, artifact APIs, and screens without duplicating business logic.

### Quality and release

- Keep CI green from Milestone 0 onward.
- Add security and performance checks as workflows mature.
- Publish releases only after explicit owner approval.
- Treat `TODO.md:34` as the current release blocker until `make license-decision-live-audit` and `make license-decision-approval-gate` show `approved:true` for the project license decision.

## First execution slice

Recommended first TDD slice:

```text
Slice: rforge project create writes a manifest and provenance event
Milestone: 0
Primary skill: research-forge-foundation-tdd
Red test: project create service test expects rforge.project.toml and event log entry
Green implementation: minimal project package and CLI command
Refactor target: isolate project paths and test clock/ID generation
Validation: go test ./... && go run ./cmd/rforge project create ./tmp/demo --title "Demo Review"
Acceptance: demo project has manifest, lockfile placeholder, SQLite placeholder or data dir, and provenance event
```
