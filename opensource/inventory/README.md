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
| Scholarly discovery graph | OpenAlex, Semantic Scholar, NASA ADS | source-specific IDs, author/institution/concept/bibcode search, citation/reference expansion, cursor/resume state, rate-limit provenance, source-API drift dashboards |
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
rforge oss inventory-roadmap opensource/inventory/manifest.json --todo TODO.md
rforge oss inventory-report opensource/inventory/manifest.json --area scholarly-graph-source
```

The verifier checks that every manifest entry has governance metadata (`area`, `disposition`, `licensePolicy`, `risk`, `nextSlice`) and that its Markdown note exists inside this directory. The refresh command updates GitHub-backed entries with stars, forks, archive status, and push timestamp; a manually reviewed SPDX value is preserved when GitHub reports no license assertion. The policy command flags archived or stale repositories, missing/unknown licenses on repository-backed entries, and GPL/AGPL/LGPL entries whose disposition is not `adapter-only` or `pattern-reference`. API-only services remain governed by `licensePolicy`. The drift command compares manifest metadata with note headings and optional note fields (`Area`, `Disposition`, `Repository`, `URL`, `License policy`, `Next slice`) and flags unreferenced note files. The roadmap command accepts direct and grouped note references in `TODO.md`, reports uncovered inventory work, and does not re-suggest covered tasks. The report command renders a deterministic Markdown ecosystem report with tool, area, disposition, refreshed metadata, risk, next slice, and note columns.

## Current maintenance priorities

Snapshot refreshed 2026-07-28:

1. Reassess stale upstreams before enabling or expanding adapters: s2orc-doc2json, PaperMage, CERMINE, Science Parse, and RobotReviewer currently exceed the 18-month activity threshold.
2. Keep stale or copyleft projects behind `adapter-only` or `pattern-reference` boundaries; do not vendor their code, models, or fixtures without separate review.
3. Preserve manually verified SPDX metadata for repositories where GitHub returns `NOASSERTION`, and rerun refresh, policy, drift, and roadmap checks together.
4. Treat `TODO.md` as implementation truth. Update `nextSlice` only when a genuinely new, source-backed gap is identified instead of re-listing completed super-tool work.
