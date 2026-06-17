# Anystyle study note

- Repository/ecosystem: `inukshuk/anystyle`.
- Area: reference string parsing and bibliography extraction.
- Disposition: `adapter-only`.
- License/action constraint: call as an optional external parser; do not vendor code or training data without review.

## Why it matters

ResearchForge already extracts references through GROBID-like parsed documents. Anystyle is a useful fallback for noisy plain-text bibliographies and manual reference lists.

## Patterns to learn

- Preserve raw reference strings beside parsed fields.
- Report confidence/warnings for parsed fields.
- Keep parser engine/version in provenance.
- Route parsed DOI/title candidates into source-normalization connectors rather than silently merging.

## ResearchForge status

Implemented nearby capabilities:

- Parsed reference records from parser outputs.
- `rforge parse references --parser anystyle` preserves parsed DOI/title, raw reference strings, confidence scores, parser name, and parser version.
- `rforge parse review-refs` creates a manual review queue for low-confidence or incomplete parsed references.
- Parser-run manifests record parser source/version/command, input/output checksums, reference JSON output kind, license constraints, shareability, warnings, and reviewer gates.
- `rforge parse adjudicate-ref` and `rforge parse adjudicated-refs --ambiguity-out` persist reviewer accept/correct/reject/defer decisions, provenance links, corrected reference fields, and exportable ambiguity queues for Anystyle/GROBID/S2ORC-normalized matches.
- Multi-engine parser arbitration scores GROBID/S2ORC-style/PaperMage/CERMINE/Science Parse/Anystyle outputs per field, routes conflicts to review, records reviewer selection reasons, and emits reconciliation outputs.
- The `/parsing` HTMX arbitration screen compares parser outputs field-by-field with confidence, raw text, offsets, warnings, and accept/correct/reject controls.
- Crossref/OpenAlex/Semantic Scholar connectors.
- Duplicate report/merge/split UX.
- Source-fusion identity resolution merges DOI/arXiv/PMID/PMCID/OpenAlex/Semantic Scholar/Crossref/Zotero IDs with explainable rules, conflict records, reversible merge/split decision logs, and `library identity-decision apply` support.
- Cross-tool benchmarks report deterministic fixture metrics for discovery recall, dedupe precision, parser field accuracy, reference normalization, retrieval quality, screening effort savings, and report/package reproducibility.

Missing features:

- Rich Anystyle adapter options beyond JSON-producing `RFORGE_ANYSTYLE_CMD`.
- Rich plain-text bibliography import workflow beyond `rforge parse references --parser anystyle`.
- Rich parsed-reference normalization workflow beyond Crossref/OpenAlex/Semantic Scholar top-match reports.
- Higher-touch UI for confidence-aware review beyond CLI/exported ambiguity queues.

## Recommended slice

Build an HTMX reviewer screen over the persisted reference-adjudication log and ambiguity queue exports.
