# OpenSearch study note

- Repository/ecosystem: OpenSearch.
- Area: lexical/full-text search, filtering, aggregations.
- Disposition: `adapter-only`.
- License/action constraint: optional external service; record endpoint/config without secrets; avoid making OpenSearch mandatory for local-first workflows.

## Why it matters

ResearchForge needs robust keyword search over metadata, parsed sections, passages, evidence, and reports. OpenSearch is a mature candidate for heavier projects.

## Patterns to learn

- Index mappings should be versioned.
- Bulk indexing needs retry and partial-failure reporting.
- Search results must cite exact source records/passages.
- Local workflows need a fallback index when OpenSearch is unavailable.

## ResearchForge status

Implemented nearby capabilities:

- Optional OpenSearch service check.
- Adapter seam/backlog exists.
- Local retrieval/index package exists.

Missing features:

- Real OpenSearch indexing/query adapter.
- Mapping version recorded in lockfile.
- Bulk indexing command with partial-failure provenance.
- Highlighted passage search results.
- Opt-in OpenSearch integration test.

## Recommended slice

Add an adapter contract and fake-backed tests first, then a gated real OpenSearch integration behind `RFORGE_OPENSEARCH_URL`.

Acceptance target:

```sh
rforge index rebuild --backend opensearch
rforge retrieve --backend opensearch --query <query>
```
