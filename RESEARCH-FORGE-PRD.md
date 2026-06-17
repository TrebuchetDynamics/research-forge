# ResearchForge PRD: Open Academic Research and Meta-Analysis Engine

## 1. Purpose

Build **ResearchForge**, an open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, and meta-analysis across scientific domains including physics, engineering, computer science, mathematics, materials science, and adjacent fields.

The command-line tool name is **`rforge`**.

The product must be implemented primarily in **Go/Golang**, with both a **CLI** and a **local web GUI**. The engine should continuously study relevant open-source repositories, APIs, research tools, and academic-data projects so the system can keep improving its connectors, extraction strategies, workflows, and implementation choices.

The engine should combine mature open-source scholarly tooling with a provenance-first architecture. It should help researchers discover papers, map citation graphs, screen literature, extract evidence, run meta-analyses, generate auditable research reports, and maintain an evolving knowledge base of the open-source research ecosystem.

Core principle:

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

LLMs may assist with query expansion, screening rationales, extraction suggestions, and report prose, but every suggested sentence must trace back to exact source metadata, passages, tables, equations, datasets, figures, or citations and remain unaccepted until reviewer approval.

---

## 2. Target Users

- Academic researchers
- Graduate students
- Independent scientists
- Research engineers
- Literature-review authors
- Meta-analysis authors
- R&D teams in physics, engineering, CS, math, and materials science

---

## 3. Problem Statement

Scientific research workflows are fragmented across reference managers, PDF folders, search engines, spreadsheets, statistical packages, and manual notes. Existing tools solve parts of the workflow but rarely provide an integrated, reproducible system for:

- Cross-source academic search
- Legal full-text discovery
- PDF parsing
- Citation graph expansion
- Systematic review screening
- Evidence extraction
- Meta-analysis
- Audit trails and reproducible reports

The goal is to build a modular Go engine that composes best-in-class open-source projects and APIs into one research workflow, while also keeping a local, queryable registry of open-source projects it has studied.

---

## 4. Open-Source Projects and Components to Study or Use

### 4.1 Reference and Library Management

| Project | Repository | Use | Notes |
|---|---|---|---|
| Zotero | `zotero/zotero` | Paper collection, metadata, citations, annotations | Best open-source reference-manager ecosystem. Good model for local library, sync, citation UX. |
| JabRef | `JabRef/jabref` | BibTeX/BibLaTeX management | Excellent for LaTeX-heavy fields: physics, math, CS. |

### 4.2 Scholarly Metadata and Discovery Sources

| Source | Use | Notes |
|---|---|---|
| OpenAlex | Cross-disciplinary paper, author, institution, concept, and citation graph | Essential open scholarly graph. |
| Crossref | DOI metadata, publishers, references | Strong metadata backbone. |
| arXiv | Physics, math, CS, engineering preprints | Essential for target domains. |
| PubMed / Europe PMC | Biomedical, bioengineering, medical physics, materials-adjacent literature | Useful for interdisciplinary searches. |
| Semantic Scholar API / S2ORC | Citation data, paper metadata, embeddings/full text where available | Useful; verify API and data terms. |
| Unpaywall | Legal open-access full-text discovery | Important for copyright-safe full-text acquisition. |
| DOAJ / CORE | Open-access paper discovery | Useful secondary full-text sources. |
| NASA ADS | Physics and astronomy literature | Very strong for astrophysics/physics. |

### 4.3 PDF and Full-Text Parsing

| Project | Repository | Use | Notes |
|---|---|---|---|
| GROBID | `kermitt2/grobid` | Extract titles, authors, abstracts, references, sections from PDFs | Recommended primary scholarly PDF parser. Apache-2.0. |
| s2orc-doc2json | `allenai/s2orc-doc2json` | Convert PDF/LaTeX/JATS to structured JSON | Good S2ORC-style document representation. |
| Science Parse | `allenai/science-parse` | Scientific PDF parsing | Older but useful to study. |
| PaperMage | `allenai/papermage` | NLP/CV representation of scientific papers | Useful for rich document modeling. |
| Anystyle | citation parsing | Parse references and bibliographies | Useful for reference cleanup. |
| CERMINE | scientific article extraction | Metadata/reference extraction | Older Java tool, still worth studying. |

