---
name: research-forge
description: Research academic or OSS topics using rforge with provenance, source coverage, human gates, and reproducible outputs. Use for literature search, OSS study, systematic review, evidence extraction, meta-analysis, or review packages.
metadata:
  min_rforge_version: "0.1.22"
  required_capabilities: "search=1,paper=1,brief=1,compare=1,provenance=1"
---

# rforge — Academic Research Skill

Use this skill to conduct academic research with `rforge` through CLI or MCP. Works in any project — rforge is a standalone tool, not a dependency of your codebase.

Core principle: **retrieval-first, provenance-first, statistics-first, LLM-assisted**. Retrieve source metadata first, preserve exact query/source provenance, use auditable statistics where applicable, and never self-approve human review gates.

## 3-command quickstart (no project setup needed)

```sh
# 1. Search — writes results.jsonl, raw/, manifest.json to <dir>
rforge search batch --out research/my-topic --query "<topic>" --sources scholarly-fast --depth standard

# 2. Fetch open-access PDFs for included papers (run AFTER screening)
rforge oa fetch --dir research/my-topic

# 3. Generate numbered bibliography from all topic subdirs
rforge citations build --research-dir research
```

**Source presets:** `scholarly-fast` (Standard starting preset) | `all` (slow scout; use on a small query subset) | `biomedical` | `preprints` | `openalex,arxiv` (quick only)

Cross-topic analysis: `rforge meta overlap --research-dir research [--min-topics 2]`

Resolve and rank before deep reading:

```sh
# Resolve one paper by DOI, arXiv ID, OpenAlex ID, or title (merges OpenAlex +
# Crossref + arXiv + Semantic Scholar + Unpaywall metadata in one shot)
rforge paper 10.7717/peerj.4375
rforge paper 2301.99999 "attention is all you need"

# Decide what to read first: citation-weighted, recency-aware, OA-aware score
rforge rank "mechanistic interpretability sparse autoencoders" \
  --sources openalex,crossref,semantic-scholar --limit 20 --out rank.json

# Boost rank with citation-graph prestige (run `citations expand` first)
rforge citations expand --source openalex --paper <work-id> --direction both --out graph.json
rforge rank "sparse autoencoders" --graph graph.json

# Synthesize a one-page brief from a completed search dir
rforge brief research/my-topic --out brief.md

# Compare two search dirs: overlap table + shared vs unique gaps
rforge compare research/topic-a research/topic-b --labels A,B --out comparison.md

# When multiple topics are done, synthesize the whole corpus (Feynman `lit` analog):
# consensus themes, cross-topic papers, open questions, screening status
rforge lit --research-dir research --out research/synthesis

# Keep per-paper notes in a project for later recall (Feynman `alpha annotate` analog)
rforge note add --project <path> --paper <doi> --note "Key result"
rforge note list --project <path>
```

For machine-readable command catalog: `rforge help --json`

---

Before running searches, choose an output location and use **Standard depth by default**:

- If inside a repo with `artifacts/`, use `artifacts/research/<topic-slug>/`.
- If inside a repo without `artifacts/`, use `research/<topic-slug>/`.
- If no clear project home exists, use `~/research/<topic-slug>/`.

### Search depth and stopping

| Depth | When to use | Query plan |
| --- | --- | --- |
| Standard (`standard`) | Default | 5–8 query variants × `scholarly-fast` sources, plus citation expansion |
| Quick (`quick`) | User explicitly requests a quick search | Up to 3 query variants × 2 sources |
| Comprehensive (`comprehensive`) | Standard exhausts its budget without saturation, or the user asks | 20–30 query variants × `scholarly-fast` or domain-relevant sources, citation expansion, evidence grid, and explicit gaps |

