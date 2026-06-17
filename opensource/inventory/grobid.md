# GROBID study note

- Repository/ecosystem: `kermitt2/grobid`.
- Area: scholarly PDF parsing: metadata, sections, references, TEI XML.
- Disposition: `adapter-only`.
- License/action constraint: call as external/local service; do not vendor GROBID code.

## Why it matters

GROBID is the best-known open scholarly PDF parser and is already the recommended primary parser in the PRD.

## Patterns to learn

- Prefer structured TEI output over ad hoc text scraping.
- Reference extraction and section segmentation need stable passage IDs.
- Parser version must be recorded for reproducibility.
- Parsing failures and low-confidence extraction should stay visible to reviewers.

## ResearchForge status

Implemented nearby capabilities:

- GROBID service check.
- GROBID client/adapter with mocked TEI tests.
- Parsed-document model with sections/passages/references.
- Parser-run manifests record parser source/version/command, input/output checksums, TEI/JSON output kind, license constraints, shareability, warnings, and reviewer gates.
- The forge workflow DAG includes parse checkpoints with inputs, outputs, provenance actions, and restart-safe skips.
- The local project knowledge graph merges GROBID parsed references with collections/tags, OpenAlex concepts, citation edges, evidence, screening, analysis, and report claims for `rforge knowledge query`.
- Multi-engine parser arbitration scores GROBID/S2ORC-style/PaperMage/CERMINE/Science Parse/Anystyle outputs per field, routes conflicts to review, records reviewer selection reasons, and emits reconciliation outputs.
- The `/parsing` HTMX arbitration screen compares parser outputs field-by-field with confidence, raw text, offsets, warnings, and accept/correct/reject controls.
- The `/evidence` extraction grid links fields to source passage/table/figure/equation support, parser offsets, PDF views, reviewer status, correction history, and downstream analysis inclusion.
- Evidence gap analysis cross-checks the research question, screened-in studies, parsed passages, accepted evidence fields, full-text acquisition, citation-locked claims, and analysis inputs before final inclusion.
- Reproducible review packages bundle the meta-analysis spine first-done artifact with project manifests, lockfiles, source plans, dedupe decisions, parser manifests, screening audit, extraction schema, accepted evidence, analysis/report artifacts, redaction policy, replay helper, audit placeholder, and checksums; `/package` previews parser manifests and package readiness before creation.
- Cross-tool benchmarks report deterministic fixture metrics for discovery recall, dedupe precision, parser field accuracy, reference normalization, retrieval quality, screening effort savings, and report/package reproducibility.
- Citation-locked synthesis can draft query expansions, screening rationales, extraction candidates, and report prose only when every suggested sentence has exact source support and remains unaccepted until reviewer review.
- The method-comparison workbench compares parser choices, retrieval backends, screening rankers, effect-size models, and publication-bias diagnostics side-by-side before recording the reviewer-selected method locked into the final report.
- Citation-to-evidence trace views link report claims back to parser outputs, passage offsets/text, PDFs, accepted evidence, effect-size rows, and source/reference-manager provenance.
- Opt-in real GROBID endpoint e2e listed in TODO as complete.

Missing features:

- Parser confidence/quality report.
- Reference normalization against Crossref/OpenAlex/Semantic Scholar.
- Full bibliography-to-citation-graph import.
- Multiple parser fallback comparison.
- Per-passage provenance in generated reports.

## Recommended slice

Add parsed-reference normalization: take GROBID references, search by DOI/title where possible, and attach normalized identifiers while preserving raw reference text.

Acceptance target:

```sh
rforge references normalize --paper <id> --source crossref|openalex
```
