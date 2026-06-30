# ResearchForge

[![CI](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml)
[![Playwright e2e](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml/badge.svg)](https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/research-forge.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/research-forge)

Search academic literature, download open-access PDFs, and generate a citable bibliography — in three commands. For teams that need a traceable, replayable systematic review, ResearchForge also builds a full auditable package.

The command-line tool is `rforge`.

## Quickstart

```sh
# 1. Search papers on a topic (saves results.jsonl + provenance)
rforge search batch --out ./research/my-topic \
  --query "prediction markets information aggregation" \
  --sources openalex,arxiv

# 2. Download open-access PDFs
rforge oa fetch --dir ./research/my-topic

# 3. Generate CITATIONS.md for every downloaded paper
rforge citations build --research-dir ./research
```

That's it. `./research/my-topic/pdfs/` holds the PDFs; `./research/CITATIONS.md` holds numbered references [1]–[N] sorted by first author.

### Multiple queries, one topic

Put queries one per line in a file:

```sh
# queries.txt
prediction markets information aggregation
LMSR logarithmic market scoring rule
binary prediction market trading strategy
```

```sh
rforge search batch --out ./research/my-topic \
  --queries queries.txt --sources openalex,semantic-scholar,arxiv
rforge oa fetch --dir ./research/my-topic
rforge citations build --research-dir ./research
```

### Multiple topics, cross-topic analysis

```sh
rforge search batch --out ./research/topic-a --query "topic A" --sources openalex,arxiv
rforge search batch --out ./research/topic-b --query "topic B" --sources openalex,arxiv
rforge oa fetch --dir ./research/topic-a
rforge oa fetch --dir ./research/topic-b

# Which papers appear in both topics?
rforge meta overlap --research-dir ./research --min-topics 2

# Citations for everything downloaded across all topics
rforge citations build --research-dir ./research
```

## Sources

`--sources` accepts a preset or a comma-separated list:

| Preset | Covers |
|---|---|
| `openalex,arxiv` | Fast, broad coverage (good default) |
| `openalex,arxiv,semantic-scholar` | Adds citation graph + AI/CS depth |
| `scholarly-fast` | OpenAlex + arXiv + Crossref |
| `all` | All 44 sources (slow) |
| `biomedical` | PubMed, Europe PMC, bioRxiv |
| `preprints` | arXiv, bioRxiv, medRxiv, ChemRxiv |
| `open` | Open-access sources only |

Single sources: `openalex`, `arxiv`, `crossref`, `semantic-scholar`, `europepmc`, `pubmed`, and 38 more.

## Commands

### Core research workflow

| Command | What it does |
|---|---|
| `rforge search batch --out <dir> --query <q> --sources <s>` | Search papers, write `results.jsonl` |
| `rforge search batch ... --queries <file>` | Batch search from a query file |
| `rforge search resume --dir <dir>` | Retry any failed queries |
| `rforge oa fetch --dir <dir>` | Download open-access PDFs to `<dir>/pdfs/` |
| `rforge citations build --research-dir <dir>` | Write `CITATIONS.md` with numbered references |
| `rforge meta overlap --research-dir <dir>` | Find papers appearing across multiple topics |

### Citation graph

| Command | What it does |
|---|---|
| `rforge citations expand --source semantic-scholar --paper <id> --direction both --depth 2 --out graph.json` | Build a citation network around a paper |
| `rforge citations report --graph graph.json --out report.md` | Summarize the citation graph |

### Utilities

| Command | What it does |
|---|---|
| `rforge search stats --dir <dir>` | Show hit counts and failure summary |
| `rforge oa lookup <doi>` | Check open-access status of a DOI |
| `rforge doctor` | Verify environment (pdftotext, network) |
| `rforge version` | Print version |

## Full reproducible review

For auditable systematic reviews with logged decisions and replayable packages:

```sh
rforge project create ./my-review --title "High entropy superconductors"
rforge forge init --project ./my-review \
  --question "Do artificial photosynthesis catalysts improve solar fuel generation?"
rforge forge status --project ./my-review
rforge forge next  --project ./my-review  # guided step-by-step workflow
```

The `forge` workflow walks you through approval gates, screening, evidence extraction, meta-analysis, and package export. The result is a `*.rforgepkg` any researcher can audit offline:

```sh
rforge package audit  ./review.rforgepkg
rforge package replay ./review.rforgepkg
```

See [docs/reproducible-review-package.md](./docs/reproducible-review-package.md) for the full workflow.

## Installation

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/install.sh | bash
```

No Go required. Run `rforge version` to verify. To build from source (Go 1.26+):

```sh
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
```

## LLM / agent usage

ResearchForge ships an agent skill — [`skills/research-forge/SKILL.md`](./skills/research-forge/SKILL.md) — that works in any project. Install it once:

```sh
mkdir -p ~/.claude/skills/research-forge && \
  curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/skills/research-forge/SKILL.md \
  > ~/.claude/skills/research-forge/SKILL.md
```

Then invoke from any Claude Code session:

```
Use the research-forge skill to research: <your topic>
```

The skill installs `rforge` if missing, runs the batch search, fetches PDFs, and writes `provenance.json` before finishing.

## Architecture

```
Research question
  -> rforge search batch      (44 scholarly sources)
  -> rforge oa fetch          (open-access PDF download)
  -> rforge citations build   (CITATIONS.md, numbered bibliography)
  -> rforge meta overlap      (cross-topic synthesis)
  -> rforge forge             (full auditable review package)
```

| Layer | Choice |
|---|---|
| Language | Go |
| CLI | `rforge` |
| Local web GUI | Go + HTMX (`rforge ui`) |
| Database | SQLite |
| PDF parsing | `pdftotext`; GROBID adapter seam |
| Metadata sources | OpenAlex, arXiv, Crossref, Semantic Scholar, PubMed, Europe PMC, and 38 more |
| Meta-analysis | arm-pair effect sizes and scientific benchmarking (`--effect raw-continuous`) |

## Development

See [SKILLS.md](./SKILLS.md) and [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md). All new features require TDD (red-green-refactor).

## Decision-gated scope

Local web GUI delivery targets Go + HTMX (ADR 0006; tracked in issue #2 and [docs/web-gui-plan.md](docs/web-gui-plan.md)). License was selected by the repository owner on 2026-06-13 (tracked in issue #1 and [docs/owner-decisions.md](docs/owner-decisions.md)).

Run `make todo-audit` to verify unchecked `TODO.md` items are covered by owner decisions, `make todo-completion-audit` for the closeout checklist, or `make decisions-markdown` for a blocker table.

## License

MIT License (SPDX: `MIT`), Copyright (c) 2026 Trebuchet Dynamics. See [LICENSE](LICENSE).
