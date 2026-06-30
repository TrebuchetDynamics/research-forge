# ResearchForge Context

This glossary captures stable project language for ResearchForge. It is not a feature specification; implementation details belong in the PRD, roadmap, development plan, TODO, or ADRs.

## Glossary

### ResearchForge

The open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, meta-analysis, and auditable reporting.

### `rforge`

The planned command-line tool for ResearchForge workflows.

### Research project

A local ResearchForge workspace containing the research question, source records, documents, screening decisions, extracted evidence, analyses, reports, project manifest, lockfile, and provenance.

### Project manifest

The human-readable project configuration that describes a ResearchForge project's research question, sources, schemas, storage mode, external services, and export settings.

### Workflow lockfile

The machine-written record of tool versions, external-service parameters, parser/model versions, and analysis settings needed to reproduce project outputs.

### Project health report

An actionable summary of Research project invariants such as project manifest, Workflow lockfile, and local storage availability, intended for CLI and future local web GUI status displays.

### Provenance

The audit trail that records where research data came from, which actions were taken, which tools and parameters were used, and which source material supports claims.

### Paper record

A normalized scholarly metadata entry for a paper or preprint, preserving identifiers, source-specific metadata, and source provenance.

### Document asset

A local PDF, XML, JATS, HTML, text file, or related full-text artifact with acquisition source, legality/OA status, license metadata where available, checksum, and provenance.

### Parsed document

A structured representation of a document asset, including sections, references, passages, and optionally tables, figures, equations, or other scientific content units.

### Passage

A stable, citable unit of parsed document text that can support retrieval results, evidence extraction, and report audit links.

### Title/abstract screening decision

An include, exclude, or uncertain judgment made on a paper's **title and abstract metadata alone**, before any full-text PDF is acquired. The first and primary screening stage. Records that survive this stage proceed to full-text acquisition; excluded records are never downloaded.

_Avoid:_ "screening decision" alone (ambiguous about stage), "pre-screening"

### Full-text eligibility decision

An include, exclude, or uncertain judgment made after reading the full-text PDF of a paper that passed [[title/abstract screening decision]]. The second screening stage — applied only to the subset of records whose PDFs were acquired.

_Avoid:_ "screening decision" alone (ambiguous about stage), "secondary screening"

### Screening queue CSV

The self-contained CSV file emitted by `rforge screen queue --out queue.csv` for offline title/abstract review. Columns filled by rforge: `doi`, `arxiv_id`, `title`, `authors`, `year`, `abstract`, `source`. Columns filled by the reviewer: `decision` (`include`/`exclude`/`uncertain`) and `reason` (optional free text). Includes abstract text so reviewers can screen without any external tool. Imported back with `rforge screen import --csv queue.csv`, which writes entries to the [[Screening decision store]].

_Avoid:_ "label file" (ASReview terminology), "screening export"

### Screening decision store

The append-only JSONL file (`screening.jsonl`) in a topic dir that records every [[title/abstract screening decision]] and [[full-text eligibility decision]]. Each line carries: identifier (DOI or ArXivID), decision (`include`/`exclude`/`uncertain`), stage (`title-abstract` or `full-text`), reviewer, optional reason, and timestamp. The CSV export/import path is the primary Milestone 4 interface: `rforge screen queue --out queue.csv` emits pending records; the reviewer fills in the `decision` and `reason` columns; `rforge screen import --csv queue.csv` writes entries back to `screening.jsonl`.

_Avoid:_ "decisions file", "labels file" (ASReview terminology), "screening database"

### Two-stage screening pipeline

The ResearchForge screening workflow: [[title/abstract screening decision]] first (on metadata already in results.jsonl), then `rforge oa fetch` for included records only, then [[full-text eligibility decision]] on acquired PDFs. This order means full-text acquisition is a post-screening step, not a prerequisite. Milestone 4 (screening) does not depend on Milestone 3 (GROBID) completing first.

### Evidence item

A structured extracted value or claim that is linked to exact source support such as a passage, table, figure, equation, dataset, or citation.

### Analysis run

A reproducible statistical execution over accepted evidence, including inputs, model settings, scripts or notebooks, tool versions, outputs, warnings, and checksums.

### Meta-analysis spine

The first-priority ResearchForge super-tool workflow for meta-analysis authors: a reproducible path from source planning, import/deduplication, legal full-text acquisition, parser arbitration, screening, and evidence extraction through statistical analysis and auditable reporting. Broader research-cockpit features should build on this spine rather than bypass it.

### Report build

A generated research report export with citations, evidence tables, screening summaries, analysis outputs, audit appendix, and build metadata.

### Reproducible review package

The first "done" artifact for the Meta-analysis spine: a portable ResearchForge package that proves the review can be audited and replayed, including report outputs plus manifests, lockfiles, source query plans, deduplication decisions, parser manifests, screening audit, extraction schema, accepted evidence, analysis artifacts, redaction policy, and checksums. The `/package` reproducibility/export center previews package contents, redaction results, checksums, lockfiles, external-tool versions, parser manifests, analysis artifacts, report outputs, and reviewer decision logs before package creation.

### Local web GUI

The ResearchForge browser interface launched by `rforge ui` for local visualization and review of project state, papers, diagrams, meta-analysis outputs, and report artifacts generated by CLI workflows.

### Local research cockpit

The final purpose of the Local web GUI: a local human review and navigation layer for opening Research projects, inspecting papers, exploring CLI-generated artifacts, and guiding small local workflow actions while the CLI remains the reproducible automation path.