Recommendation: start with GROBID for PDFs and prefer JATS/XML when available from publishers, PubMed Central, or arXiv sources.

### 4.4 Screening and Systematic Review Tools

| Project | Repository | Use | Notes |
|---|---|---|---|
| ASReview | `asreview/asreview` | Active-learning screening for systematic reviews | Highly relevant. Human-in-the-loop screening with fewer paper reviews. |
| RobotReviewer | `ijmarshall/robotreviewer` | Automated evidence extraction from RCTs | Biomedical-focused but useful design reference. |
| revtools | R ecosystem | Screening/dedup workflows | Useful systematic review reference. |
| metagear | R ecosystem | Research synthesis tools | Useful for systematic review workflows. |

Recommendation: imitate ASReview's core idea: human-in-the-loop relevance screening with active learning and auditable labels.

### 4.5 Meta-Analysis and Statistics

| Project | Language | Use |
|---|---|---|
| metafor | R | Gold-standard meta-analysis package: fixed/random effects, heterogeneity, forest/funnel plots, meta-regression. |
| meta | R | General meta-analysis workflows. |
| metagear | R | Research synthesis tools. |
| PyMARE | Python | Python meta-analysis and regression engine. |
| statsmodels | Python | Statistical modeling support. |
| PyMC / Stan / brms | Python/R | Bayesian meta-analysis. |

Recommendation: initially call R `metafor` from the engine for statistical correctness, then add Python-native implementations as needed.

### 4.6 Scientific NLP

| Project | Use |
|---|---|
| SciSpaCy | Scientific/biomedical entity recognition. |
| BERTopic | Topic modeling and research cluster discovery. |
| SentenceTransformers | Embeddings for semantic search. |
| KeyBERT | Keyword extraction. |
| spaCy | General NLP pipeline. |
| OpenAlex concepts/citation graph | Domain mapping and research clustering. |

Domain-specific model candidates:

- SciBERT
- SPECTER / SPECTER2
- MatSciBERT
- ChemBERTa
- domain-specific Hugging Face models for materials science, physics, chemistry, and engineering

### 4.7 Search, Retrieval, and RAG Infrastructure

| Project | Use |
|---|---|
| OpenSearch | Keyword and full-text search. |
| Qdrant | Vector database; strong local/production choice. |
| Milvus | Large-scale vector search. |
| Weaviate | Vector + object storage. |
| Chroma | Simple local vector DB. |
| Haystack | Retrieval/search/RAG pipelines. |
| LlamaIndex | Document ingestion and retrieval agents. |
| LangChain | Agent/tool orchestration; useful, but core logic should remain independent. |

Recommended backend search stack:

- PostgreSQL for canonical metadata and workflow state
- OpenSearch for lexical/full-text search
- Qdrant for semantic/vector search

### 4.8 Go/CLI/Web GUI Implementation References

The project should study these implementation ecosystems while remaining product-owned and modular:

| Area | Candidate projects/libraries | Use |
|---|---|---|
| Local web GUI | Go + HTMX, D3/Cytoscape/Plotly/Vega-Lite | Browser-based local visualization for project state and CLI-generated artifacts. |
| CLI | Cobra, urfave/cli, Bubble Tea | Command structure, TUI workflows, scripting. |
| HTTP/API clients | Go standard library, Resty | Metadata source connectors. |
| Storage | pgx, database/sql, SQLite for local mode | Canonical project and paper storage. |
| Search | OpenSearch Go client, Bleve for local mode | Full-text retrieval. |
| Vector search | Qdrant Go client | Semantic retrieval. |
| Graph | Cayley, Gonum graph, custom PostgreSQL schema | Citation graph and repository dependency graph. |
| Reports | Go templates, Goldmark, LaTeX export | Markdown/HTML/LaTeX generation. |
| Plots | gonum/plot, Vega-Lite export | Forest plots, funnel plots, trend charts. |

### 4.9 Continuous Open-Source Repository Study

