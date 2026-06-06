# ResearchForge Development Plan

This plan turns the ResearchForge PRD into an implementation roadmap for building the `rforge` CLI, Fyne desktop app, reproducible research workspace, scholarly ingestion engine, screening workflow, evidence extraction pipeline, meta-analysis module, report generator, and OSS repository intelligence system.

See [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md) for product requirements.

## 1. Development principles

1. **Retrieval-first** — search, imports, PDFs, parsed passages, and reports must preserve source identity.
2. **Provenance-first** — every user action, external query, parser output, extraction, model suggestion, and statistical run must be logged.
3. **Statistics-first** — meta-analysis features must prefer auditable statistical routines over opaque generated conclusions.
4. **LLM-assisted, never LLM-owned** — LLMs may suggest query expansions, summaries, and extraction candidates, but claims must link to exact evidence.
5. **CLI/UI parity** — core workflows must live in shared Go services and be exposed through both `rforge` and Fyne where practical.
6. **Local-first privacy** — local projects should run without cloud dependencies unless users explicitly configure external APIs/services.
7. **External-tool governance** — GROBID, OpenSearch, Qdrant, R/metafor, and other tools must be versioned in project lockfiles.
8. **OSS-study safety** — external source clones stay out of source control; ResearchForge stores notes, metadata, licenses, risks, and integration decisions.

## 2. Target repository shape

```text
research-forge/
├── cmd/rforge/                    # CLI entry point
├── internal/
│   ├── app/                       # application wiring and dependency injection
│   ├── config/                    # global and project configuration
│   ├── domain/                    # core domain types and interfaces
│   ├── project/                   # project workspace, manifest, lockfile
│   ├── provenance/                # event log and audit trail
│   ├── ingest/                    # source connector orchestration
│   ├── sources/                   # OpenAlex, Crossref, arXiv, Unpaywall, etc.
│   ├── library/                   # paper records, dedupe, identifiers, exports
│   ├── pdf/                       # PDF acquisition and parser adapters
│   ├── index/                     # full-text and vector indexing adapters
│   ├── screening/                 # include/exclude workflow and PRISMA counts
│   ├── extraction/                # extraction schemas and evidence tables
│   ├── analysis/                  # effect sizes and meta-analysis adapters
│   ├── reports/                   # Markdown/HTML/LaTeX/report assets
│   ├── oss/                       # OSS repository registry and study workflows
│   └── ui/                        # Fyne app shell and view models
├── pkg/                           # stable public packages, only when justified
├── migrations/                    # database migrations
├── assets/                        # icons, templates, static UI/report assets
├── docs/
│   ├── adr/                       # accepted architecture decisions
│   ├── user/                      # user docs
│   ├── developer/                 # developer docs
│   └── research/                  # source/tool study notes
├── examples/                      # sample projects and fixtures
├── testdata/                      # small fixtures, no copyrighted PDFs
├── opensource/
│   ├── README.md                  # clone-workspace rules
│   ├── inventory/                 # committed metadata and study notes
│   └── clones/                    # gitignored local external repos
├── rforge.example.toml
├── go.mod
├── Makefile
└── README.md
```

## 3. Core domain model

Initial domain concepts should be modeled before feature work begins:

- **ResearchProject** — a local workspace containing manifests, source records, documents, review decisions, extraction tables, analyses, reports, and provenance.
- **ProjectManifest** — human-readable project configuration: research question, sources, schemas, storage mode, external services, and export settings.
- **WorkflowLockfile** — machine-written versions and parameters for connectors, parsers, indexes, models, and statistical tools.
- **SourceQuery** — an auditable query sent to OpenAlex, Crossref, arXiv, Unpaywall, or another source.
- **PaperRecord** — normalized scholarly metadata with identifiers, source-specific payloads, and provenance links.
- **DocumentAsset** — local PDF/XML/JATS/HTML/text assets with acquisition legality and checksums.
- **ParsedDocument** — structured sections, references, tables, equations, and passage IDs.
- **EvidenceItem** — extracted structured claim/value linked to exact source passages/assets.
- **ScreeningDecision** — include/exclude/uncertain decision with reviewer, reason, timestamp, and stage.
- **AnalysisRun** — statistical model execution with inputs, scripts, outputs, plots, and versions.
- **ReportBuild** — reproducible exported report with citations, tables, audit appendices, and build metadata.
- **OSSRepositoryStudy** — repository metadata, license, studied components, integration notes, risks, and decisions.

