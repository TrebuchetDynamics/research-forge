---
name: research-forge
description: Research academic or OSS topics using rforge with provenance, source coverage, human gates, and reproducible outputs. Use for literature search, OSS study, systematic review, evidence extraction, meta-analysis, or review packages.
---

# rforge — Academic Research Skill

Use this skill to conduct academic research with the `rforge` CLI. Works in any project — rforge is a standalone tool, not a dependency of your codebase.

Core principle: **retrieval-first, provenance-first, statistics-first, LLM-assisted**. Retrieve source metadata first, preserve exact query/source provenance, use auditable statistics where applicable, and never self-approve human review gates.

## 3-command quickstart (no project setup needed)

```sh
# 1. Search — writes results.jsonl, raw/, manifest.json to <dir>
rforge search batch --out research/my-topic --query "<topic>" --sources scholarly-fast

# 2. Fetch open-access PDFs for included papers (run AFTER screening)
rforge oa fetch --dir research/my-topic

# 3. Generate numbered bibliography from all topic subdirs
rforge citations build --research-dir research
```

**Source presets:** `scholarly-fast` (comprehensive default) | `all` (slow scout; use on a small query subset) | `biomedical` | `preprints` | `openalex,arxiv` (quick only)

Cross-topic analysis: `rforge meta overlap --research-dir research [--min-topics 2]`

For machine-readable command catalog: `rforge help --json`

---

Before running searches, choose an output location and use Comprehensive depth unless the user explicitly asks quick/standard:
- If inside a repo with `artifacts/`, use `artifacts/research/<topic-slug>/`.
- If inside a repo without `artifacts/`, use `research/<topic-slug>/`.
- If no clear project home exists, use `~/research/<topic-slug>/`.
- Quick only when explicitly requested: 3 query variants × 2–3 sources. Standard only when explicitly requested: 5–8 query variants × scholarly-fast sources plus citation expansion. Comprehensive default: 20–30 query variants × scholarly-fast or domain-relevant sources, optional `all` scout on 3–5 queries, citation expansion, evidence grid, and explicit gaps.

## Step 0 — Check and install rforge

```sh
rforge version
```

If the command is not found, install it:

```sh
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
```

If Go is not installed, tell the user:

> rforge requires Go 1.22+. Install from https://go.dev/dl/ then run the above command.

---

## Quick-start routing

| Goal | Entry point |
|---|---|
| Find papers on a topic | 3-command quickstart above, then Phase 2 — Discover for more depth |
| Screen papers before downloading PDFs | Phase 4 — Analyze (screening before oa fetch) |
| Study an open-source project | Phase 2 — Discover (OSS) |
| Full systematic review | Phase 1 → 2 → 3 → 4 → 5 |
| Save research notes or artifacts to a folder | Phase 5 — Save |

---

## Phase 1 — Setup

### Option A: ResearchForge project folder (preferred)

```sh
rforge project create <path> --title "<title>"
rforge project inspect <path>
```

Creates `<path>/manifest.json`, `<path>/data/`, and `<path>/research-forge.lock.json`. All subsequent commands use `--project <path>`.

For a guided end-to-end workflow:

```sh
rforge forge init --project <path> --question "<research question>" \
  [--sources openalex,semantic-scholar --tools grobid,qdrant]
rforge forge status --project <path>
rforge forge next --project <path>
```

### Option B: Arbitrary folder

Use any directory. Save all outputs there with consistent naming:

```
<folder>/search-results-<timestamp>.json
<folder>/papers.json
<folder>/provenance.json    ← always required (see Provenance rules)
<folder>/report.md
```

When `rforge` is not available or not initialized, write outputs directly to these files and populate `provenance.json` manually before finishing.

---

## Phase 2 — Discover

### Academic literature search

Start by expanding the question into query variants: canonical term, abbreviations, mechanism/material/method variants, application variants, broader/narrower forms, and recent-year filters when useful. Prefer `search batch` for multi-source sweeps so failures, dedupe, manifests, and stats are saved together.

```sh
# Standalone batch search — no project required
rforge search batch --out <dir> --query "<query>" --sources scholarly-fast \
  [--queries <file>] [--limit N] [--continue-on-error] [--stats]

# With a queries file (one query per line)
rforge search batch --out <dir> --queries queries.txt --sources scholarly-fast --stats

# Retry failed queries
rforge search resume --dir <dir>

# Show hit counts and library record count for a search dir
rforge search stats --dir <dir>

# Single-source search (prints results, does not save)
rforge search --source openalex|arxiv|crossref|semantic-scholar|europepmc|pubmed \
  --query "<query>" [--from-year YYYY] [--to-year YYYY] [--open-access true|false]
```

If using individual `rforge search` commands instead of `search batch`, always run coverage stats before reporting:

```sh
rforge search stats --dir <dir>
```

Pick 3–5 high-signal seed papers from the sweep for citation expansion: surveys, high-citation papers, Nature/Science/ACM/IEEE venues, or method-defining preprints.

