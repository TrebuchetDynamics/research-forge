# Science Parse study note

- Repository/ecosystem: `allenai/science-parse`.
- Area: scholarly PDF metadata and bibliography parsing.
- Disposition: `pattern-reference`.
- License/action constraint: historical/reference study only unless maintenance and licensing are revalidated.

## Why it matters

Science Parse is useful as historical context for PDF metadata/reference extraction tradeoffs, but it may not be the best active dependency. ResearchForge should document fallback candidates and avoid stale-parser lock-in.

## Patterns to learn

- Parser maintenance status matters as much as extraction quality.
- Bibliography extraction should preserve raw strings and normalization state.
- Parser outputs need quality checks before entering evidence/report workflows.

## ResearchForge status

Implemented nearby capabilities:

- GROBID adapter.
- Parsed reference model.
- OSS inventory governance for adapter decisions.
- Parser comparison reports include Science Parse as stale-reference fallback metadata when present in parsed outputs.

Missing features:

- Rich parser maintenance/risk scoring beyond static parser-candidate metadata.
- Comparative parser benchmark fixtures.
- Stale fallback policy.

## Recommended slice

Extend parser candidate scoring with benchmark fixtures and policy gates for stale fallback acceptance.
