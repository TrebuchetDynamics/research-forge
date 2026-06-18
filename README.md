# ResearchForge

[![CI](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml)
[![Playwright e2e](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/research-forge.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/research-forge)

ResearchForge is an open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, and meta-analysis.

The command-line tool is **`rforge`**.

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

ResearchForge helps researchers discover papers, map citation graphs, screen literature, extract evidence, run meta-analyses, generate auditable reports, visualize artifacts locally, and maintain a knowledge base of relevant open-source scholarly tooling.

## Status

**v0.1.0 — pre-alpha.** The `rforge` CLI is functional with 607 passing tests across 29 packages. Core workflows — project workspaces, scholarly source connectors, document parsing, citation graph expansion, screening, evidence extraction, analysis, reporting, and local web GUI — are implemented and test-covered. The project format, CLI surface, and APIs may change before 1.0.

See [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md) for full product requirements, [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md) for the implementation plan, [ROADMAP.md](./ROADMAP.md) for milestone sequencing, and [SKILLS.md](./SKILLS.md) for development and agent-usage skills.

## Installation

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/install.sh)
```

Or manually:

```sh
git clone https://github.com/TrebuchetDynamics/research-forge
cd research-forge
go build -o bin/rforge ./cmd/rforge
```

Requires Go 1.22+. Optional services (GROBID, OpenSearch, Qdrant, R/metafor) are not required for normal use.

## Quickstart

```sh
# Create a research project
rforge project create ./my-review --title "High entropy superconductors"

# Search for papers
rforge search --source openalex --query "high entropy superconductors" --from-year 2020

# Import search results into the project library
rforge search import --source openalex --query "high entropy superconductors" \
  --pages 3 --project ./my-review

# Check for duplicates
rforge duplicate report --project ./my-review

# Build a report
rforge report build --out ./my-review/report.md

# Launch the local research cockpit
rforge --project ./my-review ui
```

OSS repository study:

```sh
rforge oss add kermitt2/grobid
rforge oss scan --topic "meta-analysis"
rforge oss report --area parsers
```

## Goals

- Cross-source academic search (OpenAlex, Crossref, arXiv, Semantic Scholar, PubMed, Europe PMC, NASA ADS, DOAJ, CORE)
- Legal open-access full-text discovery and acquisition
- PDF and structured-document parsing (GROBID, S2ORC, PaperMage, Anystyle)
- Citation graph expansion and domain mapping
- Deduplication and systematic-review screening workflows
- Evidence extraction linked to exact source passages
- Meta-analysis and statistical reporting (R/metafor, PyMARE)
- Reproducible research packages with full audit trails
- CLI and local Go + HTMX web GUI

## Target users

- Academic researchers and graduate students
- Meta-analysis and systematic-review authors
- Independent scientists and research engineers
- R&D teams in physics, engineering, computer science, mathematics, and materials science

## Architecture

```text
Research question / domain query
  -> Query planner + protocol compiler
  -> rforge CLI
  -> Scholarly source connectors (OpenAlex, arXiv, Crossref, ...)
  -> Local SQLite project store + provenance log
  -> Document acquisition + parser adapters (GROBID, S2ORC, ...)
  -> Retrieval index (SQLite FTS / OpenSearch / Qdrant)
  -> Screening engine + active-learning scaffold
  -> Evidence extraction + risk-of-bias
  -> Meta-analysis / statistical engine (R/metafor, PyMARE)
  -> Report generator + reproducible package exporter
  -> Local Go + HTMX research cockpit (rforge ui)
```

## Stack

| Layer | Choice |
|---|---|
| Language | Go |
| CLI | `rforge` |
| Local web GUI | Go + HTMX (`rforge ui`) |
| Database | SQLite (PostgreSQL adapter seam planned) |
| Search | SQLite FTS; OpenSearch and Qdrant as optional adapters |
| PDF parsing | GROBID (optional); S2ORC, PaperMage, Anystyle adapter seams |
| Metadata sources | OpenAlex, Crossref, arXiv, Semantic Scholar, PubMed, Europe PMC, NASA ADS, Unpaywall, DOAJ, CORE |
| Meta-analysis | R `metafor`; PyMARE adapter seam |

## Design principle

ResearchForge is a scientific workflow engine, not an opaque AI answer machine. Every output should be able to answer:

- What did we search, and where?
- Where did each paper come from?
- Why was each paper included or excluded?
- What exact source supports each extracted claim?
- What statistical model was run, with which settings?
- Can another researcher reproduce the result from the package?

## Agent usage

An [agent skill](./skills/research-forge/SKILL.md) is included for LLM agents running `rforge` to conduct research and save outputs to a project folder or arbitrary path.

## License

MIT License (SPDX: `MIT`), Copyright (c) 2026 Trebuchet Dynamics. See [LICENSE](LICENSE). The license was selected by the repository owner on 2026-06-13; the decision record is tracked in issue #1 and [docs/owner-decisions.md](docs/owner-decisions.md).

## Decision-gated scope

Local web GUI delivery targets Go + HTMX; the implementation tracker is recorded in issue #2 and ADR 0006. Run `make todo-audit` to verify that remaining unchecked `TODO.md` items are covered by owner decisions, `make todo-completion-audit` for the closeout prompt-to-artifact checklist, or `make decisions-markdown` for a review-friendly blocker table.
