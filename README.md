# ResearchForge

[![CI](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml)
[![Playwright e2e](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/research-forge.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/research-forge)

LLMs can summarize a hundred papers in seconds. What they cannot give you is an auditable systematic review: logged searches, traceable sources, recorded inclusion decisions, evidence tied to exact passages, and a replayable workflow another researcher can verify.

**ResearchForge** builds the package behind that review. Searches are logged. Sources are tracked. Inclusion decisions are stored. Evidence links to exact passages. The finished package replays offline.

The command-line tool is `rforge`.

## What it does

- Search thirty-five scholarly sources: OpenAlex, arXiv, Crossref, Semantic Scholar, PubMed, Europe PMC, NASA ADS, DOAJ, CORE, bioRxiv/medRxiv, Zenodo, INSPIRE HEP, dblp, ClinicalTrials.gov, OSF, OpenCitations, BASE, zbMATH Open, figshare, DataCite, Lens.org, ERIC, HAL, Dimensions, PubChem, ChemRxiv, NTRS, DOAB, OpenAIRE, PLOS, OSTI, Dryad, Research Square, CiNii, BioStudies
- Import and deduplicate papers into a local project store
- Track provenance end to end; every reference knows where it came from
- Screen studies with recorded inclusion and exclusion decisions
- Extract evidence linked to exact source passages
- Run meta-analysis and statistical reporting
- Build auditable reports
- Export replayable review packages that another researcher can audit offline

## Status

**Pre-alpha.** The `rforge` CLI has 723 passing tests across 30 packages. The project format, CLI surface, and APIs may change before 1.0.

**Works today:**
- Project workspaces
- Multi-source search and import
- Source and reference provenance logs
- Deduplication reporting
- Offline forge fixture workflow (`rforge forge`)
- Package audit and replay
- Local UI skeleton (`rforge ui`)

**Experimental:**
- Document parsing
- Screening engine
- Evidence extraction
- Meta-analysis and statistical reporting
- Web GUI beyond the skeleton

**Adapter seams planned:** GROBID (PDF parsing), OpenSearch, Qdrant (search), R/metafor, PyMARE (meta-analysis), PostgreSQL

See [ROADMAP.md](./ROADMAP.md) for milestone sequencing.

## Installation

```sh
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
```

Requires Go 1.22+. Run `rforge --version` to verify.

To build from source:

```sh
git clone https://github.com/TrebuchetDynamics/research-forge
cd research-forge
go build -o bin/rforge ./cmd/rforge
```

## Quickstart

### Search, import, and report

```sh
rforge project create ./my-review --title "High entropy superconductors"
rforge search import --source openalex --query "high entropy superconductors" \
  --pages 3 --project ./my-review
rforge duplicate report --project ./my-review
rforge report build --out ./my-review/report.md
rforge --project ./my-review ui
```

### Reproducible review package workflow

`rforge forge` is the guided workflow for building an auditable review package — a self-contained artifact that can be verified and replayed offline.

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

### OSS repository study

```sh
rforge oss add kermitt2/grobid
rforge oss scan --topic "meta-analysis"
rforge oss report --area parsers
```

## Example package

A review package is a directory:

```
review.rforgepkg
├── protocol.yaml
├── searches/
├── sources/
├── screening/
├── evidence/
├── analysis/
├── report.md
└── provenance.jsonl
```

`rforge package audit` verifies the package is complete. `rforge package replay` re-runs the workflow from the provenance log.

## Local UI

```sh
rforge --project ./my-review ui
```

Opens a local Go + HTMX web interface for browsing references, provenance, screening decisions, and evidence.

## LLM and agent usage

ResearchForge does not replace scientific judgment with AI answers. It gives researchers and agents structured tools for searching, screening, extracting, and reporting, while keeping citations, source passages, and replayable logs as the record of truth.

LLM outputs that enter the workflow are stored with provenance like any other step. The system is model-agnostic and works without any LLM connection.

ResearchForge ships a standalone agent skill — [`skills/research-forge/SKILL.md`](./skills/research-forge/SKILL.md) — that works in any project, not just this repo. The skill auto-installs `rforge` if it is not on the system, then handles literature search, provenance, and review packaging for any academic topic.

### Install the skill

**Claude Code / Pi — install globally (one command):**

```sh
mkdir -p ~/.claude/skills/research-forge && \
  curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/skills/research-forge/SKILL.md \
  > ~/.claude/skills/research-forge/SKILL.md
```

**Any other harness** — paste the contents of [`skills/research-forge/SKILL.md`](./skills/research-forge/SKILL.md) as your system prompt or opening message.

### Use the skill

Once installed, invoke it from any project:

```
Use the research-forge skill to research: <your topic>
```

The skill will check for `rforge`, install it if missing, create a project workspace, search the relevant sources, and write `provenance.json` before finishing.

## Architecture

```
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

| Layer | Choice |
|---|---|
| Language | Go |
| CLI | `rforge` |
| Local web GUI | Go + HTMX (`rforge ui`) |
| Database | SQLite (PostgreSQL adapter seam planned) |
| Search | SQLite FTS; OpenSearch and Qdrant as optional adapters |
| PDF parsing | GROBID (optional); S2ORC, PaperMage, Anystyle adapter seams |
| Metadata sources | OpenAlex, Crossref, arXiv, Semantic Scholar, PubMed, Europe PMC, NASA ADS, Unpaywall, DOAJ, CORE, bioRxiv/medRxiv, Zenodo, INSPIRE HEP, dblp, ClinicalTrials.gov, OSF, OpenCitations, BASE, zbMATH Open, figshare, DataCite, Lens.org, ERIC, HAL, Dimensions, PubChem, ChemRxiv, NTRS, DOAB, OpenAIRE, PLOS, OSTI, Dryad, Research Square, CiNii, BioStudies |
| Meta-analysis | R `metafor`; PyMARE adapter seam |

## Development

See [SKILLS.md](./SKILLS.md) before starting implementation work — each development skill owns a specific slice and enforces red-green-refactor TDD. See [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md) for full product requirements and [docs/reproducible-review-package.md](./docs/reproducible-review-package.md) for the package contract.

## License

MIT License (SPDX: `MIT`), Copyright (c) 2026 Trebuchet Dynamics. See [LICENSE](LICENSE). The license was selected by the repository owner on 2026-06-13; the decision record is tracked in issue #1 and [docs/owner-decisions.md](docs/owner-decisions.md).

## Decision-gated scope

Local web GUI delivery targets Go + HTMX; the implementation tracker is recorded in issue #2 and ADR 0006. Run `make todo-audit` to verify that remaining unchecked `TODO.md` items are covered by owner decisions, `make todo-completion-audit` for the closeout prompt-to-artifact checklist, or `make decisions-markdown` for a review-friendly blocker table.
