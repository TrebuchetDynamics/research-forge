<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="ResearchForge turns scholarly searches into traceable records, visible failures, citations, and replayable review packages">
</p>

<p align="center">
  <a href="https://github.com/TrebuchetDynamics/research-forge/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/TrebuchetDynamics/research-forge?label=release&color=FF6B35"></a>
  <a href="https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml"><img alt="CI status" src="https://github.com/TrebuchetDynamics/research-forge/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml"><img alt="Playwright end-to-end status" src="https://github.com/TrebuchetDynamics/research-forge/actions/workflows/playwright-e2e.yml/badge.svg"></a>
  <a href="https://pkg.go.dev/github.com/TrebuchetDynamics/research-forge"><img alt="Go package reference" src="https://pkg.go.dev/badge/github.com/TrebuchetDynamics/research-forge.svg"></a>
  <a href="./LICENSE"><img alt="MIT license" src="https://img.shields.io/badge/license-MIT-0F1115"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="#choose-your-workflow">Choose a workflow</a> ·
  <a href="#source-coverage">Sources</a> ·
  <a href="#use-it-from-an-agent">Agent skill</a> ·
  <a href="#trust-boundaries">Trust</a>
</p>

**ResearchForge** is a local-first CLI for literature discovery and reproducible reviews. It searches scholarly sources, normalizes and deduplicates records, keeps failures visible, retrieves legal open-access files, builds citations, and packages attributable work for audit or replay.

Use a **research directory** for fast literature scouting. Use a guided **forge project** when screening, extraction, analysis, and export decisions must be attributable.

## Proof, not promises

A tracked run in this repository used **6 queries across 4 sources** and produced **280 records, 237 deduplicated records, and 10 preserved failures**. Inspect the real [`manifest.json`](./research/open-source-project-search/manifest.json); these are repository artifacts, not adoption or performance claims.

Every batch search leaves an ordinary directory behind:

```text
research/my-topic/
├── manifest.json           # exact queries, sources, counts, timestamp
├── results.jsonl           # normalized records from successful sources
├── results-deduped.jsonl   # merged DOI/title identities
├── failures.jsonl          # retryable source/query failures
├── batch-state.json        # checkpoint for interrupted sweeps
├── search-stats.txt        # coverage and dedupe summary
├── raw/                    # readable source/query responses
└── pdfs/                   # approved open-access downloads
```

Nothing is hidden in a hosted dashboard. Review the JSONL, version it, diff it, retry it, or hand it to another tool.

## Quick start

### 1. Install

```sh
curl -fsSL \
  https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/install.sh | bash
```

Prebuilt binaries support Linux, macOS, and Windows. To build from source (Go 1.26+):

```sh
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
```

### 2. Search two reliable starter sources

```sh
rforge search batch --out ./research/my-topic \
  --query "prediction markets information aggregation" \
  --sources openalex,arxiv --stats
```

### 3. Build a bibliography

```sh
rforge citations build --research-dir ./research
```

You now have normalized records, a deduplicated set, a failure queue, a manifest, statistics, and `research/CITATIONS.md`. After title/abstract screening, fetch legal open-access files for included papers:

```sh
rforge oa fetch --dir ./research/my-topic
```

## How the evidence trail works

<p align="center">
  <img src="./assets/readme/workflow.svg" width="100%" alt="Five-stage ResearchForge workflow: frame a question, search 44 sources, normalize records, screen before legal open-access retrieval, then create citations or replayable packages">
</p>

ResearchForge automates **reversible work**: retrieval, normalization, deduplication, checkpointing, retry queues, provenance checks, and package audits. Humans still approve inclusion/exclusion, full-text acquisition, accepted extraction, analysis methods, final claims, and package export.

## Choose your workflow