The engine should include an OSS intelligence subsystem that periodically studies useful open-source projects.

Requirements:

- Track repositories by topic: scholarly PDF parsing, metadata APIs, systematic review, meta-analysis, search, vector databases, scientific NLP, citation graphs, web GUI, Go CLI tooling.
- Pull repository metadata from GitHub/GitLab/source archives where allowed.
- Store repo name, URL, license, stars, activity, language, tags, README summary, release status, and integration notes.
- Support manual notes: why a repo matters, risks, useful APIs, whether to integrate, imitate, or avoid.
- Detect stale, archived, or license-incompatible projects.
- Generate periodic ecosystem reports: "best parser candidates", "best Go CLI libraries", "new systematic review tools", etc.
- Never auto-copy code into the product without license review and human approval.

### 4.10 Local Open-Source Clone Workspace

ResearchForge should support an optional local clone workspace for deeper study of open-source projects without mixing external code into product source.

Recommended layout:

```text
opensource/
  README.md
  inventory.json
  clones/
    grobid/
    asreview/
    zotero/
  notes/
    grobid.md
    asreview.md
  reports/
    parser-comparison.md
```

Requirements:

- Keep `opensource/clones/` gitignored by default to avoid vendoring third-party code accidentally.
- Commit only curated notes, inventories, comparison matrices, and license-review records.
- Clone with explicit commands such as `rforge oss clone kermitt2/grobid --into opensource/clones`.
- Record source remote URL, commit SHA, clone timestamp, license, and study purpose.
- Support shallow clones by default and full clones only when history analysis is needed.
- Mark each studied project as `integrate`, `adapter-only`, `pattern-reference`, `avoid`, or `needs-license-review`.
- Add a hard rule: local clones are for inspection and testing only; no source copying into ResearchForge without human license approval.

---

## 5. Proposed System Architecture

```text
                 ┌────────────────────────┐
                 │ Research question /    │
                 │ PICO / domain query    │
                 └───────────┬────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │ Query planner                            │
        │ expands terms, synonyms, concepts, DOI   │
        └───────────┬─────────────────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Go application core                  │
 │ shared domain services for CLI/web UI   │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Ingestion layer                      │
 │ OpenAlex, Crossref, arXiv, PubMed,   │
 │ Semantic Scholar, Unpaywall, ADS     │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Document store                       │
 │ metadata, PDF, XML, references,      │
 │ citation graph, provenance           │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Parsing/extraction                   │
 │ GROBID, doc2json, OCR if needed      │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Indexing                             │
 │ PostgreSQL + OpenSearch + Qdrant     │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Review engine                        │
 │ dedup, inclusion/exclusion, ASReview │
 │ active learning, PRISMA tracking     │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Evidence extraction                  │
 │ variables, methods, sample sizes,    │
 │ equations, materials, datasets       │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Meta-analysis/statistics             │
 │ effect sizes, heterogeneity, bias,   │
 │ forest/funnel plots, sensitivity     │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Report generator                     │
 │ PRISMA diagram, citations, tables,   │
 │ reproducible notebooks, audit trail  │
 └──────────────────┬──────────────────┘
                    │
 ┌──────────────────▼──────────────────┐
 │ Interfaces                           │
 │ CLI + local web GUI                     │
 └─────────────────────────────────────┘
```

---

## 6. Minimum Viable Product Stack

- Primary language: Go/Golang
- Application core: Go domain services shared by CLI and web GUI
- CLI: `rforge`, a Go command-line interface, preferably Cobra or urfave/cli
- Local web GUI: browser-based app served locally by `rforge ui`; Go + HTMX is the selected local web GUI stack, using server-rendered local-first screens with progressive enhancement and optional visualization libraries for rich charts/graphs
- Database: PostgreSQL for server/workstation mode; optional SQLite for local single-user mode
- Search: OpenSearch for full deployments; optional Bleve for local mode
- Vector database: Qdrant
- PDF parser: GROBID as an external service called from Go
- Metadata ingestion: Go connectors for OpenAlex + Crossref + arXiv + Unpaywall
- OSS repository study: GitHub/GitLab/source connector plus local repository registry
- Screening: integrate or imitate ASReview concepts; implement product workflow in Go
- Meta-analysis: call R `metafor` from Go initially; later add Go-native/Python/R adapters where useful
- UI prototype: local web GUI from the start, focused on visualizing papers, meta-analysis results, PRISMA/citation diagrams, and report artifacts generated by the CLI
- Reproducibility: every query, source, exclusion, extraction, repository-study note, and analysis must be logged

