# ResearchForge self-improvement study: evidence from real rforge usage

## Method and limits

Evidence source: every `provenance.json` produced by rforge across all projects
under `/home/xel/git/*` (excluding research-forge's own agent worktrees and
build/dist copies). 128 provenance files across 9 distinct projects:
artificial-photosynthesis, fecim-lattice-tools, games, jepa,
pi-package-goal, polymarket-mega-bot, sdrhf, trebuchet-dynamics, visor-box.

Versions observed: v0.1.1 through v0.1.17 plus `dev`/`none`/dirty builds, over
2026-06-18 to 2026-07-08. Aggregates computed with `jq` over the provenance
corpus; error strings deduped and counted; file-structure patterns confirmed
by directory listing.

Limits: this is observational, not a controlled trial. Provenance is
self-reported by whatever rforge version wrote it, so schema drift between
versions is itself evidence. No live API calls were made; all evidence is from
already-saved artifacts. Adoption counts (screening, CITATIONS.md, vault) are
lower bounds — a feature may have been used without leaving a tracked artifact.

## Bottom line

Real usage is overwhelmingly **search → OA fetch → hand-written report.md**.
The roadmap invests heavily in breadth (44 source connectors, full
screening/extraction/meta-analysis/reports/vault pipeline), but measured
adoption of the later pipeline stages is near zero. The dominant pain is
**reliability of the four core sources that already exist** (Semantic Scholar
429s are 45% of all recorded errors), **provenance schema drift** across 27
version strings, and **no first-class support for iteration** (users hand-suffix
dirs `-v2`/`-v3`/`-wave` because rforge cannot re-run or diff a topic).

## Evidence

### E1. Semantic Scholar rate-limiting dominates failures

- 226 of 506 total error entries (45%) are Semantic Scholar HTTP 429 / rate-limit.
- Error text repeatedly says "configure an API key (check connector env vars)".
- Rate limits produce **empty `search-semantic-scholar-*.txt` files** silently
  (confirmed empty files in polymarket-mega-bot, jepa, sdrhf).
- Users compensate manually: "Semantic Scholar returned HTTP 429 ... OpenAlex/
  arXiv/Crossref carried the evidence" (jepa/time-series-jepa-encoders report).
- TODO.md:164 marks rate-limit/backoff `[x]` done — but it is not effective in
  practice; backoff alone cannot fix a hard 429 without an API key, and empty
  files are written instead of recorded failures.

### E2. Provenance schema drift across versions

- `rforge_version`: 27 distinct string formats — "rforge v0.1.7 (3c1ae39, ...)",
  "v0.1.17", "rforge dev (unknown, unknown)", "none", dirty builds.
- `depth`: free-form, not an enum — "standard", "comprehensive", but also
  "standard-plus", "comprehensive-lite", "standard search sweep script",
  "quick web research".
- `sources`: hand-typed inconsistent names — "rforge:oss search-plan" vs
  "rforge-oss-search-plan" vs "rforge:semantic-scholar citations".
- `errors`: mixed types — some strings, some nested JSON objects
  (`{"repo": "..."}`, `<HTTPError 404>`), breaking aggregation.
- Root cause: rforge never enforced a versioned provenance schema, so every
  version and every hand-edit diverges.

### E3. No first-class iteration/re-run support

- 13 topic dirs manually versioned: `-v2`, `-v3`, `-v019` (artificial-
  photosynthesis, fecim, jepa) and `-third-wave` through `-seventh-wave`
  (flutter-fractal-forge fractal-types).
- Users re-run the same research question and create a new directory because
  rforge has no `search refresh`, no topic versioning, and no diff between
  runs. The "next wave" pattern shows the same query re-issued repeatedly
  with no record of what changed.

### E4. Worktree duplication and no cross-worktree awareness

- polymarket-mega-bot has `.worktrees/coverage-gap-measurement/research/`
  duplicating 10+ topic dirs (and `CITATIONS.md`, `_foundational`) from the
  main worktree's `research/`.
- rforge has no concept of a shared research cache; each worktree re-runs and
  re-stores identical searches.

### E5. Pipeline-stage adoption is near zero; core loop is manual

| Feature | Roadmap milestone | Tracked adoption |
|---|---|---|
| `screening.jsonl` (screening workflow) | M4 | **0** |
| `CITATIONS.md` (quickstart headline) | M1 | **3** |
| `vault/` (Obsidian export) | — | **1** |
| `evidence-grid/gaps` (Comprehensive) | M5 | 22 |
| manual `report.md` | M7 (not used) | **134** |
| `failures.jsonl` (resume candidates) | — | 39 (not cleared) |

Users hand-write 134 `report.md` files and skip the screening/citation/vault
stages almost entirely. `search resume` exists but 39 `failures.jsonl` files
persist un-cleared, suggesting it is not run or does not fully recover.

### E6. OA-fetch framing conflates "no OA copy" with failure

- "No copyrighted full text acquired/downloaded" recorded as an error (9+
  occurrences). A paper simply having no legal OA copy is expected, not a
  failure; recording it as an error obscures real fetch failures.

### E7. Bulk-paper organization is manual

