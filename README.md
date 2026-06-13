# ResearchForge

ResearchForge is an open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, and meta-analysis.

The command-line tool is **`rforge`**.

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

ResearchForge is intended to help researchers discover papers, map citation graphs, screen literature, extract evidence, run meta-analyses, generate auditable reports, visualize CLI-generated papers/diagrams/statistical artifacts locally, and maintain a local knowledge base of relevant open-source research tooling.

## Status

This repository is in **pre-alpha implementation**. It includes the PRD and planning docs plus a Go `rforge` CLI foundation with local project workspaces, provenance, scholarly source connectors, document/retrieval workflow slices, screening, evidence, analysis, reporting, and deterministic tests.

See [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md) for the full product requirements, [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md) for the implementation plan, [ROADMAP.md](./ROADMAP.md) for milestone sequencing, [TODO.md](./TODO.md) for the end-to-end task checklist, and [SKILLS.md](./SKILLS.md) for the TDD-only project development skills.

## Goals

- Cross-source academic search
- Legal open-access full-text discovery
- PDF and structured-document parsing
- Citation graph expansion
- Deduplication and screening workflows
- Evidence extraction linked to source passages
- Meta-analysis and statistical reporting
- Reproducible research reports with audit trails
- CLI and local web GUI interfaces
- Continuous study of relevant open-source scholarly tooling

## Target users

- Academic researchers
- Graduate students
- Independent scientists
- Research engineers
- Literature-review authors
- Meta-analysis authors
- R&D teams in physics, engineering, computer science, mathematics, and materials science

## Planned architecture

```text
Research question / domain query
  -> Query planner
  -> Go application core
  -> Ingestion connectors
  -> Document store
  -> Parsing and extraction
  -> Indexing
  -> Review engine
  -> Evidence extraction
  -> Meta-analysis/statistics
  -> Report generator
  -> CLI + local web GUI
```

## Planned MVP stack

- **Language:** Go
- **CLI:** `rforge`, currently using a standard-library parser while the command surface stabilizes
- **Local web GUI:** Go + HTMX local research cockpit launched by `rforge ui` for project review, artifact navigation, guided local actions, and embedded visualization libraries where needed
- **Database:** SQLite-first local storage, with a PostgreSQL adapter seam prepared for later
- **Search:** local retrieval index first; OpenSearch/Qdrant remain optional future adapter seams
- **Vector database:** Qdrant planned as an optional adapter seam
- **PDF parsing:** GROBID as an optional external parser adapter
- **Metadata sources:** OpenAlex, Crossref, arXiv, Unpaywall, and related scholarly APIs
- **Meta-analysis:** R `metafor` integration initially

## Planned CLI examples

```sh
rforge project create "superconductor-review"
rforge search --source openalex --query "high entropy superconductors"
rforge ingest bibtex references.bib
rforge dedupe
rforge screen export --format csv
rforge report build
```

OSS repository study examples:

```sh
rforge oss add kermitt2/grobid
rforge oss scan --topic "meta-analysis"
rforge oss report --area parsers
```

## Design principle

ResearchForge should not be an opaque AI answer machine. It should be a scientific workflow engine that can answer:

- What did we search?
- Where did each paper come from?
- Why was each paper included or excluded?
- What exact source supports each extracted claim?
- What statistical model was run?
- Can another researcher reproduce the result?

## License

No license has been selected yet. This is an owner decision tracked in issue #1 and [docs/owner-decisions.md](docs/owner-decisions.md); `rforge decisions` lists the currently blocked decisions. The remaining owner inputs are the license choice and exact copyright holder string; the required owner response fields are the license SPDX identifier, approver, and approval date. Run `make license-decision-live-audit` to inspect issue #1 and `make license-decision-approval-gate` before adding `LICENSE`.

## Decision-gated scope

Local web GUI delivery now targets Go + HTMX; the implementation tracker is recorded in issue #2 and ADR 0006. Run `make todo-audit` to verify that remaining unchecked `TODO.md` items are covered by owner decisions, `make todo-completion-audit` for the closeout prompt-to-artifact checklist, or `make decisions-markdown` for a review-friendly blocker table.