---

## 7. Core Product Features

Implementation rule: ResearchForge may study and adapt patterns from open-source tools, but it must not copy code, assets, schemas, or documentation text without license review and explicit approval. "Steal" here means learn from proven workflows, UX patterns, APIs, and architecture ideas legally.

### 7.1 Search Strategy Builder

Requirements:

- Boolean query construction
- Synonym and concept expansion
- Field-specific search: title, abstract, full text, references
- Source-specific query translation for OpenAlex, Crossref, arXiv, PubMed, etc.
- Search versioning and saved strategies

Open-source/reference tools to study:

- Zotero advanced search and saved searches — query UX and library filtering.
- JabRef search/groups — BibTeX-centric filtering and saved groups.
- ASReview project search/import flow — review-oriented search setup.
- OpenAlex, Crossref, arXiv, PubMed APIs — source-specific query constraints and pagination.
- OpenSearch query DSL — boolean and fielded full-text search model.
- Semantic Scholar API — academic search result ranking and citation expansion patterns.

### 7.2 Ingestion

Requirements:

- Import from OpenAlex, Crossref, arXiv, PubMed, Semantic Scholar, Unpaywall, NASA ADS
- Import user PDFs and BibTeX/RIS files
- Store canonical paper records
- Preserve source-specific IDs: DOI, arXiv ID, PubMed ID, OpenAlex ID, Semantic Scholar ID
- Record source, query, timestamp, and API response provenance

Open-source/reference tools to study:

- Zotero translators/importers — multi-source metadata ingestion patterns.
- JabRef import/export — BibTeX, RIS, DOI, arXiv, and library import workflows.
- rOpenSci `rcrossref`, `rentrez`, and `fulltext` — scholarly API connector patterns.
- S2ORC/doc2json — structured corpus ingestion formats.
- Unpaywall data/API — legal open-access full-text resolution.
- arXiv API clients — preprint metadata and PDF discovery behavior.

### 7.3 Deduplication

Requirements:

- DOI exact matching
- arXiv/PubMed/OpenAlex ID matching
- Title fuzzy matching
- Author/year matching
- Manual merge/split workflow
- Dedup audit trail

Open-source/reference tools to study:

- Zotero duplicate detection/merge UX — human-confirmed duplicate workflows.
- JabRef cleanup/quality checks — bibliographic normalization ideas.
- ASReview import dedup behavior — systematic-review-safe duplicate handling.
- OpenAlex ID graph — canonical IDs and cross-source entity resolution.
- Crossref DOI normalization rules — DOI-based canonical matching.

### 7.4 Citation Graph

Requirements:

- Backward citations: papers cited by a paper
- Forward citations: papers citing a paper
- Co-citation clusters
- Bibliographic coupling
- Research lineage view
- Citation graph export

Open-source/reference tools to study:

- OpenAlex works graph — cited-by, references, concepts, authors, institutions.
- Semantic Scholar graph APIs — citation velocity, influential citations, related papers.
- Connected Papers / Litmaps / ResearchRabbit concepts — visual lineage and discovery UX to emulate, not copy.
- Cayley and Gonum graph — Go graph modeling and traversal.
- Gephi formats — graph export interoperability.

### 7.5 Screening and Review Workflow

Requirements:

- Include/exclude decisions
- Exclusion reason tags
- Reviewer assignment
- Conflict resolution between reviewers
- Active-learning prioritization
- PRISMA flow tracking
- Reproducible screening history

Open-source/reference tools to study:

- ASReview — active-learning screening, labeling, project state, reviewer loop.
- RobotReviewer — automated evidence extraction and risk-of-bias concepts.
- revtools/metagear — systematic review screening and dedup workflow ideas.
- PRISMA 2020 templates/tools — flow diagram categories and reporting requirements.
- Covidence/Rayyan concepts — reviewer assignment/conflict-resolution UX to emulate, not copy.

