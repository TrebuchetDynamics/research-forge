# Making ResearchForge better and more successful: evidence and implemented change

## Method and limits

Question: how can ResearchForge become a better and more widely adopted tool?

Two evidence layers:

1. **Fresh retrieval (this run, 2026-10-20).** `rforge search batch` with 10 query
   families across 4 sources (`openalex`, `arxiv`, `crossref`,
   `semantic-scholar`), 10 records per query, `--continue-on-error --stats`.
   Coverage: openalex 100, arxiv 100, crossref 100, semantic-scholar 10 records;
   310 records, 294 deduplicated, 225 unique DOIs, 9 failed queries. Citation
   expansion (depth 1, both directions, OpenAlex) for two anchors. No full text
   was downloaded.
2. **Repository evidence already in scope.** Two prior studies in this repository:
   [`rforge-usage-improvement-study`](../rforge-usage-improvement-study/report.md)
   (128 real `provenance.json` files across 9 projects, v0.1.1–v0.1.17) and
   [`open-source-academic-tools`](../open-source-academic-tools/report.md).

Limits: this is a **bounded standard-depth sweep (10 variants), not the
Comprehensive depth the skill defaults to**; Semantic Scholar answered only 1 of
10 queries (HTTP 429), so its coverage is a lower bound. The generic queries
retrieve a noisy mix; themes below are restricted to records that directly
concern research-software practice. Prior-study numbers are observational.

## Bottom line

The evidence does not point to more features. It points to **distribution and
first-run reliability**: (a) research software is judged by whether a newcomer
can install it and reproduce a result, and (b) this repository's own documented
one-command install currently fails for every user because the newest release
predates the capability gate the plugins require. I verified (b) live and fixed
the recoverable half of it: the installer now rebuilds a checkout when the
release binary fails the gate, instead of leaving the user with a broken engine.

## Main themes

### 1. Documentation quality and newcomer onboarding predict adoption

- `10.2139/ssrn.6864982`, **A Software Engineering Perspective on Documentation
  Quality: Modeling Its Impact on Newcomer** (2026) — documentation quality is
  modelled as a direct driver of newcomer onboarding outcomes.
- `10.1007/979-8-8688-2157-8_2`, **The Command Line** (2026) — the CLI is still the
  assumed interface for developer workflows; its ergonomics matter.
- `10.1039/d6dd00121a`, **Scikit-package — software packaging standards and
  roadmap for sharing reproducible** (2026, with public review/decision letters in
  the corpus) — packaging standards are treated as research output, not overhead.

Implication: the install path and the first successful command are the product.
Both must work on a fresh machine, and the failure mode must name the fix.

### 2. Transparency, provenance and trust are the differentiators for research tools

- `10.1002/cesm.70096`, **Transparency without reproducibility? Open science
  practices in systematic reviews** (2026) — transparency claims that cannot be
  reproduced are discounted by reviewers.
- `10.2196/preprints.113791`, **Trust, Privacy, and Acceptance Across
  Data-Provenance Conditions** (2026) — provenance presentation changes acceptance.
- `10.1145/3774748.3787757`, **Investigating How Governance Shapes the Health of
  Open Source Software Projects** (2026) — governance structure is measurable in
  project health, not just in policy documents.

Implication: ResearchForge's provenance-first stance is the right bet, but it only
counts if the artifacts are produced by the default path and are self-consistent
(see the prior study's schema-drift finding).

### 3. Reproducible benchmarks are how tools earn technical credibility

- `10.64898/2026.06.02.729500`, **A Reproducible and Extensible Benchmark of
  Supervised Cell Type Annotation Tools** (2026)
- `10.48550/arxiv.2605.21800`, **stable-worldmodel: A Platform for Reproducible
  World Modeling Research and Evaluation** (2026)

Implication: the repository's checksum-locked engine-tier baseline is aligned with
how comparable tools argue for trust; it should stay linked from the front page.

### 4. Core-loop reliability dominates pipeline breadth

- Fresh evidence: **9 of 10 Semantic Scholar queries failed with HTTP 429** in
  this run, matching the prior study's finding that 45% of recorded errors were
  Semantic Scholar rate limits.
- Prior study (`rforge-usage-improvement-study`, E5): 134 hand-written
  `report.md` files versus 0 `screening.jsonl` and 3 `CITATIONS.md` — users adopt
  search → fetch → write, not the later pipeline stages.

Implication: reliability and iteration of the four core sources beat new stages.

## Claim hygiene

- "Documentation quality drives onboarding" is a modelled relationship in one 2026
  preprint/SSRN record, not a controlled experiment; it is consistent with, but not
  proof of, this project's experience.
- The 429 rate (9/10) is one run without an API key; it is not a stable error rate.
- Prior-study adoption counts are lower bounds derived from artifacts; absence of
  `screening.jsonl` does not prove nobody screened outside rforge.
- No user interviews were conducted; the "why" behind low pipeline adoption
  remains unmeasured.

## Evidence gaps

- No direct evidence on which install failure message users actually encounter
  (needs telemetry or issue reports).
- Semantic Scholar coverage is 1/10 queries, so citation-oriented literature is
  under-represented in this sweep.
- No data on whether the new automatic checkout rebuild helps real users versus
  only developers with Go installed.

## Implications and next steps

Implemented in this change (evidence: live install failure + theme 1):

1. **Installer recovery for a release that lags the capability gate.** When the
   downloaded release fails `rforge capabilities`, and this script runs from a
   buildable checkout with Go, it now rebuilds the checkout (with commit/date
   metadata), reports why, and succeeds. Without a checkout it still fails closed
   with the existing guidance. Covered by two installer tests and verified against
   the real v0.1.21 release.

Recommended next (not implemented here):

2. **Publish a release at or above the required capability version** so the plain
   `curl | bash` path works without a checkout. This is the remaining half of the
   same blocker and the highest-leverage adoption action available.
3. Keep the front page's install, proof and trust sections ahead of pipeline
   breadth (theme 4), and keep the checksum-locked baseline linked (theme 3).
