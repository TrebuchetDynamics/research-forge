# OpenAlex study note

- Source/API: OpenAlex.
- Area: open scholarly metadata, works, authors, institutions, concepts, citation graph.
- Disposition: `adapter-only`.
- License/action constraint: use public API respectfully with documented request parameters, rate handling, and provenance.

## Why it matters

OpenAlex is a broad open scholarly graph and a strong default discovery source for ResearchForge.

## Patterns to learn

- Preserve source IDs and source-specific provenance.
- Support pagination/cursors for reproducible searches.
- Concepts and related works can support domain mapping.
- Open metadata still needs normalization and deduplication across sources.

## ResearchForge status

Implemented nearby capabilities:

- `rforge search --source openalex` with cursor/filter support and `rforge search import --source openalex --pages N` paginated library import.
- `rforge citations expand --source openalex` for reference/citation graph export.
- OpenAlex source connector with mocked tests, including works, author/institution entity searches, and related-work discovery records.
- Source refs stored in `PaperRecord` conversion.
- Live manual searches have been used for ResearchForge-backed reports.

Missing features:

- Rich cursor-based multi-page import workflow beyond fixed page-count import.
- Rich concepts/domain-map import beyond source metadata.
- Rich CLI UX for institution/author search beyond `rforge search --source openalex --entity authors|institutions`.
- Higher-level works filter presets beyond generic `--filter`.
- Opt-in live connector smoke test.

## Recommended slice

Add paginated search/import so a research project can freeze the exact OpenAlex search query, pages, cursor state, and imported records.

Acceptance target:

```sh
rforge --project <project> search import --source openalex --query <query> --pages 3
```
