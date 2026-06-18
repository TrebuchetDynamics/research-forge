# ResearchForge

[![CI](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml)
[![Playwright e2e](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/research-forge.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/research-forge)

ResearchForge is an open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, and meta-analysis.

The command-line tool is **`rforge`**.

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

ResearchForge helps researchers discover papers, map citation graphs, screen literature, extract evidence, run meta-analyses, generate auditable reports, visualize artifacts locally, and maintain a knowledge base of relevant open-source scholarly tooling.

## Status

**Pre-alpha.** The `rforge` CLI is functional with 626 passing tests across 29 packages. Core workflows — project workspaces, scholarly source connectors, source/import provenance, reference-manager interop, legal acquisition gates, document parsing, citation graph expansion, screening, evidence extraction, analysis, reporting, reproducible review packages, and local web GUI — are implemented and test-covered. The project format, CLI surface, and APIs may change before 1.0.

Current product direction: `rforge forge` is the guided Meta-analysis spine. The first "done" artifact is a Reproducible review package that can be audited and replayed offline.

See [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md) for full product requirements, [ROADMAP.md](./ROADMAP.md) for milestone sequencing, [docs/reproducible-review-package.md](./docs/reproducible-review-package.md) for the package contract, and [SKILLS.md](./SKILLS.md) for development and agent-usage skills.

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

Guided offline spine smoke test:

```sh
rforge forge init --project ./my-review \
  --question "Do artificial photosynthesis catalysts improve solar fuel generation outcomes?"
rforge forge approve --project ./my-review --gate "question approval" --note accepted
rforge forge approve --project ./my-review --gate "protocol approval" --note accepted
rforge forge approve --project ./my-review --gate "network/API approval" --note accepted
rforge forge source-fixture --project ./my-review
rforge forge reference-fixture --project ./my-review
rforge forge approve --project ./my-review --gate "identity approval" --note accepted
rforge forge acquisition-fixture --project ./my-review
# Continue parser/screening/evidence/analysis/report approvals, then:
rforge forge package-fixture --project ./my-review --out ./review.rforgepkg
rforge package audit ./review.rforgepkg
rforge package replay ./review.rforgepkg
```

Direct search/import workflow:

```sh
rforge project create ./my-review --title "High entropy superconductors"
rforge search --source openalex --query "high entropy superconductors" --from-year 2020
rforge search import --source openalex --query "high entropy superconductors" \
  --pages 3 --project ./my-review
rforge duplicate report --project ./my-review
rforge report build --out ./my-review/report.md
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

## Skills

ResearchForge includes repo-local Pi skills under [`skills/`](./skills/) and an index in [`SKILLS.md`](./SKILLS.md).

- [`skills/research-forge/SKILL.md`](./skills/research-forge/SKILL.md) is the agent usage skill for running `rforge` to conduct provenance-first research.
- The `research-forge-*-tdd` skills are development skills for specific slices: workflow orchestration, architecture, fixtures, scholarly ingestion, OSS intelligence, document pipeline, screening, evidence extraction, meta-analysis, reporting, web UI, governance, security, performance, release packaging, and developer docs.
- All development skills require red-green-refactor TDD unless the change is documentation-only or explicitly marked as emergency scaffolding.

Use `SKILLS.md` before starting implementation work so the right specialist skill owns the slice.

## License

MIT License (SPDX: `MIT`), Copyright (c) 2026 Trebuchet Dynamics. See [LICENSE](LICENSE). The license was selected by the repository owner on 2026-06-13; the decision record is tracked in issue #1 and [docs/owner-decisions.md](docs/owner-decisions.md).

## Decision-gated scope

Local web GUI delivery targets Go + HTMX; the implementation tracker is recorded in issue #2 and ADR 0006. Run `make todo-audit` to verify that remaining unchecked `TODO.md` items are covered by owner decisions, `make todo-completion-audit` for the closeout prompt-to-artifact checklist, or `make decisions-markdown` for a review-friendly blocker table.
