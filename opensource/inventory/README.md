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
| JabRef | BibTeX/BibLaTeX reference management | `pattern-reference` | [jabref.md](./jabref.md) |
| RobotReviewer | Evidence extraction / risk of bias | `pattern-reference` | [robotreviewer.md](./robotreviewer.md) |
| revtools | Screening / dedup clustering | `pattern-reference` | [revtools.md](./revtools.md) |
| PyMARE | Secondary meta-analysis engine | `adapter-only` | [pymare.md](./pymare.md) |
| SentenceTransformers | Semantic embeddings | `adapter-only` | [sentence-transformers.md](./sentence-transformers.md) |
| BERTopic | Topic modeling / domain maps | `pattern-reference` | [bertopic.md](./bertopic.md) |
| SciSpaCy | Scientific entity extraction | `adapter-only` | [scispacy.md](./scispacy.md) |
| KeyBERT | Keyword extraction / query expansion | `pattern-reference` | [keybert.md](./keybert.md) |
| NASA ADS | Physics/astronomy source API | `adapter-only` | [nasa-ads.md](./nasa-ads.md) |
| DOAJ / CORE | Open-access discovery | `adapter-only` | [doaj-core.md](./doaj-core.md) |

## Feature coverage map

| ResearchForge super-tool area | Inventory examples | Features to study/combine |
| --- | --- | --- |
| Reference/library management | Zotero, JabRef | collections, tags, notes, citation keys, BibTeX/BibLaTeX cleanup, linked-file privacy, annotation import/export |
| Scholarly discovery graph | OpenAlex, Semantic Scholar, NASA ADS | source-specific IDs, author/institution/concept/bibcode search, citation/reference expansion, cursor/resume state, rate-limit provenance |
| Open-access acquisition | Unpaywall-backed implementation, DOAJ / CORE | OA/license metadata, full-text candidates, acquisition approval queues, shareability flags, source URL provenance |
| Full-text parsing | GROBID, s2orc-doc2json, PaperMage, CERMINE, Science Parse | parser arbitration, stable offsets, section/passages/references, parser risk scoring, raw-output manifests |
| Reference parsing/normalization | Anystyle, GROBID, S2ORC-style output | raw reference preservation, DOI/title candidates, confidence, reviewer adjudication, Crossref/OpenAlex/Semantic Scholar normalization |
| Screening and review | ASReview, revtools | active learning, uncertainty/exploration, cluster review, dedupe visualization, reviewer progress, stopping diagnostics, audit exports |
| Evidence/risk assessment | RobotReviewer, SciSpaCy, PaperMage | citation-locked extraction suggestions, risk-of-bias schemas, entity extraction, source span links, reviewer acceptance state |
| Retrieval/semantic search | OpenSearch, Qdrant, SentenceTransformers, KeyBERT | lexical/vector/hybrid ranking, embedding privacy, keyword/query expansion, benchmarked retrieval configurations |
| Domain maps | BERTopic, OpenAlex concepts, Semantic Scholar graphs | topic clusters, representative passages, concept maps, citation neighborhoods, reviewer-edited labels |
| Statistics/meta-analysis | metafor, PyMARE | engine comparison, model settings, heterogeneity, sensitivity/influence diagnostics, publication-bias checks, artifact manifests |
| Reproducible super-tool UX | All entries | `rforge forge` DAG, HTMX cockpit screens, connector capability registry, provenance journal, reproducible-review package |

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

1. Build the `rforge forge` super-tool path: a resumable, provenance-first workflow that combines source discovery, reference-manager import, dedupe, legal full-text acquisition, parser arbitration, retrieval, screening, extraction, statistics, and report packaging.
2. Deepen reference-manager interoperability: Zotero/JabRef collection/group mapping, citation-key fidelity, annotation import/export, BibTeX cleanup diffs, and linked-file privacy gates.
3. Add parser arbitration and parsed-reference normalization across GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse, and Anystyle with reviewer-adjudicated conflicts.
4. Expand source coverage: NASA ADS for physics/astronomy and DOAJ/CORE for OA discovery; PubMed / Europe PMC / PMC full-text workflows now cover ID linking, OA license capture, structured JATS import, supplementary discovery, and opt-in drift smoke planning.
5. Combine ASReview/revtools/RobotReviewer patterns into auditable review assistance: active-learning queues, cluster review, risk-of-bias/evidence suggestions, stopping diagnostics, and exportable audit bundles.
6. Turn retrieval/NLP tools into governed adapters: OpenSearch/Qdrant/SentenceTransformers/KeyBERT/SciSpaCy/BERTopic with privacy profiles, model locks, benchmarks, and citation-linked suggestions.
7. Add cross-engine statistical validation: metafor remains primary, with PyMARE-style secondary comparison reports and publication-ready artifact manifests.
