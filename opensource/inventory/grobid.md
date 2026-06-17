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