```sh
rforge citations expand --source semantic-scholar --paper <id> \
  --direction references|citations|both --depth N --out <graph.json>

rforge search related --source openalex --paper <work-id>
```

Draft a source-specific query plan (requires human review before use):

```sh
rforge protocol compile --type pico|peco|spider|freeform --question "<text>"
rforge protocol plan-sources --type pico --question "<text>"
rforge protocol capabilities
```

### OSS study and open-source project discovery

First generate a provider-coverage search plan. This is safe: it does not clone repos, add dependencies, or approve integration.

```sh
rforge oss search-plan --query "<project/functionality to find>" \
  [--ecosystem all|go|python|js|rust|data]

rforge --json oss search-plan --query "<project/functionality to find>" --ecosystem all
```

Use the plan to search multiple open-source ecosystems rather than GitHub alone:

- Code forges: GitHub, GitLab, Codeberg/Forgejo, SourceHut.
- Archives: Software Heritage.
- Package registries: pkg.go.dev, PyPI, npm, crates.io, plus ecosystem-specific registries when relevant.
- Security/supply-chain signals: OpenSSF Scorecard, release cadence, CI, security policy, dependency metadata.
- Human gates: clone approval for large repos, license review, dependency/import approval, and integration disposition.

Then record selected repositories in a ResearchForge project:

```sh
rforge --project <path> oss add <owner/repo> [--area <area>]
rforge --project <path> oss scan <owner/repo> --topic "<topic>"
rforge --project <path> oss report --area <area>
rforge oss inventory-check <manifest.json>
rforge oss inventory-refresh <manifest.json> --source github
rforge oss inventory-policy <manifest.json> [--stale-after 18mo]
rforge oss inventory-roadmap <manifest.json> --todo TODO.md
```

Do not treat stars/downloads as quality proof. Use them only as one signal alongside maintenance, license, activity, security posture, package metadata, and domain fit.

### Fallback when rforge is not available

Save raw API responses to `<folder>/search-results-<timestamp>.json` and record the query, source, and timestamp in `provenance.json`.

---

## Phase 3 — Collect

```sh
rforge import bibtex|ris|csl-json|zotero-rdf|json|csv <file>
rforge library list
rforge library refresh-doi <doi>
rforge duplicate report [--source <source>]
rforge duplicate merge | split
```

### Open-access PDF acquisition

**Recommended order (ADR 0008): screen on title/abstract first (Phase 4), then fetch PDFs for included papers only. This reduces fetch volume by the exclusion rate (~60–80%).**

```sh
# Standalone: download open-access PDFs for all papers in <dir>/results.jsonl
# Run this AFTER screening; PDFs go to <dir>/pdfs/
rforge oa fetch --dir <dir>

# Inspect availability without downloading
rforge oa sources
rforge oa resolve-plan <doi>
rforge oa lookup <doi>
```

Legal OA source coverage includes Unpaywall, OpenAlex OA locations, Europe PMC/PMC, arXiv, bioRxiv/medRxiv, ChemRxiv, DOAJ, CORE, Semantic Scholar/Crossref hints, Internet Archive/Open Library, and Software Heritage for software archival. Sci-Hub-like sources are intentionally unsupported.

For project-based workflows, acquisition requires human approval gates:

```sh
rforge --project <path> oa candidates
rforge --project <path> oa acquisition-queue
```

Stop and surface this to the human invoker before proceeding:

```
Acquisition requires human approval. Please run:
  rforge --project <path> oa acquisition-approve <id> \
    --reviewer <name> --reason "<text>"
Waiting — do not download any files until approval is confirmed.
```

After approval, to parse PDFs:

```sh
rforge --project <path> research acquire-pdftotext \
  --doi <doi> --pdf-url <url> --license <license> --oa-status <status> \
  --out <path>/parsed/<paper-id>.json
```

---

## Phase 4 — Analyze

### Screening (run BEFORE oa fetch to reduce download volume)

The two-stage screening pipeline follows PRISMA 2020: title/abstract screen first (on metadata already in `results.jsonl`), then `oa fetch` for included papers only, then full-text eligibility on PDFs.

**Standalone dir-based workflow (no project required):**

```sh
# 1. Export pending papers to a self-contained CSV for reviewer decisions
#    Columns filled by rforge: doi, arxiv_id, title, authors, year, abstract, source
#    Columns filled by reviewer: decision (include|exclude|uncertain) and reason
rforge screen queue --dir <topic-dir> --out queue.csv

# 2. Reviewer fills in 'decision' and 'reason' columns in queue.csv, then:
rforge screen import --dir <topic-dir> --csv queue.csv [--reviewer <name>]
#    Writes decisions to <topic-dir>/screening.jsonl (last-write-wins on re-import)

# 3. Check progress
rforge screen progress --dir <topic-dir>
rforge screen progress --dir <topic-dir> --json   # machine-readable counts
```

**After screening, fetch PDFs for included papers only:**
```sh
rforge oa fetch --dir <topic-dir>
```

