# Semantic Scholar study note

- Source/API: Semantic Scholar Graph API / S2 ecosystem.
- Area: paper metadata, citation/reference graph, author/citation enrichment.
- Disposition: `adapter-only`.
- License/action constraint: obey API terms, rate limits, field restrictions, and API-key redaction; do not cache restricted payloads beyond documented policy.

## Why it matters

Semantic Scholar complements OpenAlex with practical citation/reference expansion and paper metadata useful for discovery and graph building.

## Patterns to learn

- Graph expansion should preserve direction: citing paper -> referenced paper.
- API-key handling must be optional and redacted.
- Request fields should be explicit and minimal.
- Raw request references should be recorded for provenance.

## ResearchForge status

Implemented nearby capabilities:

- `rforge search --source semantic-scholar`.
- Optional `RFORGE_SEMANTIC_SCHOLAR_API_KEY` sent as `x-api-key`.
- `ExpandCitationGraph` source adapter for references, citations, or both.
- `rforge citations expand --source semantic-scholar --paper <id> --direction references|citations|both --depth N --out <file>`.
- `--import-library` on citation expansion imports discovered graph records into the project library.
- `--depth N`, `--max-records N`, `--max-api-calls N`, `--retry-budget N`, `--resume-cursor <cursor>`, and `--dry-run` on citation expansion support recursive graph expansion with budget estimates, deduplication, bounded imports/exports, and resumability controls before live calls.
- Project-scoped `citations.expand` provenance events record source, seed paper, direction, limit, depth, output path, edge count, record count, import count, raw reference, and graph-expansion budget settings.
- `rforge citations report --graph <graph.json> --out <report.md>` generates a Markdown citation-graph summary with top cited/citing papers and co-citation/coupling counts.
- `rforge duplicate report --source semantic-scholar` filters duplicate-review candidates involving graph-imported Semantic Scholar records and shows left/right source provenance for merge UX.
- Quota/transient retry policy uses the shared source HTTP backoff, honors `Retry-After`, and can be tuned with `RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES`.
- Opt-in live smoke target `make semantic-scholar-live-smoke` supports `RFORGE_SEMANTIC_SCHOLAR_API_KEY`.
- Web artifacts view renders exported citation graphs as an accessible SVG preview.
- `rforge citations accessible-view` provides no-JS review views with graph summaries, filtered node tables, tabular edge lists, domain-topic rows, keyboard-navigation guidance, and exportable graph reports alongside interactive SVGs.
- Deterministic mocked HTTP tests for search, graph expansion, recursive expansion, library import, graph-import dedupe filtering, web visualization, and report generation.
- The forge workflow DAG includes discovery/import checkpoints with inputs, outputs, provenance actions, and restart-safe skips.
- The local project knowledge graph merges Semantic Scholar citation edges with Zotero collections/tags, OpenAlex concepts, parsed references, evidence, screening, analysis, and report claims for `rforge knowledge query`.

Missing features:

- Rich live smoke coverage beyond lightweight search.
- Rich interactive graph exploration beyond the current artifact SVG preview.

## Completed slice

`--import-library` was added to citation expansion so discovered references/citing papers become normalized `PaperRecord`s in the project library while graph JSON remains exported.

Implemented command:

```sh
rforge --project <project> citations expand --source semantic-scholar --paper <id> --direction both --depth 2 --out graph.json --import-library
```

## Recommended next slice

Extend interactive review of graph-expansion runs using the persisted budget estimate, resume cursor, and visited/frontier state.