## 4. Milestones

### Milestone 0 — Project foundation

**Goal:** create the Go/Fyne/CLI foundation and reproducibility spine.

Deliverables:

- Go module and initial package layout.
- `rforge` CLI skeleton with help, version, config path, and JSON output convention.
- Fyne app shell with project dashboard placeholder.
- Project workspace creation/open/list commands.
- Project manifest: `rforge.project.toml`.
- Workflow lockfile: `rforge.lock.json`.
- Append-only provenance event log.
- Storage abstraction with SQLite local mode first.
- `rforge doctor` for local dependency checks.
- Initial CI: formatting, linting, tests, vulnerability check.

Validation:

```sh
go test ./...
go run ./cmd/rforge --help
go run ./cmd/rforge project create ./tmp/demo --title "Demo Review"
go run ./cmd/rforge project inspect ./tmp/demo --json
go run ./cmd/rforge doctor --json
```

Exit criteria:

- A user can create a project from CLI.
- The project contains a manifest, lockfile, database, and event log.
- Fyne app starts and can open/show a project placeholder.

### Milestone 1 — Scholarly metadata and library MVP

**Goal:** search/import scholarly metadata and maintain a normalized library.

Deliverables:

- Source connector interfaces.
- OpenAlex connector.
- Crossref connector.
- arXiv connector.
- Unpaywall lookup for OA status and full-text links.
- Query cache with request/response provenance.
- Paper normalization into `PaperRecord`.
- DOI, arXiv ID, title/year/author deduplication.
- Library list/search commands.
- BibTeX, RIS, CSV, JSON import/export MVP.
- Fyne library table and search-result review screen.

Validation:

```sh
go test ./...
rforge search --source openalex --query "high entropy superconductors" --limit 10 --json
rforge search --source arxiv --query "graph neural networks" --limit 10 --json
rforge library list --json
rforge export --format bibtex --output references.bib
```

Exit criteria:

- A user can search at least OpenAlex and arXiv.
- Records are stored once after dedupe.
- Exports are reproducible from the stored library.

### Milestone 2 — OSS repository intelligence MVP

**Goal:** catalog and study open-source tools relevant to the research workflow.

Deliverables:

- `opensource/clones/` gitignore rules and documentation.
- OSS repository registry schema.
- `rforge oss add <owner/repo>`.
- `rforge oss clone <owner/repo>` using shallow clone by default.
- `rforge oss scan --topic <topic>` metadata workflow.
- License detection and risk notes.
- Study-note template for integration decisions.
- Fyne OSS dashboard placeholder/table.

Validation:

```sh
rforge oss add kermitt2/grobid
rforge oss clone kermitt2/grobid --depth 1
rforge oss license-check kermitt2/grobid --json
rforge oss report --area parsers --format markdown
```

Exit criteria:

- External repos are studied without committing clones.
- Committed output is limited to inventory, notes, licenses, and integration findings.

### Milestone 3 — PDF acquisition, parsing, and indexing

**Goal:** retrieve legal full text, parse scholarly documents, and index passages.

Deliverables:

- Legal OA PDF acquisition through Unpaywall and source-provided links.
- Document asset checksums and license/OA metadata.
- GROBID service adapter.
- Parsed sections, references, tables, and passage IDs.
- Parser confidence/warning recording.
- Local full-text search with Bleve or SQLite FTS for single-user mode.
- Optional OpenSearch adapter.
- Optional Qdrant adapter for embeddings.
- `rforge parse`, `rforge index`, and `rforge retrieve` commands.
- Fyne PDF/section view placeholder.

Validation:

```sh
rforge pdf fetch --doi <open-access-doi> --json
rforge parse --paper <paper-id> --parser grobid --json
rforge index rebuild --json
rforge retrieve --query "critical temperature" --limit 5 --json
```

Exit criteria:

- A paper has metadata, legal asset info, parsed sections, passages, and searchable text.
- Retrieval results include exact paper/section/passage provenance.

### Milestone 4 — Screening workflow

**Goal:** support systematic-review screening with auditability and PRISMA counts.

Deliverables:

- Screening stages: title/abstract, full text, final inclusion.
- Include/exclude/uncertain decisions.
- Configurable exclusion reason tags.
- Reviewer attribution.
- Conflict/uncertain queue.
- CSV export/import for external review workflows.
- PRISMA counts generated from event history.
- Basic active-learning prioritization scaffold inspired by ASReview.
- Fyne screening queue UI.

