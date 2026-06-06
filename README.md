# ResearchForge

ResearchForge is an open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, and meta-analysis.

The command-line tool is planned as **`rforge`**.

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

ResearchForge is intended to help researchers discover papers, map citation graphs, screen literature, extract evidence, run meta-analyses, generate auditable reports, and maintain a local knowledge base of relevant open-source research tooling.

## Status

This repository currently contains the product requirements document and early project planning. Implementation has not started yet.

See [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md) for the full product requirements and [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md) for the implementation roadmap.

## Goals

- Cross-source academic search
- Legal open-access full-text discovery
- PDF and structured-document parsing
- Citation graph expansion
- Deduplication and screening workflows
- Evidence extraction linked to source passages
- Meta-analysis and statistical reporting
- Reproducible research reports with audit trails
- CLI and Fyne desktop interfaces
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
  -> CLI + Fyne desktop UI
```

## Planned MVP stack

- **Language:** Go
- **CLI:** `rforge`, likely Cobra or urfave/cli
- **Desktop UI:** Fyne
- **Database:** PostgreSQL, with optional SQLite for local single-user mode
- **Search:** OpenSearch, with optional Bleve for local mode
- **Vector database:** Qdrant
- **PDF parsing:** GROBID as an external service
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

No license has been selected yet.
