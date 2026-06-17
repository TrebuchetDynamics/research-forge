# Qdrant study note

- Repository/ecosystem: `qdrant/qdrant` and Go client ecosystem.
- Area: vector retrieval, semantic search, nearest-neighbor indexes.
- Disposition: `adapter-only`.
- License/action constraint: call as optional local/remote service; keep embeddings and private text local unless user explicitly configures otherwise.

## Why it matters

ResearchForge needs semantic retrieval across abstracts, passages, evidence, and reports. Qdrant is a strong local/production vector store candidate.

## Patterns to learn

- Collections need explicit vector size/model metadata.
- Embedding model versions must be locked for reproducibility.
- Payload schemas should preserve paper IDs, passage IDs, source refs, and project scope.
- Vector backends must be optional; local-first workflows should still work without them.

## ResearchForge status

Implemented nearby capabilities:

- Optional Qdrant service check.
- Adapter seam/backlog exists.
- Retrieval package supports local indexing scaffolds.
- Qdrant passage vector indexing/search adapter with mocked HTTP tests.
- CLI backend selection through `rforge index rebuild --backend qdrant` and `rforge retrieve --backend qdrant`.
- Deterministic local hash embedding scaffold for offline tests and reproducible smoke workflows, with `RFORGE_EMBEDDING_DIMENSIONS` dimension configuration recorded in `data/retrieval.lock.json`.
- Opt-in HTTP embedding provider via `RFORGE_EMBEDDING_URL`, `RFORGE_EMBEDDING_MODEL`, and mandatory `RFORGE_EMBEDDING_CONSENT=1`, with backend/model metadata recorded in `data/retrieval.lock.json`.
- Hybrid lexical + vector retrieval through `--backend hybrid`, combining SQLite FTS with Qdrant results using reciprocal-rank fusion and deterministic de-duplication; `rforge retrieve tune-hybrid` writes calibrated tuning files with backend weights, evaluation scores, selected configuration, and query-set checksums.
- Retrieval rebuilds write `data/retrieval.lock.json` with backend and deterministic embedding metadata.
- Retrieval benchmark report compares Qdrant fixture results against SQLite FTS, OpenSearch, and hybrid ranking with reproducibility/privacy notes.
- Embedding-provider compliance profiles document text egress, required consent/config, model version locks, dimensionality, retention policy, and redaction behavior before Qdrant/HTTP indexing runs.
- The `/map` local web cockpit combines concept maps, citation neighborhoods, screening priority, parser quality, retrieval hits, and evidence coverage with no-JS server rendering and `/map/snapshot.json` audit exports.
- Cross-tool benchmarks report deterministic fixture metrics for discovery recall, dedupe precision, parser field accuracy, reference normalization, retrieval quality, screening effort savings, and report/package reproducibility.

Missing features:

- Production embedding provider presets beyond the generic HTTP embedding contract.
- Learned rerankers beyond calibrated deterministic hybrid weighting.
- Opt-in Qdrant integration test.

## Recommended slice

Keep Qdrant benchmark fixtures and privacy notes current with embedding/payload changes, then add learned reranker experiments behind deterministic reports.

Acceptance target:

```sh
rforge index rebuild --backend qdrant
rforge retrieve --backend qdrant --query <query>
```
