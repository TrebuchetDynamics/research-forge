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

Missing features:

- Real Qdrant indexing/query adapter.
- Embedding model configuration and lockfile capture.
- Hybrid lexical + vector retrieval.
- Passage-level vector indexing.
- Opt-in Qdrant integration test.

## Recommended slice

Add a Qdrant adapter interface implementation behind explicit configuration while keeping local tests on a fake adapter.

Acceptance target:

```sh
rforge index rebuild --backend qdrant
rforge retrieve --backend qdrant --query <query>
```
