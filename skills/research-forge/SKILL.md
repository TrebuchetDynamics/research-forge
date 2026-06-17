---
name: research-forge
description: Run ResearchForge (rforge) to conduct research and save outputs to a project folder or arbitrary path. Covers literature search, OSS study, meta-analysis, and general knowledge capture. Pragmatic: use rforge where available, fall back to raw output when not; always record provenance; surface irreversible gates to the human invoker.
---

# ResearchForge Research Agent

Use this skill to conduct research with the `rforge` CLI and save outputs to a ResearchForge project folder or any arbitrary directory.

Core principle: **retrieval-first, provenance-first, LLM-assisted** — retrieve and cite source material, preserve provenance at every step, and surface any LLM-generated suggestions to the human invoker rather than self-accepting them.

## Quick-start routing

| Goal | Entry phase |
|---|---|
| Find papers on a topic | Phase 2 — Discover |
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

```sh
rforge search --source openalex|arxiv|crossref|semantic-scholar|europepmc|pubmed \
  --query "<query>" \
  [--from-year YYYY] [--to-year YYYY] \
  [--preset systematic-review|open-access-review|recent-domain-map] \
  [--open-access true|false]

rforge search import --source openalex --query "<query>" --pages N \
  [--project <path>]

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

### OSS study

```sh
rforge oss add <owner/repo>
rforge oss scan --topic "<topic>"
rforge oss report --area <area>
rforge oss inventory-check <manifest.json>
```

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

### Full-text acquisition — irreversible gate: surface to human

Review candidates first; do not download without explicit human approval:

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

Same pattern for privacy review:

```sh
rforge --project <path> oa privacy-review
# Stop and surface before running:
# rforge --project <path> oa privacy-approve --reviewer <name> --reason "<text>"
```

---

## Phase 4 — Analyze

### Screening

```sh
rforge screen configure
rforge screen queue --out <queue.csv>
rforge screen decide
rforge screen progress
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

Every run must write or append `provenance.json` before finishing:

```json
{
  "question": "<research question or task description>",
  "sources": ["openalex", "arxiv"],
  "queries": ["<exact query string used>"],
  "timestamp": "<ISO 8601>",
  "rforge_version": "<from rforge --version, or 'not available'>",
  "outputs": ["<relative paths to all saved files>"]
}
```

If `rforge` is not available, write this file manually. A run without `provenance.json` is incomplete.

---

## Red lines

- Do not self-approve acquisition, privacy, or LLM-suggestion gates — surface them to the human invoker and stop.
- Do not hit live scholarly APIs inside any automated test or CI context.
- Do not finish a run without writing `provenance.json`.
- Do not accept LLM-generated suggestions without surfacing the queue to the human invoker first.
- Do not package or distribute reports before `oa privacy-review` is human-approved.
- Do not store copyrighted full text without an acquisition-approved record.
