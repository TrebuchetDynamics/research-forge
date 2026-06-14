# ResearchForge OSS inventory

This directory stores committed, source-controlled study notes for open-source projects and public scholarly infrastructure that ResearchForge learns from.

Policy:

- Default disposition is `pattern-reference` unless a note explicitly escalates it.
- Do not copy external source code, schemas, fixtures, icons, or assets into ResearchForge from these projects without a separate license/provenance review.
- Local clones belong under `opensource/clones/` and stay gitignored.
- Notes here capture workflow ideas, integration risks, and concrete ResearchForge gaps.

## Initial top-tool inventory

| Tool/source | Area | Disposition | Study note |
| --- | --- | --- | --- |
| Zotero | Reference management | `pattern-reference` | [zotero.md](./zotero.md) |
| ASReview | Screening / active learning | `pattern-reference` | [asreview.md](./asreview.md) |
| GROBID | PDF/full-text parsing | `adapter-only` | [grobid.md](./grobid.md) |
| metafor | Meta-analysis/statistics | `adapter-only` | [metafor.md](./metafor.md) |
| Semantic Scholar | Scholarly graph/source API | `adapter-only` | [semantic-scholar.md](./semantic-scholar.md) |
| OpenAlex | Scholarly graph/source API | `adapter-only` | [openalex.md](./openalex.md) |
| Qdrant | Vector retrieval | `adapter-only` | [qdrant.md](./qdrant.md) |
| OpenSearch | Full-text retrieval | `adapter-only` | [opensearch.md](./opensearch.md) |

## Current highest-priority gaps from this inventory

1. Zotero-compatible CSL-JSON / Better BibTeX import-export.
2. Semantic Scholar citation expansion import into the local library with provenance.
3. PubMed / Europe PMC connector implementation.
4. Opt-in live connector smoke tests for API drift.
5. Richer ASReview-style ranking loop: model-based prioritization, reviewer feedback metrics, and stopping diagnostics.