Validation:

```sh
rforge screen configure --reasons reasons.yaml
rforge screen decide --paper <paper-id> --decision exclude --reason wrong-population
rforge screen queue --stage title-abstract --json
rforge prisma counts --json
```

Exit criteria:

- Inclusion/exclusion decisions are auditable.
- PRISMA counts can be regenerated from stored data.

### Milestone 5 — Evidence extraction

**Goal:** extract structured evidence linked to exact source material.

Deliverables:

- Extraction schema definition format.
- Manual extraction CLI and Fyne table UI.
- Evidence items linked to paper, document asset, section, passage, table, figure, or equation where available.
- Validation status: suggested, accepted, rejected, corrected.
- LLM-assisted suggestion adapter behind explicit configuration.
- Export evidence tables to CSV/JSON/Markdown.
- Audit report for unsupported or weakly supported evidence.

Validation:

```sh
rforge extraction schema add extraction.schema.yaml
rforge extract add --paper <paper-id> --field sample_size --value 120 --passage <passage-id>
rforge extract suggest --paper <paper-id> --schema extraction.schema.yaml --json
rforge extract export --format csv --output evidence.csv
```

Exit criteria:

- Every accepted evidence value links to source support.
- LLM output cannot become accepted evidence without review/provenance.

### Milestone 6 — Meta-analysis MVP

**Goal:** run basic auditable meta-analyses from extracted evidence.

Deliverables:

- Effect-size calculation helpers.
- R/metafor adapter.
- Generated R scripts or notebooks.
- Analysis input snapshots.
- Forest plot output.
- Funnel plot output where applicable.
- Heterogeneity metrics.
- Sensitivity-analysis scaffold.
- Fyne analysis setup and result viewer.

Validation:

```sh
rforge analysis prepare --schema extraction.schema.yaml --output analysis-input.csv
rforge analysis run --model random-effects --engine metafor --json
rforge analysis export --format markdown --output analysis.md
```

Exit criteria:

- A user can run one basic meta-analysis from an evidence table.
- Inputs, model, tool versions, scripts, and outputs are reproducible.

### Milestone 7 — Report generation

**Goal:** export a reproducible research report with citations, PRISMA data, evidence tables, analysis outputs, and audit trail.

Deliverables:

- Markdown report builder.
- HTML export.
- LaTeX export scaffold.
- Citation tables and bibliography export.
- PRISMA diagram output.
- Evidence tables.
- Meta-analysis plots/results.
- Audit appendix: queries, sources, decisions, parser versions, extraction links, analysis versions.
- Fyne report builder/export flow.

Validation:

```sh
rforge report build --format markdown --output report.md
rforge report build --format html --output report.html
rforge report audit --json
```

Exit criteria:

- A report can answer what was searched, where records came from, why papers were included/excluded, what supports claims, what model was run, and how to reproduce it.

### Milestone 8 — Hardening and beta release

**Goal:** make the MVP usable by early researchers.

Deliverables:

- Install/release automation for Linux, macOS, and Windows where practical.
- Example research project using open fixtures.
- User documentation.
- Developer documentation.
- Data privacy documentation.
- External service setup documentation.
- Error handling and recovery pass.
- Performance pass for medium libraries.
- Backup/export/import project archive.
- Accessibility and UX pass for Fyne screens.

Validation:

```sh
go test ./...
go test -race ./...
rforge doctor
rforge project archive ./demo --output demo.rforge.zip
rforge project restore demo.rforge.zip ./restored-demo
```

Exit criteria:

- A new user can install ResearchForge, create a project, complete the MVP workflow on open data, and export a reproducible report.

## 5. Implementation sequence by vertical slice

Prefer vertical slices over isolated infrastructure. Each slice should include domain types, persistence, CLI, tests, provenance, and a minimal UI hook when relevant.

1. Project create/open/list.
2. Provenance event log.
3. Manifest and lockfile.
4. OpenAlex search into library.
5. arXiv search into library.
6. Deduplication.
7. Export BibTeX/CSV/JSON.
8. OSS repository registry and clone workspace.
9. Unpaywall OA lookup.
10. PDF asset fetch and checksum.
11. GROBID parse adapter.
12. Passage search.
13. Screening decision model.
14. PRISMA counts.
15. Extraction schema and manual extraction.
16. Evidence export.
17. R/metafor analysis run.
18. Markdown report export.
19. Fyne dashboard/library/search parity.
20. Fyne screening/evidence/report parity.

