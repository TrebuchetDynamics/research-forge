# ResearchForge Meta-analysis spine super-tool blueprint

This blueprint turns the OSS inventory into a first-priority ResearchForge super-tool path for meta-analysis authors. The first done artifact is the **Reproducible review package**: a portable package that proves a review can be audited and replayed, with report outputs as included artifacts rather than the sole product promise.

## Scope and ownership

ResearchForge composes studied OSS projects through provenance-aware adapters and product-owned workflow modules. Per ADR 0002, `pattern-reference` entries inform workflows and UX; `adapter-only` entries are called through explicit seams; no external code, schemas, fixtures, icons, or model assets are copied without separate license/provenance review.

The Meta-analysis spine is the canonical implementation order until the Reproducible review package is audit/replay safe:

1. question/protocol/source plan;
2. library import, identity resolution, and deduplication;
3. legal full-text acquisition;
4. parser arbitration and reference normalization;
5. retrieval, graph, and domain-map layer;
6. screening and review assistance;
7. evidence extraction and gap analysis;
8. statistical analysis and method comparison;
9. report traceability and package export;
10. Go + HTMX cockpit and one-command `rforge forge` wrapper.

Broader research-cockpit features build on this spine and must not bypass its review gates.

## Module map

| Module | Responsibility | OSS influence / disposition |
| --- | --- | --- |
| `forge` orchestration | Resumable Meta-analysis spine DAG, checkpoints, blocked gates, replay state | Product-owned; all OSS via modules below |
| question/protocol | Research question, inclusion/exclusion criteria, source plan, extraction schema seed | OpenAlex/Semantic Scholar/ASReview as `adapter-only`/`pattern-reference`; KeyBERT/SciSpaCy/LLM suggestions require reviewer approval |
| library/import | Reference-manager import/export, source records, citation keys, collections, tags, notes | Zotero/JabRef as `pattern-reference`; source APIs as `adapter-only` |
| identity/dedupe | DOI/arXiv/PMID/PMCID/OpenAlex/Semantic Scholar/Crossref/Zotero/ADS identity clusters, reversible merge/split | revtools as `pattern-reference`; source APIs as `adapter-only` |
| acquisition | OA/license-aware document candidate queue and document assets | Unpaywall/DOAJ/CORE/PubMed/Europe PMC/arXiv as `adapter-only` |
| parsing | Parser runs, parser manifests, parsed documents, passage IDs, raw output refs | GROBID/S2ORC/CERMINE/Anystyle as `adapter-only`; PaperMage/Science Parse as `pattern-reference` until escalated |
| parser arbitration | Field-level parser comparison, confidence/warnings, reviewer decisions | Product-owned comparison layer over parser adapters |
| reference normalization | Raw reference strings, parsed candidates, external-source matches, ambiguity queues | Crossref/OpenAlex/Semantic Scholar/NASA ADS as `adapter-only`; Anystyle/GROBID/S2ORC as parser inputs |
| retrieval/index | SQLite FTS/OpenSearch/Qdrant/hybrid indexes and query evaluation | OpenSearch/Qdrant as `adapter-only`; SentenceTransformers provider registry as `adapter-only` |
| domain map | Citation graph, topic clusters, concepts, accessible graph tables | OpenAlex/Semantic Scholar as `adapter-only`; BERTopic as `pattern-reference` |
| screening | Queues, reviewer assignments, conflicts, active-learning ranking, stopping diagnostics | ASReview/revtools as `pattern-reference` |
| evidence | Evidence schema, source support, extraction suggestions, risk-of-bias suggestions, gap analysis | RobotReviewer/SciSpaCy/PaperMage as `pattern-reference`/`adapter-only` suggestions only |
| analysis | Effect sizes, metafor scripts/results, sensitivity/bias/heterogeneity, secondary-engine comparison | metafor as `adapter-only`; PyMARE as future `adapter-only` |
| report/package | Claim traceability, report outputs, redaction, manifests, checksums, replay/audit commands | Product-owned package format; all module artifacts included by reference |
| web cockpit | Go + HTMX review/control surface over shared services | Product-owned; no core scientific logic in templates/JS |
| OSS governance | Inventory, capability registry, policy/drift/refresh reports | Product-owned governance over all entries |

