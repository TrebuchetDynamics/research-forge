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
- `rforge citations expand --source openalex` for reference/citation graph export, with shared depth/node/API-call/retry/resume/dry-run budget controls before live expansion.
- `rforge citations accessible-view` provides no-JS review views with graph summaries, filtered node tables, tabular edge lists, domain-topic rows, keyboard-navigation guidance, and exportable graph reports alongside interactive SVGs.
- OpenAlex source connector with mocked tests, including works, author/institution/concept entity searches, related-work discovery records, and disambiguation review queues for ambiguous people/institutions/concepts.
- Source refs stored in `PaperRecord` conversion.
- The forge workflow DAG includes discovery/import checkpoints with inputs, outputs, provenance actions, and restart-safe skips.
- The local project knowledge graph merges OpenAlex concepts with Zotero collections/tags, Semantic Scholar citation edges, parsed references, evidence, screening, analysis, and report claims for `rforge knowledge query`.
- Live manual searches have been used for ResearchForge-backed reports.
- Source-fusion identity resolution merges DOI/arXiv/PMID/PMCID/OpenAlex/Semantic Scholar/Crossref/Zotero IDs with explainable rules, conflict records, reversible merge/split decision logs, and `library identity-decision apply` support.
- The `/map` local web cockpit unifies citation graph, OpenAlex concepts, Zotero collections/tags, screening status, retrieval clusters/hits, and evidence coverage with filters, keyboard navigation, no-JS tables, and `/map/snapshot.json` audit exports.
- Cross-tool benchmarks report deterministic fixture metrics for discovery recall, dedupe precision, parser field accuracy, reference normalization, retrieval quality, screening effort savings, and report/package reproducibility.
- The `/dedupe` workbench shows identity clusters, conflicting source fields, Zotero collection/tag context, citation-key preservation, merge/split history, and reversible identity decisions.

Missing features:

- Rich cursor-based multi-page import workflow beyond fixed page-count import.
- Rich concepts/domain-map import beyond source metadata.
- Rich CLI UX for institution/author/concept search beyond JSON disambiguation queues.
- Higher-level works filter presets beyond generic `--filter`.
- Opt-in live connector smoke test.

## Recommended slice

Build reviewer workflows over OpenAlex disambiguation queues and domain-map import decisions.

Acceptance target:

```sh
rforge --project <project> search import --source openalex --query <query> --pages 3
```