- fecim-lattice-tools has 68 `research/parsed/<paper-slug>/manifest.json` dirs
  hand-organized. rforge has no first-class "paper library" view that
  aggregates fetched+parsed papers across topics into one queryable set.

## Improvement plan (prioritized by evidence weight)

### P0 — Make the four core sources reliable (E1, E5)

1. **Semantic Scholar API key support end-to-end.** Read `RFORGE_SEMANTIC_SCHOLAR_API_KEY`,
   send the `x-api-key` header, and surface in `rforge doctor` whether the key
   is set. The 429 message already tells users to configure one; make it real.
2. **Never write an empty result file on a rate-limit.** A 429 is a failure:
   write to `failures.jsonl` and skip the empty `.txt`. Empty files pollute
   stats and look like zero-hit queries.
3. **Exponential backoff with jitter + per-source budget.** TODO:164 says done,
   but evidence shows it does not recover 429s. Re-verify against a real
   rate-limited run; the budget should defer the query to `failures.jsonl`
   rather than dropping it.
4. **`search resume` that actually clears `failures.jsonl`.** 39 uncleared
   failure files means resume is not trusted. Add a `--dry-run` estimate and a
   clear "N failures remaining" report after resume.

### P1 — Enforce a versioned provenance schema (E2)

5. **Bump `provenance.json` to a versioned schema** with an enum `depth`
   (`quick|standard|comprehensive`) and a normalized `rforge_version` object
   (`{major, minor, patch, commit, dirty}`). Reject free-form values at write
   time.
6. **Normalize `sources` to canonical connector IDs** at write time so
   `"rforge:oss search-plan"` and `"rforge-oss-search-plan"` cannot both exist.
7. **Coerce `errors` to strings.** Nested objects break `jq` aggregation;
   stringify or move structured errors to a separate `error_details` field.

### P1 — First-class iteration and re-run (E3)

8. **`rforge search refresh --dir <topic>`** that re-issues the stored queries,
   writes a new timestamped run, and reports **new vs unchanged vs gone** DOIs
   since the last run. This replaces the `-v2`/`-wave` hand-suffix pattern.
9. **Topic versioning in `manifest.json`**: a `runs[]` array with timestamps,
   query hash, source coverage, and DOI delta, so a topic dir is a timeline
   rather than a frozen snapshot.

### P2 — Worktree-aware shared cache (E4)

10. **A `~/.cache/rforge/` (or `XDG_CACHE_HOME`) source-response cache keyed by
    query+source+limit**, shared across worktrees. polymarket's duplicated
    worktree re-runs identical OpenAlex/arXiv queries. Cache hits skip the API
    and write provenance noting "served from cache".

### P2 — Fix OA-fetch error framing (E6)

11. **Distinguish "no OA copy exists" (expected, info) from "fetch failed"
    (error).** A paper with no Unpaywall OA location should not appear in
    `errors`; a network/HTTP failure should. Add an `oa_unavailable_count` to
    provenance instead.

### P3 — Right-size the roadmap against adoption (E5)

12. **Stop investing roadmap effort in screening/vault before core-loop
    reliability.** Screening has 0 adoption and CITATIONS.md has 3; the 134
    hand-written reports show users want **search+fetch+provenance to be
    excellent**, not more pipeline stages. Reorder so P0/P1 above precede any
    new M4–M7 work.
13. **A `rforge library aggregate --research-dir <dir>`** that builds one
    queryable paper set across all topic subdirs (replaces fecim's 68 manual
    `parsed/` dirs). This is the bulk-paper view users are hand-rolling.

## Claim hygiene

- "45% of errors are Semantic Scholar 429s": 226/506 from `jq` aggregation over
  128 provenance files (2026-06-18 to 2026-07-08). Not a controlled sample;
  projects with more Semantic Scholar queries are over-represented.
- "0 screening adoption": absence of `screening.jsonl` across all scanned
  projects. Lower bound; a project could screen outside rforge.
- "134 hand-written reports": `report.md` count excluding research-forge agent
  worktrees. Some may be rforge-generated, but provenance `outputs` rarely list
  a generated report, and report prose is project-specific, indicating manual
  authorship.

## Evidence gaps

- No data on **why** users skip screening — could be the CSV round-trip is too
  heavy, or they genuinely don't need it for lightweight scouting. Needs a user
  question, not artifacts.
- No measurement of **`search resume` failure mode** — whether users tried it
  and it failed, or never tried. The 39 uncleared `failures.jsonl` files are
  ambiguous.
- No data on **citation-expand reliability** beyond success/fail counts; the
  structured graph quality was not assessed.
- rforge versions in provenance are self-reported and inconsistent (E2), so
  per-version error rates are approximate.

## Implications and next steps

The highest-leverage work is not more sources or more pipeline stages. It is:
make the four sources users actually rely on (openalex, arxiv, crossref,
semantic-scholar) reliable under rate limits, enforce a stable provenance
schema so 128 files can be aggregated without drift, and add iteration support
so users stop hand-versioning directories. P0 and P1 are each independently
shippable TDD slices; P2/P3 can follow once the core loop is trustworthy.

Recommended first slice: P0 item 2 + 3 (no empty files on 429; verify backoff
recovers) — smallest, directly attacks the 45% error category, and is testable
offline against a fixture 429 response.