### 7.6 Evidence Extraction

Requirements:

- Extract structured fields from papers:
  - Methods
  - Datasets
  - Materials
  - Experimental conditions
  - Sample sizes
  - Equations
  - Variables
  - Measurement units
  - Uncertainty/error bars
  - Outcomes
- Support manual verification
- Link each extracted datum to source passage/table/figure/page
- Allow schema templates per domain

Example domain templates:

- Physics experiment template
- Materials property template
- ML benchmark template
- Engineering performance template
- Mathematical theorem/result template

Open-source/reference tools to study:

- GROBID — section/reference/table-adjacent structure from scholarly PDFs.
- PaperMage — paper object model for passages, figures, tables, and layout-aware extraction.
- SciSpaCy — scientific entity extraction patterns.
- S2ORC/doc2json — normalized paper JSON structure.
- Science Parse and CERMINE — alternative parser outputs and failure modes.
- MatSciBERT/SciBERT/SPECTER models — domain-aware embeddings/classification approaches.
- Label Studio — human verification and annotation workflow concepts.

### 7.7 Meta-Analysis Module

Requirements:

- Effect-size calculation
- Fixed-effects models
- Random-effects models
- Heterogeneity metrics: I², τ², Q
- Meta-regression
- Publication bias checks
- Funnel plots
- Forest plots
- Sensitivity analysis
- Subgroup analysis
- Export to R notebooks/scripts
- Reproducible statistical reports

Open-source/reference tools to study:

- R `metafor` — primary reference for meta-analysis correctness and model coverage.
- R `meta` — general meta-analysis workflows and report outputs.
- PyMARE — Python-native meta-analysis/regression design reference.
- RevMan concepts — forest/funnel plot and systematic-review output conventions.
- JASP/Jamovi concepts — user-friendly statistical UI patterns to emulate, not copy.
- Gonum/stat and gonum/plot — Go-native statistical and plotting primitives where appropriate.

### 7.8 Report Generator

Requirements:

- Generate PRISMA diagrams
- Generate citation tables
- Generate evidence tables
- Generate forest/funnel plots
- Generate reproducible notebooks
- Generate Markdown/LaTeX/HTML reports
- Include all provenance and exclusion logs

Open-source/reference tools to study:

- Quarto/Pandoc concepts — reproducible scientific report generation and multi-format export.
- R Markdown concepts — literate analysis and executable report structure.
- PRISMA diagram tools — systematic-review flow reporting.
- Zotero/JabRef citation export — bibliography and citation formatting workflows.
- CSL styles ecosystem — citation style support.
- Goldmark and Go templates — Go-native Markdown/HTML generation.

### 7.9 CLI Interface

Requirements:

- Create/list/open research projects.
- Run searches from terminal scripts.
- Import/export BibTeX, RIS, CSV, JSON, Markdown, and LaTeX assets.
- Start/stop/check external services such as GROBID, OpenSearch, and Qdrant when configured.
- Run deduplication, screening exports, parsing jobs, indexing jobs, and reports in batch mode.
- Run OSS repository study jobs, for example:
  - `rforge oss add kermitt2/grobid`
  - `rforge oss scan --topic "meta-analysis"`
  - `rforge oss report --area parsers`
- Provide machine-readable JSON output for automation.

Open-source/reference tools to study:

- Cobra — command hierarchy, help text, shell completion, config patterns.
- urfave/cli — lightweight Go CLI patterns.
- Git CLI — subcommand ergonomics and scriptability.
- GitHub CLI `gh` — authenticated API workflows and JSON output flags.
- DVC — reproducible data/workflow command patterns.
- ASReview CLI — systematic-review automation concepts.

### 7.10 Local Web GUI

Purpose: the Go + HTMX interface is a **local research cockpit** for human review, navigation, and small local workflow actions over ResearchForge projects and CLI-generated artifacts. The CLI remains the authoritative reproducible automation path.

Requirements:

