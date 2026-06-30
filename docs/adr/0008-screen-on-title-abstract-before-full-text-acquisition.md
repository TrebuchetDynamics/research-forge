# Screen on title/abstract before full-text acquisition

status: accepted

ResearchForge's screening workflow operates in two stages: title/abstract screening first (on metadata already in `results.jsonl`), then `rforge oa fetch` for records that pass, then full-text eligibility assessment on acquired PDFs. Full-text acquisition is a **post-screening** step, not a prerequisite.

This was the open question resolved during rforge self-research (2026-06-29): the 852-paper corpus across systematic-review-automation, automated-meta-analysis-techniques, and related topics confirmed that every major SR tool — ASReview, Rayyan, Abstrackr, Covidence — follows this order. PRISMA 2020 codifies it as a four-stage funnel: identify → deduplicate → title/abstract screen → full-text eligibility. Screening on metadata alone typically excludes 60–80% of records, so fetching full text before screening wastes significant fetch budget and bandwidth.

The practical consequence for the milestone sequence: **Milestone 4 (screening) does not depend on Milestone 3 (GROBID full-text parsing) completing first.** `rforge screen queue` and `rforge screen decide` can ship against the metadata already in `results.jsonl`. GROBID becomes a Milestone 3 tool for the second screening stage (full-text eligibility) on the already-screened subset.

## Considered options

- **Screen on title/abstract first (this decision)** — matches PRISMA 2020 stage 2, unlocks Milestone 4 ahead of Milestone 3, reduces fetch volume by the exclusion rate (~60–80%).
- **Require full-text before screening** — more thorough first pass, but forces `rforge oa fetch` to run on the entire candidate pool before any exclusions, fetching PDFs that will be excluded. Delays Milestone 4 until Milestone 3 (GROBID) is complete.
- **Single-stage screening on full text only** — collapses both stages into one. Non-standard (breaks PRISMA 2020 compliance), wastes full-text fetch on excluded records, and adds GROBID as a hard dependency for the simplest possible workflow.

## Consequences

- `rforge oa fetch` appears **after** `rforge screen decide` (title/abstract stage) in the recommended workflow, not before it.
- `rforge screen queue` and `rforge screen decide` operate on `results.jsonl` metadata (title, abstract, identifiers) and require no PDFs to function.
- The CONTEXT.md term **Two-stage screening pipeline** captures this order; **Title/abstract screening decision** and **Full-text eligibility decision** are the two distinct terms.
- The CSV export/import path (`rforge screen queue --out queue.csv` → edit → import) is the first-implementation interface for title/abstract screening, as it requires no TUI or web GUI and matches ASReview/Rayyan's export format.
- The web GUI screening queue (Milestone 4 stretch goal) and GROBID-backed full-text eligibility stage (Milestone 3 → 4 follow-on) can ship independently without blocking the core workflow.
- PRISMA flow counts must track both stages separately: records identified, records after deduplication, records after title/abstract screen, records after full-text eligibility.