## CLI command families

| Command family | Purpose | Required validation |
| --- | --- | --- |
| `rforge forge` | Guided Meta-analysis spine workflow and resume/replay entry point | End-to-end fake-backed CLI test; checkpoint/restart tests |
| `rforge protocol ...` | Question, criteria, source plan, extraction schema seed | Golden plan output; provenance events; no auto-accepted suggestions |
| `rforge source plan|smoke|capabilities` | Connector capability and API drift management | Mocked HTTP tests; opt-in live smoke only |
| `rforge import|export|library|duplicate` | Reference-manager import, source records, identity/dedupe decisions | Round-trip tests; reversible merge/split tests |
| `rforge oa|pdf|documents` | Legal full-text candidates and document assets | OA/license guard tests; path/privacy tests |
| `rforge parse ...` | Parser adapters, parser manifests, parser comparison/arbitration | Fixture parser tests; fake adapters before live parsers |
| `rforge references ...` | Parsed-reference normalization and adjudication | Ambiguous queue tests; source-match confidence tests |
| `rforge index|retrieve` | FTS/vector/hybrid retrieval and benchmark configs | Deterministic ranking fixtures; backend fake tests |
| `rforge graph|citations|topics` | Citation graph, domain maps, accessible exports | JSON/SVG/table golden tests |
| `rforge screen ...` | Screening queues, active learning, conflicts, stopping diagnostics | Multi-reviewer e2e; reproducible ranking fixtures |
| `rforge extract ...` | Evidence extraction and reviewer-gated suggestions | Source-support enforcement tests |
| `rforge analysis ...` | Effect sizes, metafor/PyMARE adapters, sensitivity/bias outputs | Known-result fixtures; opt-in real-engine tests |
| `rforge report ...` | Claim traceability and report outputs | Claim-support audit; reproducible report tests |
| `rforge package create|audit|replay|restore` | Reproducible review package lifecycle | Package replay/restore e2e; checksum/redaction tests |
| `rforge oss ...` | OSS inventory governance and roadmap coverage | Inventory check/drift/policy/report tests |

## Go + HTMX cockpit pages

The cockpit reviews and gates shared-service workflow state. Every action must show its CLI equivalent.

| Page / route family | Purpose | No-JS fallback |
| --- | --- | --- |
| `/forge` | Meta-analysis spine home timeline, blocked gates, next safe actions | Static timeline + CLI command list |
| `/protocol` | Question, criteria, source plan, extraction schema seed | Markdown/HTML plan preview |
| `/sources` | Connector capability, credentials/redaction status, dry-run estimates, live-smoke history | Connector table and CLI plan commands |
| `/library`, `/dedupe` | Source records, identity clusters, reversible merge/split, citation-key/collection context | Sortable tables and forms |
| `/acquisition` | OA/license queue, local path/privacy flags, approval gates | Candidate table with explicit approve/skip commands |
| `/parsing`, `/references` | Parser runs, field comparison, arbitration, parsed-reference adjudication | Field comparison tables |
| `/retrieve`, `/graph`, `/topics` | Retrieval tuning, citation/domain graph, accessible graph review | Passage tables, edge lists, node tables |
| `/screening` | Active-learning queue, conflicts, uncertain records, stopping diagnostics | Queue tables and CSV/JSON exports |
| `/evidence` | Evidence grid, support links, correction history, gap analysis | Evidence/support tables |
| `/analysis` | Prepared inputs, model settings, metafor scripts, sensitivity/bias outputs | Analysis summary tables + artifact links |
| `/report` | Claim traceability and final export blockers | Claim/support matrix |
| `/package` | Package preview, redaction, checksums, audit/replay status | Manifest/checksum/redaction tables |
| `/oss` | Inventory, capability registry, policy/drift/roadmap reports | Existing OSS dashboard tables |

## Storage files and tables

ResearchForge remains SQLite-first and local-first. File artifacts are kept in project directories when human-readable/auditable; relational state belongs in SQLite where query/review workflows need joins.