**Project-based screening (requires --project):**
```sh
rforge --project <path> screen configure
rforge --project <path> screen decide --paper <id> --stage title_abstract --decision include|exclude|uncertain --reviewer <name>
rforge --project <path> screen progress
rforge prisma counts
```

### Evidence extraction

```sh
rforge extraction schema add
rforge extract add|suggest
rforge evidence grid --out <grid.json>
rforge evidence gaps --out <report.json>
```

### Document parsing

```sh
rforge parse --paper <id> --parser grobid --pdf <file>
rforge parse quality --parsed <parsed.json> --out <report.json>
rforge parse arbitrate --left <parsed.json> --right <parsed.json> --out <report.json>
```

### Meta-analysis

```sh
rforge analysis prepare [--effect smd|log-odds-ratio|risk-ratio|mean-difference]
rforge analysis run
rforge analysis sensitivity
rforge analysis publication-bias --method egger|begg
```

### LLM-generated suggestions — do not self-accept

Commands like `entity-suggest`, `citation-suggest`, and `risk-bias-suggest` produce queues with `suggested` status. Do **not** call the corresponding `-review` commands to self-accept them. Surface the queue path to the human invoker:

```
LLM suggestions written to <queue.json>.
Human review required before acceptance:
  rforge evidence entity-review --queue <queue.json> --id <id> \
    --decision accepted|rejected|corrected --reviewer <name> --out <queue.json>
```

### Report

```sh
rforge report build --out <report.md> [--parsed <parsed.json>]
rforge report trace --claims <queue.json> --analysis <run.json> --out <trace.json>
rforge report audit
```

---

## Phase 5 — Save

### Bibliography and cross-topic analysis

```sh
# Generate numbered CITATIONS.md from all topic subdirs under <research-dir>
rforge citations build --research-dir <research-dir> [--out <file>]

# Find papers appearing in multiple topic subdirs
rforge meta overlap --research-dir <research-dir> [--min-topics 2]

# Export as an Obsidian vault — one .md note per paper with wikilinks,
# per-topic index notes, and index.md with cross-topic highlights
rforge vault build --research-dir <research-dir> --out <vault-dir>
```

The Obsidian vault output:
- `papers/<slug>.md` — per-paper notes with YAML frontmatter (title, authors, year, doi, topics) and `[[wikilinks]]` to every topic the paper appears in
- `<topic>.md` — topic index notes listing all papers with `[[papers/slug|Title]]` links
- `index.md` — main dashboard with all topics, paper counts, and cross-topic papers highlighted

### ResearchForge project package

```sh
rforge package create --out <dir> --created-by <name> --question "<text>"
rforge package archive <dir> <archive.tar>
rforge package audit <dir>
```

Do not package before `oa privacy-review` has been human-approved.

### Export to arbitrary folder

```sh
rforge export json|csv|bibtex|ris|csl-json <file>
rforge report build --out <folder>/report.md
```

Or write directly without `rforge`:

```
<folder>/report.md
<folder>/papers.json
<folder>/provenance.json
<folder>/search-results-<timestamp>.json
```

---

## Provenance rules (always required)

Every run must write or append `provenance.json` before finishing. Capture depth, exact queries, sources, coverage stats, citation expansion attempts, outputs, and failures/rate limits. If `search batch --stats` was used, include `search-stats.txt` or equivalent stats output.

```json
{
  "question": "<research question or task description>",
  "depth": "quick|standard|comprehensive",
  "sources": ["openalex", "arxiv"],
  "queries": ["<exact query string used>"],
  "timestamp": "<ISO 8601>",
  "rforge_version": "<from rforge version, or 'not available'>",
  "search_stats": {"openalex": 0, "arxiv": 0, "total_unique_dois": 0},
  "citation_expand_attempted": ["<paper id or DOI>"],
  "citation_expand_succeeded": ["<paper id or DOI>"],
  "outputs": ["<relative paths to all saved files>"],
  "errors": ["<rate limit, API failure, missing source, or empty output notes>"]
}
```

If `rforge` is not available, write this file manually. A run without `provenance.json` is incomplete.

---

## Verification gate

Before declaring research complete, verify:
- `rforge version` was run or marked unavailable in provenance.
- Every planned source/query has a saved result file, batch manifest, or explicit error note.
- Source coverage stats were run and recorded.
- `report.md` or the requested output exists and cites exact papers, repositories, or artifacts.
- `provenance.json` exists, is valid JSON, and lists all outputs.
- Human-gated actions were surfaced rather than self-approved.

---

## Red lines

- Do not self-approve acquisition, privacy, or LLM-suggestion gates — surface them to the human invoker and stop.
- Do not hit live scholarly APIs inside any automated test or CI context.
- Do not finish a run without writing `provenance.json`.
- Do not accept LLM-generated suggestions without surfacing the queue to the human invoker first.
- Do not package or distribute reports before `oa privacy-review` is human-approved.
- Do not store copyrighted full text without an acquisition-approved record.