- Run as a local browser UI launched by `rforge ui`, not as a native Fyne desktop application.
- Project dashboard for local project health, recent CLI runs, pending reviews, and generated artifacts.
- Search builder and result review.
- Paper library with metadata, PDF/section view, notes, and citation graph.
- Local visualization workspace for papers, citation graphs, PRISMA diagrams, evidence tables, forest/funnel plots, sensitivity outputs, and report assets generated by the CLI.
- Screening workflow with include/exclude decisions and reason tags.
- Evidence extraction tables linked to source passages.
- Meta-analysis setup and result visualization.
- Report/artifact browser that can open Markdown/HTML/LaTeX outputs and diagram assets from the project workspace.
- Guided local workflow actions for review tasks such as project open/create, artifact refresh, search review, library navigation, and source-link inspection, backed by shared Go services and provenance.
- OSS repository intelligence dashboard showing studied repos, licenses, risks, integration notes, and ecosystem reports.

Open-source/reference tools to study:

- Go + HTMX (the "Go HTMLX" option named in planning notes) — selected stack for a server-rendered local UI with progressive enhancement and minimal frontend infrastructure.
- Vite, D3, Cytoscape.js, Plotly, Vega-Lite — visualization and charting for citation graphs, PRISMA diagrams, and meta-analysis plots.
- Zotero web/desktop UX — library/sidebar/detail-pane/notes workflow.
- JabRef UI — BibTeX library management and cleanup dialogs.
- ASReview UI — screening workflow and active-learning progress presentation.
- Gephi/Cytoscape concepts — graph exploration UX to emulate, not copy.
- Qdrant/OpenSearch dashboards — index/status observability ideas.

### 7.11 OSS Repository Intelligence

Requirements:

- Maintain an internal catalog of open-source projects relevant to academic research tooling.
- Continuously refresh repository metadata on user command or schedule.
- Summarize README/docs/releases into structured notes.
- Compare alternatives by license, maintenance, language, maturity, and integration cost.
- Connect repo-study findings to implementation decisions.
- Preserve citations/links to the studied repositories and docs.

Open-source/reference tools to study:

- GitHub CLI `gh` and GitHub API — repository metadata, releases, topics, issues, license info.
- Libraries.io concepts — dependency/package ecosystem intelligence.
- OpenSSF Scorecard — project health and supply-chain risk signals.
- Deps.dev concepts — dependency graph and package metadata.
- Snyk/OSV concepts — vulnerability metadata and risk reporting.
- Homebrew/Nix package metadata concepts — installability and release tracking.

### 7.12 Automatic Paper Discovery and Open-Access Download

Requirements:

- Saved watched searches by topic, source, query, filters, and schedule.
- Manual and scheduled refresh of watched searches.
- New-paper inbox with change history since the last run.
- Automatic metadata ingestion by default.
- Optional automatic download of legal open-access PDFs only.
- Never bypass paywalls or restricted access controls.
- Track download URL, license, source, checksum, and retrieval timestamp.
- Allow user approval before importing or downloading, configurable per project.
- CLI examples:
  - `rforge watch add "ferroelectric HZO compute in memory" --interval weekly`
  - `rforge watch run`
  - `rforge inbox`
  - `rforge fetch pdfs --open-access-only`

Open-source/reference tools to study:

- Zotero connector/save workflows — user-controlled capture and PDF attachment behavior.
- Unpaywall — legal OA PDF discovery.
- arXiv APIs — reliable preprint PDF download.
- PubMed Central / Europe PMC — OA full-text acquisition.
- rOpenSci `fulltext` — legal full-text retrieval patterns.
- RSS/Atom feed readers — scheduled discovery/inbox workflow.
- GitHub Actions/cron concepts — scheduled job patterns for refresh automation.

---

## 8. Non-Functional Requirements

### 8.1 Reproducibility

Every result must be traceable to:

- Query string
- Source API/database
- Retrieval timestamp
- Paper/version identifier
- Parser version
- Extractor version
- Human reviewer decision
- Statistical method and parameters

### 8.2 Auditability

The system must maintain:

- Immutable event log for research workflows
- Paper ingestion provenance
- Screening decision history
- Extraction provenance
- Analysis configuration snapshots