## 6. Testing strategy

### Unit tests

- Domain validation.
- Identifier normalization.
- Deduplication scoring.
- Manifest and lockfile read/write.
- Provenance event creation.
- Source connector request building.
- Import/export parsers.
- Screening count calculations.
- Extraction schema validation.
- Analysis input generation.

### Integration tests

- SQLite project lifecycle.
- Mock OpenAlex/Crossref/arXiv/Unpaywall servers.
- GROBID adapter against a fixture or mocked TEI output.
- Report generation from a fixture project.
- OSS clone workflow with local test repositories.

### Golden tests

- BibTeX export.
- PRISMA counts.
- Markdown reports.
- Audit appendices.
- CLI JSON output schemas.

### Manual acceptance tests

- Fyne opens project.
- Fyne search results can be reviewed.
- Fyne screening decisions are persisted.
- Fyne evidence table links back to source passages.
- Fyne report export matches CLI-generated report semantics.

## 7. CI and release plan

Initial CI gates:

```sh
gofmt -w
go test ./...
go vet ./...
govulncheck ./...
```

Later CI gates:

- Staticcheck.
- Race tests for selected packages.
- Cross-platform build matrix.
- Fyne packaging smoke tests.
- Fixture-based CLI end-to-end tests.
- Dependency/license scans.
- Reproducible release artifacts.

Release stages:

1. **Pre-alpha:** CLI project/search/library foundation.
2. **Alpha:** end-to-end search -> screening -> extraction -> report on fixtures.
3. **Beta:** real open-access PDF parsing and meta-analysis MVP.
4. **1.0:** documented, reproducible, cross-platform MVP with stable project format.

## 8. Key ADR candidates

Write ADRs only when these decisions are made:

1. CLI framework: Cobra vs urfave/cli.
2. Local storage default: SQLite-first vs PostgreSQL-first.
3. Search backend: SQLite FTS/Bleve local mode vs mandatory OpenSearch.
4. Vector backend: optional Qdrant vs embedded/local alternative.
5. UI architecture: Fyne view models and background job pattern.
6. Provenance format: append-only JSONL/events table vs richer event store.
7. Project archive format and compatibility policy.
8. LLM adapter boundary and review requirements.
9. R/metafor adapter shape and reproducibility contract.

## 9. Open owner decisions

These decisions should be settled before or during Milestone 0:

1. Should SQLite be the default MVP storage mode, with PostgreSQL later? Recommended: yes, because it lowers setup cost and improves local-first usability.
2. Should the first CLI framework be Cobra? Recommended: yes, because `rforge` needs nested commands, help, completion, and mature command organization.
3. Should Fyne be built from the first milestone or after CLI MVP? Recommended: start the shell immediately, but keep feature implementation in shared services and add UI parity after CLI validation.
4. Should Qdrant/OpenSearch be mandatory for MVP? Recommended: no; make them optional service-mode backends after a local search MVP exists.
5. Should LLM features ship in the MVP? Recommended: only behind explicit configuration and never as a required path.

## 10. First two-week execution plan

Week 1:

- Create Go module and folder structure.
- Add Cobra CLI skeleton.
- Add version/build metadata.
- Add project create/open/list commands.
- Define manifest and lockfile formats.
- Add SQLite project database initialization.
- Add append-only provenance log.
- Add basic CI.

Week 2:

- Add `rforge doctor`.
- Add source connector interface.
- Add mocked-source test harness.
- Implement OpenAlex search MVP.
- Persist normalized `PaperRecord` rows.
- Add library list command.
- Add JSON output convention.
- Add Fyne app shell and project dashboard placeholder.

End-of-sprint demo:

```sh
rforge project create ./demo --title "Demo Review"
rforge search --source openalex --query "high entropy superconductors" --limit 10
rforge library list --json
rforge doctor --json
rforge ui ./demo
```

## 11. Definition of done for every feature

A feature is not done unless it has:

- Domain types or interfaces where needed.
- Persistence or explicit stateless rationale.
- Provenance events for user-visible workflow changes.
- CLI path or explicit UI-only rationale.
- Tests for success and failure cases.
- JSON output if used in automation.
- Documentation or help text.
- Privacy/copyright impact reviewed when external data is involved.
- Reproducibility impact recorded in manifest, lockfile, or event log when applicable.