Query plans are starting points, not completion quotas. `search batch` enforces query, source, and request caps; it does not generate variants or automatically escalate. The Standard caps are 10 query variants, 5 sources, and 50 requests. Use `--depth standard` explicitly for clarity; pass the selected depth for Quick or Comprehensive. See [search budgets and stop reasons](https://github.com/TrebuchetDynamics/research-forge/blob/main/docs/cli.md#search-budgets-and-stop-reasons) for all caps, thresholds, and resume semantics.

After each batch, inspect `stop_reason`, `coverage`, `budget`, and `requests_used` in `manifest.json`, `batch-state.json`, or `--json` output. Coverage measures marginal yield: new unique deduped records per query, not relevance or proof of exhaustive coverage. Only fully successful queries count toward saturation.

| `stop_reason` | Meaning and next action |
| --- | --- |
| `completed` | Selected sweep was attempted; inspect failures and coverage before reporting. Completion alone does not prove saturation or justify escalation. |
| `budget_exhausted` | A query, source, or request cap was reached. For a Standard run without saturation, escalate to Comprehensive and record the coverage gap and reason. |
| `coverage_saturated` | Consecutive low-yield queries met the configured threshold. Stop expanding; report remaining gaps rather than escalating just to fill a quota. |
| `interrupted` | Execution stopped before completion. Resume the checkpoint; do not interpret interruption as thin coverage. |
| `error_threshold` | Failures stopped the sweep (the configured consecutive-error threshold under `--continue-on-error`; first error otherwise). Address failures before retrying; errors are not evidence of saturation. |

For the escalation decision, compare `coverage.consecutive_low_yield` with `budget.saturation_queries`; do not infer saturation from total hit counts. A user request for Comprehensive is an independent reason to select it. When escalating a checkpointed batch, extend the query/source plan as needed and rerun with `--resume --depth comprehensive`. Resume retains consumed requests; `--max-requests N` sets a new total allowance, not N additional requests. An optional `all` scout on 3–5 queries is still subject to the selected source cap: record which sources were actually attempted.

Every research report must state the selected depth, effective caps, requests used, stop reason, coverage signal, and remaining gaps. Preserve the engine artifacts in provenance outputs; never relabel a capped or interrupted run as exhaustive.

## Step 0 — Check rforge compatibility

```sh
rforge version
rforge capabilities --json --min-rforge-version 0.1.22 \
  --require search=1 --require paper=1 --require brief=1 \
  --require compare=1 --require provenance=1
```

The frontmatter's `min_rforge_version` and `required_capabilities` define this
compatibility check. Release builds must meet both the minimum version and all
required capability contracts. A `dev` build passes only when those capabilities
are verified; its version label alone is not sufficient.

If rforge is missing, too old, or lacks a required capability, stop engine work
and ask the user to install or upgrade it using
[install.sh](https://github.com/TrebuchetDynamics/research-forge/blob/main/install.sh)
or the [installation guide](https://github.com/TrebuchetDynamics/research-forge/blob/main/docs/agent-install.md).
Plugins never download or install the binary. Source builds require Go 1.26+:

```sh
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
```

If the user declines installation, only use the explicitly labelled manual
fallback below; never claim engine-verified provenance for that work.

---

## Choose MCP or CLI

MCP and CLI are equivalent interfaces to the same engine services, persistence,
search budgets, provenance receipts, and human gates — not different research
methods. Use registered typed MCP tools for supported operations; use CLI for
shell scripts, hosts without MCP, and commands outside the MCP allowlist. The
CLI examples below also describe the corresponding MCP workflow.

| MCP tool | CLI equivalent |
| --- | --- |
| `search` | `rforge search batch --json` |
| `job_status` | `rforge jobs status <id> --dir <dir> --json` |
| `source_get` | `rforge paper <id> --json` |
| `research_status` | `rforge brief <dir> --json` |
| `compare` | `rforge compare <dirs...> --json` |
| `evidence_retrieve` | `rforge --project <path> retrieve --query <query> --json` |

For standalone registration, check `rforge capabilities --json --require mcp=1`
and configure the client to launch `rforge` with args `["mcp", "serve"]` over
stdio. No plugin is required. See [MCP client setup and tool arguments](https://github.com/TrebuchetDynamics/research-forge/blob/main/docs/cli.md#mcp-stdio-server).
MCP is optional; lack of MCP support does not prevent CLI use after Step 0 passes.

`search` returns `data.job_id` immediately, not completed results. Pass the same
`dir` and ID to `job_status` until terminal, then inspect `data.job.result` and
its artifact references, budget, coverage, failures and stop reason. Use
`rforge jobs resume <id> --dir <dir>` for interrupted work; closing the MCP
connection cancels active searches and preserves resumable checkpoints. MCP
success envelopes are in `structuredContent` (also JSON text); tool errors are
failures, not empty research results. Keep the same provenance and reporting
rules whichever interface is used.

MCP does not expose approval, claim acceptance, full-text acquisition, or export
tools. Use CLI workflows for unsupported operations, and surface all human gates
to the invoker — switching interfaces never grants approval.

## Quick-start routing

| Goal                                         | Entry point                                                        |
| -------------------------------------------- | ------------------------------------------------------------------ |
| Find papers on a topic                       | 3-command quickstart above, then Phase 2 — Discover for more depth |
| Screen papers before downloading PDFs        | Phase 4 — Analyze (screening before oa fetch)                      |
| Study an open-source project                 | Phase 2 — Discover (OSS)                                           |
| Full systematic review                       | Phase 1 → 2 → 3 → 4 → 5                                            |
| Save research notes or artifacts to a folder | Phase 5 — Save                                                     |

---

## Phase 1 — Setup

### Option A: ResearchForge project folder (preferred)

```sh
rforge project create <path> --title "<title>"
rforge project inspect <path>
```

Creates `<path>/rforge.project.toml`, `<path>/data/`, and `<path>/rforge.lock.json`. All subsequent commands use `--project <path>`.

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

Engine runs work without an initialized project. When `rforge` is unavailable, write outputs directly to these files and populate `provenance.json` manually with `provenance_level: "agent_reported"` before finishing.

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

# Sweeps checkpoint to batch-state.json after each (query, source) pair. If a
# sweep is interrupted (timeout guard, Ctrl-C, killed process), the partial
# results stay in the topic dir — re-run the same command with --resume to
# complete never-run pairs and retry failed ones (exit 130/143 = interrupted).
rforge search batch --out <dir> --queries queries.txt --sources scholarly-fast --resume --continue-on-error --stats

# Retry only failed queries (legacy dirs without a checkpoint)
rforge search resume --failures <dir>/failures.jsonl --out <dir> [--dry-run]

# Show hit counts, library records, and checkpoint progress for a search dir
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
rforge oss inventory-roadmap <manifest.json> --todo CHECKLIST.md
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

Maintain the topic's `provenance.json` as a schema-v2 index before finishing. Engine commands (`search batch`, `citations expand`, `rank`, `lit`, and benchmarks) write immutable `provenance/runs/<run_id>.json` receipts and an engine-kind event in `provenance/events.jsonl` in their output directory, including project-less research folders. Without `--out`, rank saves `rankings/rankings.json` under the project (or current directory). Keep receipts and events unchanged. Copy actual run IDs from command JSON or receipt filenames into the index's `runs` array; never invent IDs or copy an engine level onto manual work. Capture depth, exact queries, sources, coverage stats, citation expansion attempts, outputs, and failures/rate limits. Include the batch manifest and checkpoint in `outputs` when present so effective caps, requests used, and `stop_reason` remain auditable. If `search batch --stats` was used, include `search-stats.txt` or equivalent stats output. Depth must be exactly `quick`, `standard`, or `comprehensive`; the validator rejects free-form variants such as `standard-plus`. Run `rforge --json version` and copy its `data` object into `rforge_version` instead of pasting the human-readable version line.

```json
{
  "schema_version": "2",
  "provenance_level": "agent_reported",
  "runs": ["<actual local engine run_id; omit when unavailable>"],
  "entries": [{"provenance_level": "agent_reported", "description": "Host-agent synthesis"}],
  "question": "<research question or task description>",
  "depth": "quick|standard|comprehensive",
  "sources": ["openalex", "arxiv"],
  "queries": ["<exact query string used>"],
  "timestamp": "<ISO 8601>",
  "rforge_version": {
    "version": "<version or not available>",
    "commit": "<commit or unknown>",
    "date": "<build date or unknown>"
  },
  "search_stats": { "openalex": 0, "arxiv": 0, "total_unique_dois": 0 },
  "citation_expand_attempted": ["<paper id or DOI>"],
  "citation_expand_succeeded": ["<paper id or DOI>"],
  "outputs": [{"path": "report.md", "sha256": "<64 hexadecimal characters from sha256sum report.md>"}],
  "errors": ["<rate limit, API failure, missing source, or empty output notes>"]
}
```

For manual fallback, set `runs` to `[]`, keep `provenance_level: "agent_reported"`, and list all saved outputs with output-directory-relative paths and SHA-256 hashes (do not hash the index itself). Manual entries may be `agent_reported` or `unverified`, never `engine_verified`. An engine-only index may use empty `outputs` and `entries`; its level is the minimum of its referenced runs. With manual outputs/entries, the computed level cannot exceed those entries. If an old run's output was replaced by a later engine run, retain its immutable receipt for history but reference the current run in the index. Without an index, displays use current runs: a newer validated engine event supersedes an older run only when its whole output set is covered. Explicitly validating a historical receipt still checks its original checksums; referencing stale runs in an index fails rather than silently raising its level.

`rforge provenance validate` computes the level rather than trusting the declaration. `engine_verified` requires recomputed output checksums and a matching engine event bound to the immutable run receipt; a false `engine_verified` declaration is an error. Schema v1 remains readable as `unverified` with a warning. Reports, packages and the Forge UI display computed levels; package privacy exclusions may lower the level. These are mechanical provenance checks, not scientific validation. They guard accidental mislabeling: a deliberate forger with file access is out of scope (no cryptographic signing). A run without `provenance.json` is incomplete.

---

## Verification gate

Before declaring research complete, verify:

- `rforge version` was run or marked unavailable in provenance.
- Every planned source/query has a saved result file, batch manifest, or explicit error note.
- Source coverage stats were run and recorded, with depth, effective caps, requests used, and stop reason reported; any escalation is justified by unsaturated budget exhaustion or the user's request.
- `report.md` or the requested output exists and cites exact papers, repositories, or artifacts.
- `provenance.json` exists, is valid JSON, and lists all outputs.
- `rforge provenance validate <folder>/provenance.json` passes (computed level, checksums/run linkage, enum depth, string-only errors, required fields); fix failures and report legacy warnings without presenting them as verification.
- Human-gated actions were surfaced rather than self-approved.

---

## Red lines

- Do not self-approve acquisition, privacy, or LLM-suggestion gates — surface them to the human invoker and stop.
- Do not hit live scholarly APIs inside any automated test or CI context.
- Do not finish a run without writing `provenance.json`.
- Do not accept LLM-generated suggestions without surfacing the queue to the human invoker first.
- Do not package or distribute reports before `oa privacy-review` is human-approved.
- Do not store copyrighted full text without an acquisition-approved record.
