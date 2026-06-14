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
- Crossref/OpenAlex/Semantic Scholar connectors.
- Duplicate report/merge/split UX.

Missing features:

- Rich Anystyle adapter options beyond JSON-producing `RFORGE_ANYSTYLE_CMD`.
- Rich plain-text bibliography import workflow beyond `rforge parse references --parser anystyle`.
- Rich parsed-reference normalization workflow beyond Crossref/OpenAlex/Semantic Scholar top-match reports.
- Rich confidence-aware review workflow beyond static ambiguous-reference queues.

## Recommended slice

Extend `rforge parse references --parser anystyle --file refs.txt --out refs.json` and `rforge parse review-refs` with reviewer decision persistence for ambiguous parsed references.