| Area | Storage |
| --- | --- |
| Workflow state | `data/forge-state.json`, `data/jobs/`, provenance log, SQLite workflow tables |
| Project config | `rforge.project.toml`, `rforge.lock.json`, `.researchforge` when repo-embedded |
| Source plans | `data/source-plans/*.json`, source cache refs, connector capability registry |
| Library/identity | SQLite library tables, `data/library.*`, identity cluster decision logs |
| Full text | `documents/`, document asset metadata, OA/license acquisition queue |
| Parsing | `parsed/`, `data/parser-manifests/*.json`, parser arbitration decisions |
| References | parsed-reference records, adjudication queue, normalization match reports |
| Retrieval | local FTS tables, `data/retrieval.lock.json`, vector backend config, benchmark result files |
| Graph/topics | `data/citation-graph.json`, topic/domain-map artifacts, accessible table exports |
| Screening | screening event log/tables, active-learning run files, audit bundle exports |
| Evidence | evidence records, extraction schema files, suggestion/correction history |
| Analysis | `analysis/`, input snapshots, scripts, outputs, warnings, plot artifacts, checksums |
| Reports | `reports/`, claim traceability matrix, report audit output |
| Package | `.rforge-package/manifest.json`, checksum manifest, redaction report, replay script/audit report |
| OSS governance | `opensource/inventory/manifest.json`, inventory notes, generated inventory reports |

## Provenance events

Each event records actor, action, target, inputs, outputs, warnings, tool versions, and source refs where applicable.

Required event families:

- `forge.state.transition`
- `protocol.plan.created`, `protocol.plan.approved`
- `source.query.planned`, `source.query.executed`, `source.live_smoke.recorded`
- `library.imported`, `identity.cluster.created`, `identity.merge.approved`, `identity.split.approved`
- `document.candidate.found`, `document.acquisition.approved`, `document.asset.created`
- `parser.run.completed`, `parser.arbitration.decided`, `reference.normalized`, `reference.review.decided`
- `retrieval.index.built`, `retrieval.query.executed`, `retrieval.benchmark.completed`
- `screening.decision.recorded`, `screening.rank.generated`, `screening.conflict.adjudicated`
- `evidence.suggestion.created`, `evidence.item.accepted`, `evidence.item.corrected`, `evidence.gap.reported`
- `analysis.input.prepared`, `analysis.run.completed`, `analysis.method_compared`
- `report.build.completed`, `report.claim.audit.completed`
- `package.created`, `package.audit.completed`, `package.replay.completed`
- `oss.inventory.checked`, `oss.policy.checked`, `oss.roadmap.generated`

## Validation gates

| Gate | Required checks |
| --- | --- |
| Normal local gate | `make fmt-check`, `go mod tidy -diff`, `go test ./...`, `go vet ./...`, `make todo-completion-audit`, inventory check, `git diff --check` |
| Source gate | mocked HTTP fixtures, no live network in normal tests, opt-in live smoke documented |
| Legal acquisition gate | OA/license status present, local-path/privacy redaction checked, approval recorded before download/export |
| Parser gate | parser manifest exists, raw output checksum recorded, arbitration/conflict state explicit |
| Screening gate | reviewer attribution, conflict/uncertain handling, active-learning run reproducibility |
| Evidence gate | accepted evidence has exact source support and correction history |
| Analysis gate | input snapshot, model settings, engine versions, warnings, outputs, checksums |
| Report gate | every claim/table/figure traceable to accepted evidence or marked blocked |
| Package gate | manifest, lockfiles, checksums, redaction report, audit report, replay command, restore test |
| Dashboard gate | handler tests, CLI parity for actions, no-JS fallback, Playwright path for critical workflows |

## Adapter disposition summary

| Disposition | Entries |
| --- | --- |
| `adapter-only` | GROBID, metafor, Semantic Scholar, OpenAlex, Qdrant, OpenSearch, s2orc-doc2json, Anystyle, CERMINE, PyMARE, SentenceTransformers, SciSpaCy, NASA ADS, DOAJ/CORE |
| `pattern-reference` | Zotero, ASReview, PaperMage, Science Parse, JabRef, RobotReviewer, revtools, BERTopic, KeyBERT |

Any escalation to `integrate` or use of external schemas/fixtures/assets requires a separate license/security/maintenance review and likely an ADR.