### 8.3 Copyright Safety

The system should:

- Prefer open-access full text
- Use Unpaywall and official OA links
- Store source URLs and license metadata
- Avoid unauthorized redistribution of copyrighted PDFs
- Separate metadata records from local user-provided PDFs

### 8.4 Extensibility

The system should support plugins/connectors for:

- New metadata sources
- New parsers
- New domain extraction schemas
- New statistical models
- New visualization modules
- New open-source repository hosts and package registries

### 8.5 Interface Parity

Core workflows must be available from both CLI and web GUI where practical. The CLI is required for automation and reproducibility; the web GUI is required for interactive research, screening, extraction, graph exploration, and visual analysis.

### 8.6 Go-First Maintainability

The core product should remain Go-first. External systems such as GROBID, R `metafor`, Python NLP tools, or vector databases may be used through stable adapters, subprocesses, containers, or HTTP APIs, but the project orchestration, data model, CLI, and web GUI should be owned in Go.

### 8.7 Privacy and Sensitive Research Data

ResearchForge should treat research projects as private by default.

Requirements:

- Keep local projects local unless the user explicitly configures sync or export.
- Redact API keys, local file paths, reviewer names, and private notes from shareable reports unless included intentionally.
- Support per-project data-retention policies for PDFs, extracted passages, embeddings, logs, and temporary parser outputs.
- Document which external APIs receive queries, titles, abstracts, full text, or embeddings.

### 8.8 Supply-Chain and External Tool Governance

Because ResearchForge orchestrates many external tools, it should track runtime components explicitly.

Requirements:

- Record versions and container image digests for GROBID, OpenSearch, Qdrant, R, `metafor`, Python tools, and model files.
- Provide a `rforge doctor` command that checks service health, versions, API credentials, storage paths, and migration status.
- Prefer pinned versions in reproducible project manifests.
- Track vulnerabilities and license notices for bundled or recommended dependencies.

### 8.9 Project Manifest, Repository Config, and Workflow Lockfile

Each research project should have a portable manifest and lockfile.

When `rforge` is run inside an existing repository, ResearchForge should create repo-local `.researchforge` configuration and use `<repo>/research-forge/` as the default Research project folder. The repository may already contain academic files, PDFs, notes, datasets, or other research assets. ResearchForge must treat those assets as pre-existing local material and discover or import them only through explicit, provenance-recorded workflow steps.

The main deterministic end-to-end test topic is **artificial photosynthesis**. Fixtures, example searches, and local-first e2e scenarios should prefer this topic unless a slice has a stronger domain-specific reason to use another topic.

Example files:

- `rforge.project.toml` — user-authored project settings, sources, schemas, watched searches, and output preferences.
- `rforge.lock.json` — generated versions, connector parameters, parser versions, model IDs, service digests, and statistical engine versions.

This makes analyses rerunnable across machines and reviewable in version control.

---

## 9. Initial Roadmap

### Phase 0: Go/CLI/Web GUI Foundation

- Create Go module and application architecture.
- Define shared domain services used by both CLI and web GUI.
- Implement CLI skeleton.
- Implement local web GUI shell.
- Add project/workspace model.
- Add event log/provenance store.
- Add project manifest and workflow lockfile.
- Add `rforge doctor` for dependency/service checks.
- Add OSS repository catalog schema.
- Add gitignored `opensource/clones/` workspace plus committed notes/inventory structure.

### Phase 1: Research Library, Search MVP, and OSS Study MVP

- Connect OpenAlex, Crossref, arXiv, and Unpaywall using Go connectors.
- Store paper metadata in PostgreSQL or SQLite local mode.
- Implement DOI/title deduplication.
- Build first web GUI screens for project dashboard, search, and library.
- Build CLI commands for search/import/export.
- Add OSS repository scanner for manually supplied GitHub repositories.
- Add `rforge oss clone`, `rforge oss note`, and `rforge oss license-check` workflows for local clone study.
- Export BibTeX/CSV/JSON.

### Phase 2: Parsing and Indexing

