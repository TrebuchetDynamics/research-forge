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
- `rforge citations expand --source semantic-scholar --paper <id> --direction references|citations|both --out <file>`.
- Deterministic mocked HTTP tests for search and graph expansion.

Implemented nearby capabilities:

- `--import-library` on citation expansion imports discovered graph records into the project library.

Missing features:

- Store graph expansion provenance inside a ResearchForge project.
- Recursive graph expansion with depth limits and deduplication.
- Rate-limit/backoff policy specific to Semantic Scholar quotas.
- Live opt-in smoke test with API-key support.
- UI artifact view for generated graph JSON.

## Completed slice

`--import-library` was added to citation expansion so discovered references/citing papers become normalized `PaperRecord`s in the project library while graph JSON remains exported.

Implemented command:

```sh
rforge --project <project> citations expand --source semantic-scholar --paper <id> --direction both --out graph.json --import-library
```

## Recommended next slice

Persist graph expansion provenance events and support recursive expansion with depth/record limits.