| If you need to…                       | Start here                                               |
| ------------------------------------- | -------------------------------------------------------- |
| Scout a topic quickly                 | `rforge search batch` with `openalex,arxiv`              |
| Maintain a living search              | `search batch --resume`, then `search refresh`           |
| Run an attributable review            | `project create` + `forge init`                          |
| Audit or replay a review package      | `package audit` / `package replay`                       |
| Let an agent research with guardrails | Install the [ResearchForge skill](#use-it-from-an-agent) |

## Keep a topic reproducible

Put one query per line in `queries.txt`, then search several sources in one transaction:

```sh
rforge search batch --out ./research/market-design \
  --queries queries.txt \
  --sources openalex,arxiv,crossref,semantic-scholar \
  --continue-on-error --stats
```

### Resume an interrupted sweep

ResearchForge checkpoints each query/source pair to `batch-state.json`. After a timeout, `Ctrl-C`, or killed process, rerun the same batch with `--resume`:

```sh
rforge search batch --out ./research/market-design \
  --queries queries.txt \
  --sources openalex,arxiv,crossref,semantic-scholar \
  --continue-on-error --stats --resume

rforge search stats --dir ./research/market-design
```

### Retry, refresh, and validate deliberately

```sh
# Preview failed-query retries without using the network
rforge search resume \
  --failures ./research/market-design/failures.jsonl \
  --out ./research/market-design --dry-run

# Replay the stored manifest and report new / unchanged / gone records
rforge search refresh --dir ./research/market-design --dry-run
rforge search refresh --dir ./research/market-design

# Validate a versioned agent-authored receipt
rforge provenance validate ./research/market-design/provenance.json
```

Provenance schema v1 requires an exact depth (`quick`, `standard`, or `comprehensive`), normalized `rforge_version` data, required fields, and string-only errors.

## When a search becomes a review

Create a guided project when decisions and analysis must be replayable by another reviewer:

```sh
rforge project create ./my-review --title "High entropy superconductors"

rforge forge init --project ./my-review \
  --question "What outcomes are reported for high entropy superconductors?"

rforge forge status --project ./my-review
rforge forge next --project ./my-review
```

The state machine covers source planning, import and dedupe, legal acquisition, parser arbitration, screening, evidence extraction, analysis, reporting, privacy review, and export. Review packages include checksums, provenance, redaction records, and offline audit/replay commands:

```sh
rforge package audit  ./review.rforgepkg
rforge package replay ./review.rforgepkg
```

See the [reproducible review package specification](./docs/reproducible-review-package.md) for required files and human gates.

## Resolve and rank papers

Instead of reading search results top-to-bottom, resolve a specific paper or ask which papers to read first:

```sh
# Resolve one paper by DOI, arXiv ID, OpenAlex ID, URL, or title
rforge paper 10.7717/peerj.4375
rforge paper 2301.99999
rforge paper W123456789
rforge paper "transformer scaling laws"
rforge paper https://arxiv.org/abs/2301.99999

# Resolve several papers at once (batch)
rforge paper 10.7717/peerj.4375 10.1038/nature14539 "attention is all you need"
```

`rforge paper` merges metadata from OpenAlex, Crossref, arXiv, Semantic Scholar, and Unpaywall, so a DOI lookup returns the OpenAlex ID, arXiv ID, license, and OA status in one shot — the same resolver Feynman's `paper` command provides.

Decide what to read first with `rforge rank` — a citation-weighted, recency-aware, OA-aware score across sources:

```sh
rforge rank "mechanistic interpretability sparse autoencoders" \
  --sources openalex,crossref,semantic-scholar --limit 20 --out rank.json
```

The rank score blends log-scaled citation count, recency (1/(1+age)), and an open-access bonus, then prints a numbered list with DOIs and arXiv IDs for follow-up.

## Source coverage

`search batch` supports **44 connectors**. Start narrow; broad sweeps are slower and more exposed to upstream quotas.

| Preset                            | Good for                                                |
| --------------------------------- | ------------------------------------------------------- |
| `openalex,arxiv`                  | Fast general discovery                                  |
| `scholarly-fast`                  | OpenAlex + arXiv + Crossref                             |
| `openalex,arxiv,semantic-scholar` | Citation-oriented AI/CS research                        |
| `biomedical`                      | PubMed + Europe PMC + bioRxiv                           |
| `preprints`                       | arXiv + bioRxiv + medRxiv + ChemRxiv                    |
| `open`                            | Open-access sources                                     |
| `all`                             | Every keyword-search connector; expect partial failures |

Run `rforge doctor` before a large sweep. It reports optional configuration such as `RFORGE_SEMANTIC_SCHOLAR_API_KEY`, whose absence can cause HTTP 429 responses. For machine-readable command discovery, run `rforge help --json`.

## Use it from an agent

Install the passive **Claude Code, Codex, Pi or Hermes plugin** using the [agent installation guide](./docs/agent-install.md), including per-host upgrade and removal steps. Before migrating, delete any old curl-installed `research-forge` skill copy to avoid duplicate discovery. Plugins do not install the engine or change host approvals, permissions or schedules.

All hosts expose the same retrieval-first skill at [`skills/research-forge/SKILL.md`](./skills/research-forge/SKILL.md). It defines source breadth, provenance requirements, legal-acquisition boundaries, and human approval gates; it never authorizes an agent to approve screening, extraction, final claims, or package export.

### Quick commands: resolve, rank, brief, compare

```sh
# Resolve one paper by DOI, arXiv ID, OpenAlex ID, or title (merges 5 sources)
rforge paper 10.7717/peerj.4375
rforge paper --file papers.txt --out resolved.json

# Decide what to read first: citation-weighted, recency-aware rank
rforge rank "mechanistic interpretability" --sources openalex,crossref,semantic-scholar --limit 20

# Boost rank scores by citation graph prestige (from `citations expand`)
rforge citations expand --source openalex --paper W123456789 --direction both --out graph.json
rforge rank "sparse autoencoders" --graph graph.json

# Generate a one-page research brief from a search dir
rforge brief research/my-topic --out brief.md

# Compare two or more search dirs for overlap and gaps
rforge compare research/topic-a research/topic-b --labels "A,B" --out comparison.md

# Cross-topic literature synthesis (Feynman `lit` analog): consensus themes,
# cross-topic papers, open questions, screening status
rforge lit --research-dir research --out <dir>

# Per-paper annotations (Feynman `alpha annotate` analog), persisted in project
rforge note add --project <path> --paper <doi> --note "Key result"
rforge note list --project <path>

# Recall across past sessions: search every topic dir (Feynman session-search)
rforge history search "PRISMA 2020" --research-dir research --limit 5

# Generate a deterministic replication plan from a resolved paper
rforge replicate 10.1136/bmj.n160 --out replan.md
```

**Generic fallback — DeepSeek Harness and other shared-agent consumers without native plugins:**

```sh
mkdir -p ~/.agents/skills/research-forge
curl -fsSL \
  https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/skills/research-forge/SKILL.md \
  > ~/.agents/skills/research-forge/SKILL.md
```

Use your host's documented skill directory if it does not read `~/.agents/skills`. Upgrade the fallback by repeating the download; remove it by deleting only that skill copy. Do not keep it alongside the native plugin in the same host.

Then ask: **“Use the research-forge skill to research: `<your question>`.”** The skill calls the standalone `rforge` CLI; no product fork is required.

## Trust boundaries

- **Open access is explicit.** `oa fetch` distinguishes `oa_unavailable` from download failures; paywall-bypass sources are unsupported.
- **Failures stay visible.** Rate limits, timeouts, and source/query failures remain in `failures.jsonl` for deliberate retry.
- **Research claims remain human-owned.** Retrieval and checks can be automated; scientific approval cannot.
- **Shareable packages are reviewed.** Secrets, private paths, and restricted full text are blocked or redacted before export.
- **Local-first means inspectable.** Core workflows use the CLI and machine-readable files; the Go + HTMX web UI is optional.

## Documentation and development

- [Reproducible review package specification](./docs/reproducible-review-package.md)
- [Product requirements](./RESEARCH-FORGE-PRD.md)
- [Roadmap](./ROADMAP.md)
- [Skill catalog](./SKILLS.md)
- [Contributing guide](./CONTRIBUTING.md)
- [Release downloads](https://github.com/TrebuchetDynamics/research-forge/releases)

```sh
git clone https://github.com/TrebuchetDynamics/research-forge.git
cd research-forge
make check
```

ResearchForge uses Go, SQLite, a Go + HTMX local UI, deterministic fixtures, and test-first development.

## Decision-gated scope

The local Go + HTMX UI is tracked in issue #2 and ADR 0006. The MIT License decision is tracked in issue #1 and [docs/owner-decisions.md](docs/owner-decisions.md).

Run `make todo-audit` to verify owner decisions, `make todo-completion-audit` for the closeout checklist, or `make decisions-markdown` for a blocker table.

## License

[MIT License](./LICENSE) (SPDX: `MIT`) © 2026 Trebuchet Dynamics.
