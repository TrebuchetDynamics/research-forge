# s2orc-doc2json study note

- Repository/ecosystem: `allenai/s2orc-doc2json`.
- Area: scholarly full-text JSON conversion from GROBID/LaTeX/PDF pipelines.
- Disposition: `adapter-only`.
- License/action constraint: use as an external conversion tool or format reference only; do not vendor code or fixtures without license review.

## Why it matters

S2ORC-style JSON is a practical interchange shape for sections, paragraphs, bibliography entries, and citation spans. ResearchForge can learn from the passage/reference model while keeping local parsing adapters explicit.

## Patterns to learn

- Preserve bibliography entries separately from in-text citation spans.
- Keep section hierarchy and paragraph offsets stable enough for evidence provenance.
- Record parser version and source PDF/TEI checksums.
- Treat generated JSON as derived data, not authoritative metadata.

## ResearchForge status

Implemented nearby capabilities:

- GROBID adapter producing parsed sections/passages/references.
- S2ORC-style JSON reader available through `rforge parse --parser s2orc --s2orc <file>`.
- Citation graph export and reports.
- Evidence items can cite passage references.

Missing features:

- Rich S2ORC JSON import/export adapter beyond the current reader.
- Citation-span offsets linked to parsed passages.
- Bibliography normalization against Crossref/OpenAlex/Semantic Scholar.
- Rich parser comparison workflow across GROBID and S2ORC-style output beyond `rforge parse compare`.

## Recommended slice

Extend the S2ORC-style JSON reader with citation-span offsets linked to parsed passages.