- Add GROBID service
- Parse PDFs into structured sections/references
- Add OpenSearch full-text index
- Add Qdrant semantic index
- Implement passage-level retrieval with citations

### Phase 3: Screening Workflow

- Add include/exclude labels
- Add exclusion reasons
- Add reviewer workflow
- Add ASReview-style active-learning prioritization
- Add PRISMA counts

### Phase 4: Evidence Extraction

- Add extraction schemas
- Add manual extraction UI
- Add LLM-assisted extraction suggestions
- Link extracted facts to source passages
- Add validation/review status for extracted facts

### Phase 5: Meta-Analysis

- Integrate R `metafor`
- Add effect-size calculators
- Add forest/funnel plots
- Add heterogeneity and sensitivity analysis
- Generate reproducible statistical notebooks

### Phase 6: Research Report Generation and Ecosystem Reports

- Markdown/LaTeX/HTML report output
- PRISMA diagram generation
- Evidence tables
- Citation tables
- Audit appendix
- OSS ecosystem reports for studied repositories
- CLI and web GUI export flows

---

## 10. Recommended Study Order

1. Local web GUI architecture — Go + HTMX local server boundaries, artifact loading, visualization state, and background-job updates.
2. Go CLI architecture — Cobra/urfave/cli command design, JSON output, config handling.
3. GROBID — scholarly PDF parsing.
4. OpenAlex API/data model — scholarly graph and metadata.
5. ASReview — systematic-review screening with active learning.
6. Zotero — reference-library UX and data model.
7. metafor — statistically correct meta-analysis.
8. S2ORC/doc2json/PaperMage — structured scientific-paper representation.
9. Qdrant/OpenSearch — retrieval backend.
10. GitHub/GitLab repository metadata APIs — continuous open-source ecosystem study.

---

## 11. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| LLM hallucination | Require source-grounded outputs with exact passage/table citations. |
| Bad PDF parsing | Prefer JATS/XML when available; expose parser confidence; allow manual correction. |
| Copyright issues | Use Unpaywall/OA sources; track licenses; avoid redistribution. |
| Statistical mistakes | Delegate first version to R `metafor`; include reproducible scripts. |
| Duplicate records | Multi-key dedup: DOI, IDs, fuzzy title, author/year. |
| API rate limits | Cache responses; support batch import; source adapters with backoff. |
| Domain mismatch | Use pluggable extraction schemas per scientific field. |
| Go ecosystem gaps | Use Go orchestration with external service adapters for mature non-Go tools. |
| Local web GUI server or visualization drift | Keep long-running parsing/indexing/network jobs in Go services; expose project artifacts through stable APIs/files; make browser state reloadable from project provenance. |
| License contamination from OSS study | Store notes and metadata only; keep local clones gitignored; require human approval before using external code. |
| Reproducibility drift from external services | Pin tool versions and container digests in `rforge.lock.json`; record API parameters and parser/model versions. |
| Privacy leakage through APIs or reports | Make local-only the default; document outbound data flows; redact sensitive fields from shareable outputs. |
| Clone workspace bloat | Use shallow clones by default; allow pruning; keep clones out of normal source control. |

---

## 12. Success Criteria

The MVP is successful when a researcher can:

1. Create a research project from the CLI and open it in the local web GUI.
2. Search OpenAlex/Crossref/arXiv for a topic.
3. Deduplicate imported records.
4. Retrieve legal open-access PDFs where available.
5. Parse papers with GROBID.
6. Screen papers with include/exclude reasons.
7. Extract structured evidence into tables.
8. Run a basic meta-analysis.
9. Study and catalog relevant open-source repositories from the CLI and view them in the local web GUI.
10. Visualize CLI-generated papers, citation graphs, PRISMA diagrams, meta-analysis plots/tables, and report artifacts in the local web GUI.
11. Export a reproducible report with citations and provenance.

---

## 13. Key Design Principle

Do not build an opaque AI answer machine. Build a scientific workflow engine.

The engine should answer:

- What did we search?
- Where did each paper come from?
- Why was each paper included or excluded?
- What exact source supports each extracted claim?
- What statistical model was run?
- Can another researcher reproduce the result?
