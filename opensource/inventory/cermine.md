# CERMINE study note

- Repository/ecosystem: `CeON/CERMINE`.
- Area: PDF metadata and bibliography extraction.
- Disposition: `adapter-only`.
- License/action constraint: use only as an optional external fallback; do not vendor Java artifacts or sample documents without review.

## Why it matters

CERMINE is a candidate fallback when GROBID output is unavailable or low quality. ResearchForge should compare parser outputs rather than assuming one parser is always best.

## Patterns to learn

- Treat parser choice as a reproducibility parameter.
- Compare metadata, section, and reference counts across parsers.
- Surface parser failures and quality warnings to reviewers.
- Keep fallback outputs isolated until reviewed/accepted.

## ResearchForge status

Implemented nearby capabilities:

- GROBID service check and adapter.
- Parsed-document storage.
- Report/evidence audit paths.
- Parser comparison reports over two parsed-document JSON files through `rforge parse compare --left <parsed.json> --right <parsed.json> --out <report.json>`, including fallback candidate scoring metadata.
- Parser-run manifests record parser source/version/command, input/output checksums, TEI/JSON output kind, license constraints, shareability, warnings, and reviewer gates.
- Multi-engine parser arbitration scores GROBID/S2ORC-style/PaperMage/CERMINE/Science Parse/Anystyle outputs per field, routes conflicts to review, records reviewer selection reasons, and emits reconciliation outputs.
- The method-comparison workbench compares parser choices, retrieval backends, screening rankers, effect-size models, and publication-bias diagnostics side-by-side before recording the reviewer-selected method locked into the final report.

Missing features:

- Parser fallback orchestration beyond comparison candidate scoring.
- CERMINE adapter seam beyond fallback metadata.
- Review UI for conflicting parser outputs.

## Recommended slice

Add parser candidate scoring and a CERMINE adapter seam after comparison reports are exercised on local fixtures.
