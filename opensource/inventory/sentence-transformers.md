# SentenceTransformers study note

- Repository/ecosystem: `UKPLab/sentence-transformers` and embedding model ecosystem.
- Area: semantic embeddings for abstracts, passages, evidence, and reports.
- Disposition: `adapter-only`.
- License/action constraint: call a configured local or remote embedding provider only after model license, text-egress, and retention policy are explicit.

## Why it matters

Semantic retrieval is central to the ResearchForge super-tool: finding passages, related work, evidence gaps, and citation neighborhoods. SentenceTransformers is a common model ecosystem, but embedding private text requires careful local-first defaults.

## Patterns to learn

- Embedding model name, version, dimension, pooling, and preprocessing must be locked.
- Payloads should preserve paper IDs, passage IDs, evidence IDs, and project scope.
- Remote embedding calls require explicit user consent and redaction policy.
- Model changes invalidate vector indexes and benchmark comparisons.

## ResearchForge status

Implemented nearby capabilities:

- Qdrant vector indexing/search adapter with mocked HTTP tests.
- Deterministic local hash embedding scaffold for offline tests.
- Generic HTTP embedding provider with metadata recorded in `data/retrieval.lock.json`.
- Hybrid retrieval via reciprocal-rank fusion.

Missing features:

- Embedding model registry with license/egress metadata.
- Provider presets for common local and remote embedding services.
- Automatic vector-index invalidation when model metadata changes.
- Retrieval benchmarks for embedding model selection.

## Recommended next slice

Add an embedding provider registry that records model ID, dimensions, license notes, local/remote egress behavior, redaction policy, and compatibility with Qdrant collections.