### Go + HTMX

The selected primary stack for the Local web GUI. "Go HTMLX" is treated as an alias for this term.

### OSS repository study

A ResearchForge record of an open-source project studied for possible integration or design reference, including metadata, license, risks, notes, and integration decisions.

### OSS study disposition

The classification for how ResearchForge may use an OSS repository study: `pattern-reference` by default, or explicitly escalated to `adapter-only`, `integrate`, `needs-license-review`, or `avoid` before any dependency, code, schema, fixture, or asset is used.

### Artificial photosynthesis fixture topic

The main deterministic end-to-end test topic for ResearchForge. Use the spelling "artificial photosynthesis" in code, fixtures, docs, and test names.

### Repository-embedded ResearchForge config

A `.researchforge` configuration file or directory created in the repository where `rforge` is run. It records repo-local ResearchForge settings without requiring users to leave their working repository.

### Default ResearchForge workspace

The default Research project folder for repo-embedded use: `<repo>/research-forge/`. ResearchForge should assume a repository may already contain academic files, PDFs, notes, or other research assets and should discover or import them only through explicit, provenance-recorded workflow steps.

### Scientific benchmarking meta-analysis

A meta-analysis mode that pools a single continuous performance metric (e.g., solar-to-hydrogen efficiency, AUROC, F1) reported per study, without requiring a treatment/control arm pair. Study-level moderators (protocol, dataset, measurement condition, year, lab) replace the arm-level covariates used in clinical meta-analysis. The statistical model is a random-effects pooling of the raw outcome value, with variance estimated from reported confidence intervals, SEs, or study-level replication data. Distinct from **clinical meta-analysis**, which requires paired arm data to compute effect sizes (SMD, OR, RR). Each benchmarking analysis run is scoped to one outcome type with compatible units.

The first end-to-end target domain is **artificial photosynthesis**, primary outcome **solar-to-hydrogen (STH) efficiency %**. STH% is the canonical summary metric in photoelectrochemical device papers, normally reported in the abstract, with a natural 0–100 % scale and well-established moderator variables (electrode material, light source, electrolyte, measurement standard).

_Avoid:_ "benchmarking review", "performance meta-analysis" (too broad), "aggregate analysis"

### STH% extraction schema

The built-in [[abstract extraction]] schema for solar-to-hydrogen efficiency benchmarking. Required fields (analysis readiness check blocks if absent): `value_pct`, `device_type`, `auxiliary_bias`, `measurement_standard`, `verbatim_quote`, `confidence`. Optional fields (extracted if present, flagged if absent, not blocking): `ci_lower`, `ci_upper`, `se`, `target_reaction`, `electrode_material`, `electrolyte`, `illumination_intensity_mwcm2`, `active_area_cm2`. `device_type` values: `pec` / `pv-electrolysis` / `particle-suspension` / `biohybrid`. `auxiliary_bias` values: `unassisted` / `assisted` / `unknown`. `measurement_standard` values: `am1.5g-100` / `non-standard` / `unknown`.

### Benchmarking outcome

The single continuous metric being pooled in a [[scientific benchmarking meta-analysis]] run. Must have compatible units across all included studies. Examples: solar-to-hydrogen (STH) efficiency % for photoelectrochemical device papers; AUROC for ML classifier evaluations. Each analysis run is scoped to one benchmarking outcome type. A study contributes one `yi` value (the reported measurement) and one `vi` value (reported variance or floor-imputed variance per ADR-0007).

### Variance floor

The minimum variance assigned to a study in a [[scientific benchmarking meta-analysis]] when the paper does not report a standard error or confidence interval. Set via `--variance-floor` at `rforge analysis prepare` time. Default `0.0025` (±5% relative uncertainty). Stored in the analysis manifest as a provenance parameter. Studies using the floor are tagged `vi_source: floor`. See ADR-0007.

_Avoid:_ "default variance", "imputed SE"

### Abstract extraction

The primary extraction path for [[scientific benchmarking meta-analysis]]: a structured LLM call that receives a paper's abstract text, a target field definition (name, unit, prompt hint), and returns a measurement value, verbatim quote, unit confirmation, and confidence score. Populates `EvidenceItem.Values` without requiring full-text acquisition or GROBID parsing. Manual `rforge extract add` is the fallback for abstracts where extraction fails or returns no value. Requires extending `SuggestRequest` to carry `AbstractText` and `TargetField`.

_Avoid:_ "LLM extraction" (too broad — does not imply abstract-only scope)

### Benchmarking moderator

A study-level covariate included in a [[scientific benchmarking meta-analysis]] in place of arm-level covariates used in [[clinical meta-analysis]]. Examples for artificial photosynthesis: electrode material, light source spectrum, electrolyte, measurement standard (NREL-calibrated or not), publication year.

### Clinical meta-analysis

A meta-analysis mode that requires paired arm data (experimental vs. control) to compute a standardized effect size (SMD, odds ratio, risk ratio, mean difference). The existing `rforge analysis prepare --effect smd|log-odds-ratio|...` pipeline is built for this mode. Distinct from **scientific benchmarking meta-analysis**.

### Retrieval-first, provenance-first, statistics-first, LLM-assisted

The core ResearchForge principle: retrieve and cite source material first, preserve provenance, use auditable statistical methods, and allow LLMs only as assistants for tasks such as query expansion or extraction suggestions.
