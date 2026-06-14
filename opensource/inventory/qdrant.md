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
- Opt-in HTTP embedding provider via `RFORGE_EMBEDDING_URL` and `RFORGE_EMBEDDING_MODEL`, with backend/model metadata recorded in `data/retrieval.lock.json`.
- Hybrid lexical + vector retrieval through `--backend hybrid`, combining SQLite FTS with Qdrant results using reciprocal-rank fusion and deterministic de-duplication.
- Retrieval rebuilds write `data/retrieval.lock.json` with backend and deterministic embedding metadata.

Missing features:

- Production embedding provider presets beyond the generic HTTP embedding contract.
- Rich hybrid ranking beyond reciprocal-rank fusion, such as calibrated backend weights and learned rerankers.
- Opt-in Qdrant integration test.

## Recommended slice

Add configurable embedding provider/model selection and richer hybrid ranking over SQLite/OpenSearch plus Qdrant results.

Acceptance target:

```sh
rforge index rebuild --backend qdrant
rforge retrieve --backend qdrant --query <query>
```
