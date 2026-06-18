# Automated meta-analysis techniques for ResearchForge

## Method and limits

I ran a Standard-depth ResearchForge sweep for techniques that can support automated or semi-automated meta-analysis in ResearchForge. Queries covered automated meta-analysis, evidence extraction, living reviews, screening, machine learning, NLP, and LLM-based systematic-review automation. Sources searched were OpenAlex, Crossref, Semantic Scholar, and arXiv. The sweep returned 380 raw records and 328 DOI/title-deduplicated records. Semantic Scholar rate-limited four searches and Crossref produced one normalization failure; these are recorded in `failures.jsonl`.

No copyrighted full text was downloaded. Claims below are based on retrieved metadata, titles, venues, DOI records, and citation expansion graphs for selected seed papers.

## Bottom line

Automated meta-analysis should be treated as a pipeline of gated assists, not an autonomous one-click reviewer. The strongest technical pattern for ResearchForge is: automated search and dedupe, ML/LLM-assisted screening, schema-constrained data extraction with passage-level provenance, deterministic analysis scripts, and human review gates before acquisition, extraction acceptance, and final report claims.

## Main themes

### 1. Screening automation is mature enough for assistive queues

Rayyan is a baseline example of collaborative systematic-review screening infrastructure: **"Rayyan—a web and mobile app for systematic reviews"** (`10.1186/s13643-016-0384-4`). It is not a full meta-analysis automator, but it proves the value of a focused screening cockpit and reviewer workflow.

Recent work moves from collaboration UI toward AI-assisted triage. **"Development and evaluation of prompts for a large language model to screen titles and abstracts in a living systematic review"** (`10.1136/bmjment-2025-301762`) directly targets LLM prompt design for screening. **"Artificial Intelligence Software to Accelerate Screening for Living Systematic Reviews"** (`10.1007/s10567-025-00519-5`) also surfaced as relevant living-review screening automation.

Implication for ResearchForge: screening automation should output ranked queues, uncertainty flags, rationales, and audit bundles. It should not directly include/exclude records without reviewer decisions.

### 2. Data extraction is the critical bottleneck and needs schema constraints

The clearest older anchor is **"Automating data extraction in systematic reviews: a systematic review"** (`10.1186/s13643-015-0066-7`). It supports treating extraction as a distinct technical problem rather than assuming search/screening automation solves meta-analysis.

Newer records such as **"OpenExtract: Automated Data Extraction for Systematic Reviews in Health"** and **"Diagnosing Structural Failures in LLM-Based Evidence Extraction for Meta-Analysis"** appeared in the sweep, though not all had DOI metadata in the deduplicated output. This points toward a ResearchForge requirement: extraction needs typed schemas, exact support passages, parser offsets, reviewer status, correction history, and downstream analysis inclusion state.

Implication for ResearchForge: every automated extraction should create a suggestion queue with fields like outcome, comparator, effect measure, sample size, variance/CI, follow-up time, and risk-of-bias support. The accepted evidence table should remain reviewer-controlled.

### 3. Living systematic reviews map well to ResearchForge's provenance model

A highly relevant recent preprint is **"A Living Systematic Review Engine: LLM-Automated Evidence Surveillance Validated Against a Published Meta-Analysis of Statins for Sepsis"** (`10.21203/rs.3.rs-9308492/v1`). It suggests an architecture where the system continuously monitors evidence, compares against an existing meta-analysis, and updates queues.

This is aligned with ResearchForge's local project store and provenance log: watched searches, source snapshots, dedupe decisions, screening audit bundles, extraction grids, and analysis manifests can all be replayed.

Implication for ResearchForge: build automation around repeatable surveillance and deltas, not around hidden model state. A living-review run should emit a manifest, changed-record set, screening queue, extraction suggestions, and analysis-readiness diff.

### 4. Generative AI is useful, but evidence synthesis literature emphasizes auditability

The sweep found **"Generative artificial intelligence use in evidence synthesis: A systematic review"** (`10.1017/rsm.2025.16`) and **"Artificial Intelligence and Automation in Evidence Synthesis: An Investigation of Methods Employed in Cochrane, Campbell Collaboration, and Environmental Evidence Reviews"** (`10.1002/cesm.70046`). These should guide ResearchForge away from unverified prose generation and toward auditable task-specific assists.

A practical evidence-map source is **"Machine Learning Tools To (Semi-)Automate Evidence Synthesis: A Rapid Review and Evidence Map"** (`10.23970/ahrqepcwhitepapermachine2`). This is especially relevant for cataloging possible tool classes rather than choosing a single model.

Implication for ResearchForge: represent LLM outputs as suggestions with model/version/prompt/support metadata. Keep final claims grounded in accepted evidence and citation traceability.

### 5. Automated meta-analysis still needs deterministic statistical execution

The relevant records separate evidence synthesis automation from statistical meta-analysis. ResearchForge should keep R/metafor or equivalent deterministic engines as the analysis layer, with automation used to prepare inputs, check consistency, flag missing variance/effect-size fields, and produce sensitivity/influence diagnostics.

The record **"Transforming evidence synthesis: A systematic review of the evolution of automated meta-analysis in the age of AI"** (`10.1017/rsm.2025.10065`) is the most directly titled source for this topic and should be prioritized for full-text review if legally accessible.

## Performance claims hygiene

Do not claim that LLMs or ML tools "automate meta-analysis" globally. Instead, name the task and exact evidence:

- Screening: cite `10.1136/bmjment-2025-301762` for LLM title/abstract screening prompt evaluation.
- Data extraction: cite `10.1186/s13643-015-0066-7` for automated extraction as a systematic-review topic.
- Living surveillance: cite `10.21203/rs.3.rs-9308492/v1` for a preprint validation against an existing statins-for-sepsis meta-analysis.
- Evidence synthesis AI methods: cite `10.1017/rsm.2025.16` and `10.1002/cesm.70046` for broader reviews/investigations.

Headline numbers about time saved, accuracy, recall, or error rates should not be used until the exact paper is read and the task, dataset, comparator, and confidence intervals are extracted.

## Evidence gaps

- Semantic Scholar rate limiting left gaps in citation-rich coverage.
- Some highly relevant 2025/2026 records are preprints or newly indexed items; peer-reviewed status should be checked before product claims.
- The sweep found many irrelevant domain meta-analyses where "automated" referred to insulin delivery or other domain-specific automation, not review automation.
- Full-text methods details were not inspected, so extraction accuracy, screening recall, and workload savings are not asserted here.

## Implications for ResearchForge

Recommended implementation pattern:

1. **Search sweep and dedupe**: use `rforge search batch` to run multi-query/multi-source sweeps and produce deduped JSONL plus failure logs.
2. **Screening cockpit**: active-learning or LLM-assisted queues with uncertainty, reviewer assignment, conflict/adjudication, and audit bundle export.
3. **Extraction suggestion queues**: schema-constrained extraction with exact passage/table support, parser offsets, model metadata, and reviewer decisions.
4. **Analysis readiness checks**: deterministic validation that accepted evidence contains effect sizes, variance/CI, group counts, outcome definitions, and risk-of-bias fields before analysis.
5. **Statistical execution**: keep analysis in deterministic engines (metafor/PyMARE adapters), with scripts, settings, warnings, plots, and checksums stored.
6. **Claim traceability**: final report paragraphs, tables, and figures must link to accepted evidence and block export on weak/unresolved claims.

The safest product language is "ResearchForge automates retrieval, dedupe, queue generation, extraction suggestions, analysis preparation, and provenance capture; humans approve scientific decisions and final claims."
