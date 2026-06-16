# ResearchForge OSS inventory

This directory stores committed, source-controlled study notes for open-source projects and public scholarly infrastructure that ResearchForge learns from. The machine-readable index is [`manifest.json`](./manifest.json).

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
| s2orc-doc2json | Full-text parser interchange | `adapter-only` | [s2orc-doc2json.md](./s2orc-doc2json.md) |
| PaperMage | Layered paper document model | `pattern-reference` | [papermage.md](./papermage.md) |
| Anystyle | Reference parser | `adapter-only` | [anystyle.md](./anystyle.md) |
| CERMINE | PDF parser fallback | `adapter-only` | [cermine.md](./cermine.md) |
| Science Parse | Historical PDF parser fallback | `pattern-reference` | [science-parse.md](./science-parse.md) |

## Validation

Run:

```sh
rforge oss inventory-check opensource/inventory/manifest.json
rforge oss inventory-refresh opensource/inventory/manifest.json --source github
rforge oss inventory-policy opensource/inventory/manifest.json --stale-after 18mo
rforge oss inventory-drift opensource/inventory/manifest.json
rforge oss inventory-report opensource/inventory/manifest.json --area scholarly-graph-source
```

The verifier checks that every manifest entry has governance metadata (`area`, `disposition`, `licensePolicy`, `risk`, `nextSlice`) and that its Markdown note exists inside this directory. The refresh command updates GitHub-backed entries with stars, forks, license, archive status, and push timestamp. The policy command flags archived repositories, stale `pushedAt` timestamps, missing/unknown licenses, and GPL/AGPL/LGPL entries whose disposition is not `adapter-only` or `pattern-reference`. The drift command compares manifest metadata with note headings and optional note fields (`Area`, `Disposition`, `Repository`, `URL`, `License policy`, `Next slice`) and flags unreferenced note files. The report command renders a deterministic Markdown ecosystem report with tool, area, disposition, refreshed metadata, risk, next slice, and note columns.

## Current highest-priority gaps from this inventory

1. Rich Zotero collections/annotations beyond CSL-JSON/RDF metadata.
2. Parser comparison report and parsed-reference normalization across GROBID, S2ORC-style JSON, and Anystyle-style reference parsing.
3. Richer PubMed / Europe PMC full-text workflows; PubMed and Europe PMC search plus opt-in biomedical live smoke docs/targets are implemented.
4. Additional opt-in live connector smoke coverage for API drift beyond current source smoke tests.
5. Richer ASReview-style ranking loop: model-based prioritization, reviewer feedback metrics, and stopping diagnostics.
