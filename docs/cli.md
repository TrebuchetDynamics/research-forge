# CLI reference

Core commands:

- `rforge capabilities --json [--min-rforge-version <version>] [--require <family>=<contract-version>]...` reports build metadata and command contracts, and optionally checks engine compatibility for skills, plugins, and adapters (see [compatibility checks](#capability-report-and-compatibility-checks))

- `rforge automation policy [--action <action>]` prints the hybrid agent/human decision policy: LLM agents may decide reversible assists such as batch retrieval/dedupe, screening queues, extraction suggestions, analysis preparation, and claim-trace audits, while humans must approve inclusion/exclusion, full-text acquisition, extraction acceptance, analysis method selection, final claims, and package export
- `rforge [--project <path>] mcp serve` serves the six read/run tools over stdio using the official Go MCP SDK (see [MCP stdio server](#mcp-stdio-server)); stdout is exclusively JSON-RPC
- `rforge project create [path] --title <title>`
- `rforge project inspect <path>`
- `rforge forge init --project <path> --question <text> [--sources openalex,semantic-scholar --tools grobid,qdrant]`, `rforge forge status --project <path>`, `rforge forge next --project <path>`, `rforge forge approve --project <path> --gate <gate> [--note <text>]`, `rforge forge reopen --project <path> --state <state> --reason <text>`, and `rforge forge run-dag --project <path> --question <text> [--max-steps N]` run the guided Meta-analysis spine workflow and resumable discovery→report DAG, persisting `data/forge-state.json`, source/tool choices, privacy/legal previews, `data/forge-workflow/*` checkpoints, transition provenance, next safe actions, and reviewer gates for irreversible scientific/data-sharing decisions
- `rforge search --source openalex|arxiv|crossref|semantic-scholar|europepmc|pubmed --query <query> [--category arxiv-category] [--filter source-filter] [--preset systematic-review|open-access-review|recent-domain-map] [--from-year YYYY] [--to-year YYYY] [--type article] [--open-access true|false] [--concept C41008148] [--entity authors|institutions|concepts]`; OpenAlex entity searches include disambiguation review queues when people, institutions, or concepts are ambiguous
- `rforge search batch --queries <queries.txt> --sources all|scholarly-fast|biomedical|preprints|datasets|open|oa|openalex,crossref,... --out <dir> [--query <query>] [--limit N] [--depth quick|standard|comprehensive] [--max-requests N] [--dedupe doi,title] [--continue-on-error] [--stats] [--resume]` runs bounded multi-query/multi-source sweeps (see [search budgets](#search-budgets-and-stop-reasons)), writes raw per-source files plus `results.jsonl`, DOI/title `results-deduped.jsonl`, `results.md`, `manifest.json`, optional `search-stats.txt`, `failures.jsonl`, and an incremental `batch-state.json` checkpoint so agents do not need shell loops for resumable literature scouting. The checkpoint records every completed (query, source) pair after each network call; an interrupted sweep (SIGINT/SIGTERM/timeout — exit 130/143) keeps its partial results in the topic dir, and re-running the same command with `--resume` skips completed pairs, retries failed ones, and completes never-run ones. `rforge search stats --dir <dir>` reports checkpoint progress (`completed/failed/pending`) and prints the exact resume command when work remains
- `rforge search import --source openalex --query <query> --pages N [--limit N] [--filter source-filter] [--preset systematic-review|open-access-review|recent-domain-map] [--from-year YYYY] [--to-year YYYY] [--type article] [--open-access true|false] [--concept C41008148] [--resume-state state.json]`
- `rforge search related --source openalex --paper <work-id> [--limit N]`
- `rforge citations expand --source semantic-scholar|openalex|crossref --paper <id> --direction references|citations|both --depth N [--max-records N --max-api-calls N --retry-budget N --resume-cursor <cursor> --dry-run] --out <file> [--run-state <semantic-run.json>] [--import-library]` expands citation graphs; dry-run emits a budget estimate before live expansion, and Semantic Scholar run-state files record quota/resume fields, API field restrictions, and graph budget controls
- `rforge citations import-bibliography (--parsed <parsed.json>|--parsed-dir <dir>) --out <graph.json> --report <report.json> [--evidence <evidence.json>]` imports parsed bibliography references as citation-graph edges across one or many parsed documents and links citation spans back to passages and accepted report evidence support
- `rforge citations domain-map --parsed-dir <dir> --out <domain-map.json> [--graph <graph.json> --label topic=label --history action:topic1,topic2:result:reviewer:reason --model <name>]` writes BERTopic-style domain-map artifacts with representative papers/passages, reviewer labels, topic merge/split history, model settings, and citation-graph links
- `rforge citations accessible-view --graph <graph.json> --out <view.md|view.json> [--domain-map <domain-map.json> --filter <text> --format markdown|json]` exports accessible/no-JS citation/domain graph views with graph summaries, filtered node tables, edge lists, keyboard navigation notes, and domain topic rows
- `rforge citations report --graph <graph.json> --out <report.md>`
- `rforge protocol compile --type pico|peco|spider|freeform --question <text> [framework flags]` drafts source-specific query plans, inclusion/exclusion criteria, extraction schema seeds, and reviewer prompts; all outputs require reviewer approval before use
- `rforge protocol plan-sources --type pico|peco|spider|freeform --question <text> [framework flags]` previews the Meta-analysis spine source plan for OpenAlex, Semantic Scholar, Crossref, arXiv, PubMed/Europe PMC, NASA ADS, DOAJ/CORE, Unpaywall, Zotero/JabRef, and local imports with dry-run estimates, auth/privacy warnings, and CLI equivalents
- `rforge protocol capabilities` prints the connector capability registry: supported entities, rate-limit policy, auth needs, live-smoke status, license/shareability policy, cacheability, and provenance fields
- `rforge protocol live-smoke-snapshot --output <project>/data/source-live-smoke-snapshots/latest.json [--connector <id> --status pass|fail|skipped --message <text> --fields <csv>]` stores API drift/live-smoke snapshots for connector dashboard alerts
- `rforge protocol live-smoke-dashboard --snapshot <latest.json> [--previous <prior.json> --provenance connector=path#event]` renders the source-API drift dashboard for OpenAlex, Semantic Scholar, PubMed, Europe PMC, Crossref, arXiv, and Unpaywall with response-shape changes and connector provenance links
- `rforge protocol suggest-expansions --question <text> --source-text <quoted passage> [--source-ref <id> --paper-id <id>]` drafts KeyBERT/SciSpaCy/LLM-style query-expansion suggestion records with source text links; suggestions cannot change a source plan until reviewer-approved
- `rforge oa lookup <doi>` resolves Unpaywall metadata for one DOI
- `rforge oa resolve-plan <doi>` prints a legal OA source plan across Unpaywall, OpenAlex OA locations, Europe PMC/PMC, preprint servers, DOAJ, CORE, Semantic Scholar/Crossref hints, Internet Archive/Open Library, and Software Heritage
- `rforge oa sources` lists supported legal OA resolver sources
- `rforge --project <path> oa candidates` compares legal full-text candidates from Unpaywall, DOAJ, CORE, PubMed/Europe PMC/PMC, arXiv, and local files; every candidate requires reviewer approval before acquisition
- `rforge --project <path> oa acquisition-queue` writes `data/legal-acquisition-queue.json` with OA/license status, source URL, expected local path, restricted/shareable flags, and reviewer approval gates; `rforge --project <path> oa acquisition-approve <id> --reviewer <name> --reason <text>` records explicit approval before download/archive inclusion
- `rforge --project <path> oa privacy-review [--report <file>]` writes `data/privacy-licensing-review.json` with gates for imported attachments, notes, annotations, local paths, copyrighted/missing-license PDFs, and shareable-report leaks; `rforge --project <path> oa privacy-approve --reviewer <name> --reason <text>` records explicit approval
- `rforge --project <path> fetch pdfs` downloads legal/OA PDFs by default, writes `documents/text/*.txt` and `documents/images/*`, and gitignores `documents/`; package/export approval gates remain separate
- `rforge library list`, `rforge library refresh-doi <doi>`, `rforge library refresh-crossref`, `rforge library import-crossref-refs <doi>`
- `rforge import json|csv|bibtex|ris|csl-json|zotero-rdf <file>`
- `rforge export json|csv|bibtex|ris|csl-json|zotero-rdf <file>`
- Zotero RDF and JabRef/BibTeX imports preserve reference-manager fidelity metadata in source refs: collections/groups, tags, notes, annotations, citation keys, BibTeX cleanup diffs, and linked-file privacy redaction checks
- `rforge --project <path> library reference-manager-matrix` round-trips BibTeX/RIS/CSL-JSON/Zotero RDF and reports the interchange fidelity matrix, including Better BibTeX citation keys, tags, notes, collections/groups, redacted attachments, and per-field preserved/lost counts
- `rforge --project <path> library jabref-quality` writes a JabRef-inspired BibTeX/BibLaTeX quality report for citation-key collisions, groups/saved searches, field cleanup diffs, linked-file privacy, and reviewer-approved normalization
- `rforge --project <path> library identity-resolve` reports source-fusion identity clusters with explainable match rules across DOI, arXiv, PMID, PMCID, OpenAlex, Semantic Scholar, Crossref, Zotero item keys, and NASA ADS bibcodes
- `rforge --project <path> library pmcid-pmid-links` reports normalized PMID/PMCID/DOI links for biomedical records
- `rforge --project <path> library identity-conflicts` records and reports identity-cluster conflict records; `rforge --project <path> library identity-decision record --action merge|split ...` appends reversible merge/split decisions to `data/identity-decisions.jsonl`; `library identity-decision apply --id <decision-id>` applies the recorded after-state to the local library
- `rforge duplicate report [--source <source>]|merge|split` (use `--source semantic-scholar` for graph-import dedupe review)
- `rforge oss search-plan --query <text> [--ecosystem all|go|python|js|rust|data]` prints a multi-provider open-source project discovery plan across GitHub, GitLab, Codeberg, SourceHut, Software Heritage, OpenSSF Scorecard, package registries, and research archives with ranking signals and human review gates
- `rforge oss inventory-check <manifest.json>`
- `rforge oss inventory-refresh <manifest.json> --source github [--base-url <url>]`
- `rforge oss inventory-policy <manifest.json> [--stale-after 18mo]`
- `rforge oss inventory-drift <manifest.json>`
- `rforge oss inventory-roadmap <manifest.json> --todo CHECKLIST.md` groups inventory `nextSlice` entries by area, reports notes/next slices missing TODO coverage, and suggests adapter/test/live-smoke implementation slices without marking them complete
- `rforge oss inventory-report <manifest.json> [--area <area>]`
- `rforge oss add|list|clone|license-check|note|scan|report|refresh`
- `rforge pdf fetch --doi <doi>`, `rforge pdf fetch-arxiv --paper <arxiv-id> --kind pdf|source --url <url>`
- `rforge --project <path> pdf import-biomedical --xml <jats-or-pmc.xml> --out <json>` imports structured biomedical full text with PMCID/PMID/DOI metadata, sections, and supplementary-file links; `rforge --project <path> pdf biomedical-drift-smoke-plan` prints PubMed/Europe PMC live-drift smoke expectations
- `rforge parse --paper <id> --parser grobid --pdf <file>` or `rforge parse --paper <id> --parser tex --tex <file>` or `rforge parse --paper <id> --parser s2orc --s2orc <file>` or `rforge parse --paper <id> --parser papermage --papermage <file>` writes parser-output manifests with source/version/command, input/output checksums, license constraints, shareability, and reviewer-gate fields; parsed documents include stable offsets, layered annotations, citation spans, parser confidence, and reconciliation-compatible fields
- `rforge parse manifest-policies` lists parser-output license/provenance policies for GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse-style metadata, and Anystyle outputs; the CERMINE adapter seam is represented by `CERMINEXMLParser` and the default fallback policy requires parser conflict review before accepting output
- `rforge parse references --paper <id> --parser anystyle --file <refs.txt> --out <refs.json>` (requires `RFORGE_ANYSTYLE_CMD`) writes an Anystyle parser-output manifest beside the parsed references
- `rforge parse compare --left <parsed.json> --right <parsed.json> --out <report.json>`
- `rforge parse quality --parsed <parsed.json> [--parsed <parsed.json>...] --out <report.json>` writes a reviewer-gated parser quality/confidence report across GROBID, S2ORC-style JSON, PaperMage, CERMINE, Science Parse-style metadata, and Anystyle reference output without auto-accepting conflicting fields
- `rforge parse maintenance-risk --out <report.json>` writes stale-parser maintenance/risk scores and comparative parser benchmark fixtures; historical fallbacks such as Science Parse require a maintenance-risk review gate before enablement
- `rforge parse arbitrate (--left <parsed.json> --right <parsed.json>|--parsed <parsed.json> --parsed <parsed.json>...) --out <report.json> [--accept <parser> --reason <text> --reviewer <name>]` scores GROBID/S2ORC-style/PaperMage/CERMINE/Science Parse/Anystyle parser output per field, compares raw text/offsets/warnings, routes conflicts to review, and records why one parser output was accepted, including field-level multi-parser reconciliation outputs
- `rforge parse normalize-refs --parsed <parsed.json> --source crossref|openalex|semantic-scholar|ads --out <report.json>` preserves raw reference strings, parser name/version/confidence, request/response provenance, match confidence, candidate counts, and reviewer-visible ambiguity queues; `NormalizeParsedReferencesAcrossConnectors` combines Crossref, OpenAlex, and Semantic Scholar normalization reports for multi-source review
- `rforge parse review-refs --parsed <parsed.json> --out <report.json> [--threshold 0.75]`
- `rforge parse adjudicate-ref --parsed <parsed.json> --index <n> --decision accept|correct|reject|defer --reviewer <name> --reason <text> [--title <text> --doi <doi> --raw <text> --log <jsonl>]` persists parsed-reference reviewer decisions to `data/reference-adjudications.jsonl` by default
- `rforge parse adjudicated-refs --parsed <parsed.json> [--log <jsonl> --out <report.json> --matches <normalization.json> --ambiguity-out <queue.json>]` exports adjudicated references with accepted/corrected/rejected/deferred/unreviewed counts and reviewer-persistent ambiguity queues linked to match/provenance metadata
- `rforge knowledge query --project <path> [--term <text>]` builds and queries the project knowledge graph by merging library collections/tags/concepts, citations, parsed references, evidence items, screening decisions, analysis runs, report claims, and provenance events
- `rforge knowledge path --project <path> --from <node-id> --to <node-id>` finds an offline shortest path through the local project knowledge graph
- `rforge --project <path> graph papers` downloads legal/OA PDFs when library metadata has PDF URLs, extracts text/images under gitignored `documents/`, then writes `data/knowledge-graph.json`, central-node/community/path report `data/knowledge-graph-report.md`, and browser-openable interactive force-directed `data/knowledge-graph.html`
- `rforge benchmark run --set <dir> [--replay|--record] [--split dev|held_out] [--out <report.json>]` scores the [research-quality benchmark contract](benchmark/spec.md). Default replay is network-free and accepts only checksum-locked sets; default split is `held_out`. Requests, retries, redirects and clock checkpoints must match recorded API bytes. The harness uses actual connectors, normalization and production dedupe, rejects reference-review leakage before dedupe, enforces publication-date caps, and reports recall, fixed-denominator precision@K, pairwise dedupe and recorded request/latency counts. Citations/tokens/cost are explicitly unmeasured, not perfect/zero. Initial qualifying adapters are OpenAlex and Crossref (provider `filter` date caps required); other sources fail closed with `date_cap_unenforceable` until they expose enforceable caps/dates. Replay latency is the captured workload, **not** a new implementation speed measurement.
  - `--record` is the explicit live-network opt-in, accepts only owner-approved sets with no existing snapshots, captures both splits, projects legal metadata (no abstracts/full text), and publishes the complete bundle before atomically locking the manifest. Failed captures retain the approved manifest and discard the incomplete bundle. Newly captured raw identities are unjudged; do not interpret unjudged pairs as correct merges. Never overwrite a locked set: changed inputs/labels require new approval and a new set ID/checksum. Live recording is not a historical database reconstruction.
  - Reports are canonical JSON plus LF on stdout, or at `--out` with a path message on stderr. A scorer-private, mode-0600 raw/membership sidecar is written as `<out>.audit.json`, or to a temporary path announced on stderr. **Never expose sets, review sheets, labels, audits or reports to agents/query planners.** Exit 0 means valid scoring (including poor recall, visible source errors or budget stops), 1 means invalid/checksum/leakage/replay failure, and 2 means usage error. Invalid runs have no usable aggregate. Gold-set precision does not assert that non-gold papers are irrelevant.
- `rforge benchmark cross-tool --out <report.json>` writes explicitly labeled **demo** constants and fixture descriptors; these illustrative numbers are not measured research-quality results and never feed `benchmark run`.
- `rforge index embedding-providers` lists embedding-provider registry entries for local and remote/provider-service models with model IDs, dimensions, license notes, text-egress/consent/redaction policy, vector-index invalidation, and retrieval benchmark compatibility; HTTP embedding/Qdrant runs require `RFORGE_EMBEDDING_CONSENT=1` and `RFORGE_EMBEDDING_MODEL`
- `rforge index rebuild [--backend sqlite|opensearch|qdrant|hybrid]` (writes `data/retrieval.lock.json`; OpenSearch also writes `data/opensearch.mapping.lock.json` and `data/opensearch.bulk-report.json`; Qdrant also writes `data/qdrant.vector.lock.json` and `data/qdrant.index-report.json`, honors `RFORGE_EMBEDDING_DIMENSIONS` or opt-in `RFORGE_EMBEDDING_URL`/`RFORGE_EMBEDDING_MODEL`, `RFORGE_QDRANT_PAYLOAD_PRIVACY=redacted-checksum`, and `RFORGE_QDRANT_INVALIDATE=1`), `rforge retrieve --query <query> [--backend sqlite|opensearch|qdrant|hybrid]` (OpenSearch results include highlighted passage snippets when returned by the backend)
- `rforge retrieve benchmark --out <report.json> [--k N]` compares SQLite FTS, OpenSearch, Qdrant vector search, and hybrid ranking on deterministic passage-query fixtures with query-set checksums plus reproducibility/privacy notes
- `rforge retrieve tune-hybrid --queries <queries.json> --lexical <results.json> --vector <results.json> --out <tuning.json> [--k N]` writes calibrated hybrid retrieval tuning files with lexical/vector/backend weights, deterministic RRF calibration method, evaluation scores, selected configuration, reproducibility notes, and query-set checksums
- `rforge research acquire-pdftotext --doi <doi> --pdf-url <url> --license <license> --oa-status <status> --out <parsed.json>` downloads a legal OA PDF and parses it with local `pdftotext`; `rforge research parse-pdftotext --paper <id> --pdf <file> --out <parsed.json> [--title <title>] [--chunk-size N]`, `rforge research screen-queue --out <queue.csv> [--markdown <queue.md>] [--library <library.json>] [--search-results <dir>]`, `rforge research leakage-audit (--parsed <parsed-dir> | --text <text-dir>) --out <audit.json> [--markdown <audit.md>] [--chunk-size N]`
- `rforge screen configure|decide|adjudicate|assign|panel|audit-bundle|queue|prioritize|model-prioritize|uncertainty|active-run|sensitivity|progress|recall|stopping|conflicts`, `rforge prisma counts`
- `rforge screen active-run --stage <stage> --method active-learning|model|uncertainty --out <run.json> [--target-recall 0.95]` persists ASReview-style active-learning runs with input/decision hashes, seed decisions, ranked output, reviewer progress, stopping diagnostics, and adjudication state
- `rforge screen sensitivity --stage <stage> --relevant <paper-id> --out <report.json> [--target-recall 0.95]` compares balanced exploration/exploitation policies with recall/effort simulations and sensitivity diagnostics
- `rforge screen assign --stage <stage> --reviewer <name> [--reviewer <name>] [--per-record N] --out <assignments.json>` assigns reviewers; `rforge screen panel --stage <stage> --out <panel.json>` exports conflict/adjudication panels; `rforge screen audit-bundle --stage <stage> --out <bundle.json> [--assignments <assignments.json> --active-run <run.json>]` exports ASReview-style audit bundles with uncertain queues, frozen dataset, seed labels, ranking iterations, reviewer actions, stopping diagnostics, deterministic random seeds, and model/version metadata
- `rforge extraction schema add`, `rforge extract add|suggest`, `rforge evidence audit`, `rforge evidence grid --out <grid.json> [--parsed <parsed.json>] [--analysis <run.json>]` exports an evidence extraction grid linking fields to support refs, parser offsets, PDF anchors, reviewer status/history, and downstream analysis inclusion; `rforge evidence gaps --out <report.json> [--question <text> --screened-in <paper-id> --parsed-paper <paper-id> --outcome <name> --comparator <name> --full-text <paper-id> --available-full-text <paper-id> --claims <queue.json> --analysis <run.json>]` reports missing outcomes/comparators, screened-in papers without evidence or parsed passages, unsupported claims, incomplete full text, research-question terms without accepted evidence, and analysis-input readiness
- `rforge evidence risk-bias-templates --out <templates.json>` exports RobotReviewer-inspired risk-of-bias schema templates; `rforge evidence risk-bias-suggest --paper <id> --support <ref=text> --out <queue.json> [--model <name> --version <version>]` creates automated judgment queues with exact support text, support refs, uncertainty, model/version metadata, and suggested status; `rforge evidence risk-bias-review --queue <queue.json> --id <suggestion-id> --decision accepted|rejected|corrected --reviewer <name> --out <queue.json> [--note <text>]` records reviewer decisions
- `rforge evidence entity-suggest --parsed <parsed.json> --out <queue.json> [--model <name> --version <version>]` creates SciSpaCy-inspired scientific entity suggestions with passage offsets, abbreviation expansions, entity-link candidates, confidence, and model provenance; `rforge evidence entity-review --queue <queue.json> --id <suggestion-id> --decision accepted|rejected|corrected --reviewer <name> --out <queue.json> [--note <text>]` records reviewer decisions
- `rforge evidence citation-suggest --paper <id> --support <ref=text> --out <queue.json> [--kind query_expansion|screening_rationale|extraction|report_prose --prompt <text> --model <name> --version <version>]` creates citation-locked LLM query-expansion/screening-rationale/extraction/report-prose suggestions where every suggested sentence is linked to exact support and stays `suggested`; `rforge evidence citation-review --queue <queue.json> --id <suggestion-id> --decision accepted|rejected|corrected --reviewer <name> --out <queue.json> [--note <text>]` is required before acceptance
- `rforge analysis prepare [--effect smd|log-odds-ratio|risk-ratio|mean-difference|risk-difference|fisher-z-correlation]|run|engine-compare|method-workbench|sensitivity|influence|moderators|subgroup|meta-regression|publication-bias|bayesian|export`; `analysis run` writes publication-ready styled plots, artifact manifests, settings, checksums, scripts, engine versions, warnings, and report embedding metadata; `analysis engine-compare <run-id> --out <report.json>` writes a PyMARE-style secondary-engine comparison with locks/parity/warnings/deltas; `analysis method-workbench <run-id> --category <name> --out <report.json> [--method <name> --select <method> --reviewer <name> --reason <text>]` compares parser/retrieval/screening/effect-size/publication-bias method choices side-by-side and can record the reviewer-selected method locked into the final report; `subgroup` and `meta-regression` accept `--from-evidence <field>` to derive moderator values from accepted evidence instead of manual per-paper entry; `publication-bias` supports `--method egger|begg`; `bayesian` supports `--method normal-approx|grid`
- `rforge report build --out <file> [--parsed <parsed.json>]` includes per-passage parser/version/source-offset provenance links; `rforge report trace --claims <queue.json> --analysis <run.json> --out <trace.json>` exports citation-to-evidence trace views; `rforge report claim-panel --trace <trace.json> --out <panel.json>` audits generated paragraph/table/figure trace strength; `rforge report final-export --in <report.md> --panel <panel.json> --out <final.md>` blocks export when the panel has unresolved or weak claims; `rforge report audit` checks package/report provenance
- `rforge forge source-fixture --project <path>` writes deterministic artificial-photosynthesis source plans, fake source payload refs, import receipts, and library records from `import_plan`, then advances to `dedupe_review`; `rforge forge reference-fixture --project <path>` adds Zotero/JabRef-style citation keys, collections/groups, tags, notes, annotations, redacted attachment basenames, approved privacy review, and reference-manager fidelity reports while staying at `dedupe_review`; `rforge forge acquisition-fixture --project <path>` writes approved OA/license acquisition metadata, export-safe Document asset metadata, and harmless open-access fixture text from `full_text_acquisition`, then advances to `parser_arbitration`; `rforge forge package-fixture --project <path> --out <dir>` completes the guided Meta-analysis spine from `package_export` to `done` by writing deterministic fixture artifacts, creating a package, and requiring package audit/replay to pass; `rforge package create --out <dir> [--created-by <name> --question <text>]` creates the Reproducible review package directory format with manifest, redaction report, copied project artifacts, and SHA-256 checksums; `rforge package fixture --out <dir>` creates a standalone offline artificial-photosynthesis fixture package for deterministic audit/replay validation; imported reference-manager attachments, notes, and annotations require an approved `oa privacy-review` before packaging/shareable copying; `rforge package archive <dir> <archive.tar>` and `rforge package restore <archive.tar> <dir>` round-trip packages; `rforge package audit <dir>` and `rforge package replay <dir>` verify checksums, source records/import receipts, legal acquisition, export-safe document assets, accepted-evidence support, lockfiles, analysis/report/redaction/provenance references, and package integrity offline
- `rforge [--project <path>] ui [--addr :8080]` serves the local Go + HTMX research cockpit for a research folder (address also configurable via `RFORGE_UI_ADDR`); run multiple instances on different ports to view several projects at once. `rforge --json ui` reports the resolved address, project, and route list without binding a port. Routes: `/` (shell), `/papers` and `/papers/{id}` (read parsed full text + native PDF), `/papers/{id}/pdf`, `/library`, `/artifacts` (PRISMA flow, meta-analysis heterogeneity/plots, interactive pan/zoom/click citation graph with a static SVG fallback), `/notebook` (browsable provenance journal for imports, source refreshes, parser runs, reviewer decisions, extraction edits, analysis reruns, report builds, and `/notebook/snapshot.json` export), `/map` (concept maps, citation neighborhoods, screening priority, parser quality, retrieval hits, evidence coverage, and `/map/snapshot.json` audit export), `/oss`, `/search`, `/projects` and `/projects/switch` (in-browser folder switcher)
- `rforge decisions` lists owner decisions and implementation trackers that intentionally keep remaining TODO items open
- `rforge decisions --check CHECKLIST.md` verifies unchecked TODO items, line references, and tracking issue references are decision/tracker-covered
- `rforge decisions --completion-audit CHECKLIST.md docs/todo-completion-audit.md` verifies decision/tracker coverage plus the closeout prompt-to-artifact audit; JSON output includes `completion_blocked`, `blocked_decisions`, `blocked_decision_ids`, and `license_resolution_verified` so automation does not treat covered TODOs as finished work
- `rforge --json decisions` exposes blocker routing and choice metadata such as `issue_title`, `todo_refs`, `issue_labels`, `milestone`, `options_considered`, and `owner_response_required_fields` for owner-decision automation
- `rforge decisions --markdown` prints a review-friendly decision/evidence table with routing/options metadata
- `rforge decisions --issue-body <decision-id>` prints an owner-decision issue body scaffold
- `make license-decision-live-audit` inspects live issue #1 approval fields; `make license-decision-approval-gate` passes only when the aggregate is `approved:true`
- `rforge paper <id> [<id>...]` resolves one or more papers by DOI, arXiv ID, OpenAlex ID, URL, or title — merges metadata from OpenAlex, Crossref, arXiv, Semantic Scholar, and Unpaywall into a single normalized record with OA status; batch resolution with `rforge paper <id1> <id2> "<title>"`
- `rforge paper --file <ids.txt> [--out <file>]` batch-resolves many IDs from a file (one per line, `#` comments)
- `rforge rank <topic> [--sources openalex,crossref,semantic-scholar] [--limit N] [--out <file>] [--graph <graph.json>]` scores papers by citation count (log), recency (1/(1+age)), and open-access bonus to decide what to read first — prints a numbered ranking with DOIs and arXiv IDs; `--graph` adds citation-graph prestige (log2(2+indegree)) from a `citations expand` graph; use `rforge --json rank ...` for machine-readable output. In either display mode, `--out` saves the complete ranked list and a validator-compatible `<file>.provenance.json` sidecar; it never writes the directory's `provenance.json` (that filename is rejected as an output destination). Output/provenance write failures return a nonzero exit code
- `rforge brief <search-dir> [--out <brief.md>]` synthesizes a completed `search batch` directory into a one-page research brief: manifest summary, query list, top-papers table with DOI/arXiv links, dedupe/failure gaps, and next-step commands
- `rforge compare <dir1> <dir2> [<dir3>...] [--out <diff.md>] [--labels l1,l2,...]` builds an overlap matrix across search directories (× = present), counts shared vs unique papers, and lists which top papers are missing from each dir
- `rforge lit --research-dir <dir> [--out <dir>]` synthesizes a source-grounded literature review across all topic subdirectories — writes LIT.md (summary, consensus themes ranked by paper count, cross-topic papers, open questions/gaps, source coverage) plus LIT.provenance.json; no LLM dependency
- `rforge note add|list|clear --project <path> [--paper <id>]` manages per-paper annotations in `data/paper-notes.json`, analogous to a persistent paper-notes layer for later recall
- `rforge history search <term> --research-dir <dir> [--limit N] [--out <file>]` searches across all research topic directories for matching papers by title, abstract, DOI, or arXiv ID — Feynman session-search analog; finds papers across topics with per-dir origin
- `rforge replicate <doi|arxiv-id> [--out <plan.md>]` resolves a paper and generates a replication plan: OA status, detected GitHub code repos, method summary from abstract, and a replication checklist with status for each step
- `rforge draft <doi|arxiv-id> [--out <draft.md>]` generates a structured research-note draft from a resolved paper: identifiers, OA status, method signals extracted from abstract, and placeholder tables for claims, evidence, and limitations — ready for human review and completion
- `rforge draft --search-dir <dir> [--out <draft.md>]` generates the same draft template from a completed search-batch directory, populating topic summary, top-papers table, and gaps from the manifest

Workflow configuration and provenance:

- `paper`, `draft`, and `replicate` use `RFORGE_UNPAYWALL_EMAIL` as the Unpaywall contact; without it, the resolver skips Unpaywall and retains available source metadata. Source endpoints honor `RFORGE_OPENALEX_URL`, `RFORGE_CROSSREF_URL`, `RFORGE_ARXIV_URL`, `RFORGE_SEMANTIC_SCHOLAR_URL`, and `RFORGE_UNPAYWALL_URL` ([resolver](../internal/sources/paper_resolver.go)).
- Successful `paper`, `rank`, `brief`, `compare`, `lit`, `note`, `history search`, `replicate`, and `draft` runs with the global prefix `rforge --project <path> ...` append a `cli.command` event to `provenance/events.jsonl`; append failures return nonzero instead of reporting success ([dispatch](../internal/cli/workflow_command.go)).
- `lit` and `replicate` outputs include generation timestamps; remote metadata may also change between runs, so their outputs are not promised to be byte-reproducible.
- Validate rank sidecars with `rforge provenance validate <file>.provenance.json` ([rank output](../internal/cli/rank_command.go), [validator](../internal/cli/provenance_command.go)).

## Search budgets and stop reasons

`search batch` defaults to **standard**. Presets bound the first query variants and sources in the supplied order; they do not generate queries or automatically escalate depth.

| `--depth` | Query variants | Sources | Requests | Low yield: new unique records/query | Consecutive low-yield queries | Consecutive errors |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `quick` | 3 | 2 | 6 | < 1 | 2 | 3 |
| `standard` (default) | 10 | 5 | 50 | < 2 | 3 | 5 |
| `comprehensive` | 30 | 20 | 600 | < 1 | 5 | 10 |

`--max-requests N` overrides the preset's total request cap (positive integer). A request is one connector search invocation, including a failed or interrupted attempt; connector-internal HTTP retries and multi-endpoint lookups are not separate budget units. `--limit` still bounds records requested per invocation.

Both `manifest.json` and `batch-state.json`, plus `--json` output, expose:

- `budget`: effective depth, query/source/request caps and thresholds (`max_api_calls` uses the graph budget's `MaxAPICalls` naming).
- `requests_used`: cumulative dispatched attempts, checkpointed **before** each call.
- `coverage`: `unique_records`, per-query `new_unique`/`complete`, and `consecutive_low_yield`. Yield counts newly added records under the same DOI/title dedupe policy as aggregate output, across all selected sources for a query. Failed/incomplete queries cannot signal saturation. This measures retrieval yield, not relevance or proof of exhaustive coverage.
- `stop_reason`: `completed` (selected sweep attempted, possibly with failures under `--continue-on-error`), `budget_exhausted` (query, source or request cap), `coverage_saturated`, `interrupted`, or `error_threshold` (consecutive failed attempts reach the preset threshold). Graceful SIGINT/SIGTERM retains exit 130/143; normal structured stops return 0. Without `--continue-on-error`, the first error remains fail-fast/nonzero.

Re-run the same command with `--resume` to skip completed pairs and retry failed/pending ones. Omitting budget flags preserves the checkpoint's budget; it never replenishes requests. An explicit `--depth` selects that preset, and an explicit `--max-requests` sets a new **total**, not additional allowance. Raise it to continue a capped run, or explicitly choose comprehensive when warranted. Changed query/source lists reconcile the coverage signal without resetting request usage. Failures remain recorded until actually retried successfully. Error streaks reset on resume.

The checkpoint's `result_count` identifies the committed prefix of `results.jsonl`; resume discards uncheckpointed or torn trailing appends before recomputing yield. Legacy checkpoints without this field retain their full results and infer minimum request usage from recorded completed/failed pairs. A fresh fail-fast run still rolls back all batch artifacts; a failed resume restores prior results but retains consumed request usage. `search resume --failures ...` retries only listed failures through the same budget service when a batch checkpoint exists; it cannot bypass its cap.

`citations expand --json` uses the same typed `stop_reason` and reports `requests_used`; Semantic Scholar recursive expansion no longer encodes `budget_stopped` in `rawRef`. Its `--max-api-calls` counts references and citations separately for `--direction both`.

Implementation: [search service](../internal/search/batch.go), [checkpoint/resume](../internal/search/batch_state.go), and [shared stop reasons](../internal/sources/graph_budget.go). Update this section when preset values, checkpoint accounting, or stop semantics change.

## Persistent research jobs

Each `search batch` registers a job under `<out>/jobs/`; its ID appears as
`job_id` in JSON output and `job:` in text output. Inspect or resume it with:

```sh
rforge jobs list --dir ./research/topic
rforge jobs status <id> --dir ./research/topic
rforge jobs resume <id> --dir ./research/topic
```

`--dir` defaults to the current directory. With `--json`, list returns
`data.jobs` and status returns `data.job`, including `id`, `kind`, `status`,
`progress.completed`, `progress.total`, and `stop_reason`. Progress counts
committed pairs in the selected worklist; budget or saturation stops can finish
before total progress is reached. Statuses are `queued`, `running`, `completed`,
`interrupted`, and `failed`. Published search results are saved in `data.job.result`
as the same `{"ok":...,"data":...}` envelope printed by `search batch --json`,
including manifest and output file references; older jobs may lack `result`.

Jobs run in the invoking process, never a detached daemon. A process-held lease
allows only one job writer per research directory. Inspecting a live job does
not interrupt it; after its host exits (including SIGKILL), the next inspection
or resume persists `interrupted`. Resumption uses saved, resolved queries and
options, so the original query file is not required. Connector credentials and
endpoints still come from the current host configuration. Completed pairs and
consumed request budget come from `batch-state.json`. Fail-fast artifact rollback
is preserved; its durable job record remains `failed` and can be retried.

Implementation: [job service](../internal/jobs/persistent.go),
[search integration](../internal/search/batch.go),
[shared request](../internal/search/jobs.go), and
[CLI adapter](../internal/cli/jobs_command.go). Update this section when job
lifecycle, request persistence, or command flags change.

## MCP stdio server

### Standalone client setup

Any MCP client supporting local **stdio** servers can launch the installed
`rforge` binary directly. No ResearchForge plugin, SDK installation, credentials,
or initialized project is required to connect and list tools. First [install the
engine](../README.md#quick-start) and verify support:

```sh
rforge capabilities --json --require mcp=1
rforge mcp serve
```

The server waits for an MCP client on stdin; a quiet terminal is expected, not a
startup failure. Normally let the client launch it rather than starting a second
process yourself. Merge this entry into a client configuration using `mcpServers`:

```json
{
  "mcpServers": {
    "research-forge": {
      "command": "rforge",
      "args": ["mcp", "serve"]
    }
  }
}
```

If your client uses a different config format, set its transport to stdio and
copy the same command and argument array into its local-server fields. Do not
supply a URL or use `--json`: stdout is already exclusively newline-delimited
MCP JSON-RPC, with diagnostics on stderr. There is no HTTP listener, startup
search, index opening, or project creation. This is standalone, manual setup;
plugin-managed registration is separate.

Use the absolute installed executable path for `command` if the client's `PATH`
differs from your terminal (on Windows, use `rforge.exe` and JSON-escaped paths).
To set a default project for library imports during search and evidence retrieval,
replace `args` with `["--project", "/absolute/path/to/review", "mcp", "serve"]`,
equivalent to:

```sh
rforge --project /absolute/path/to/review mcp serve
```

The project is not created at startup. Relative project paths and tool arguments
such as `dir`, `dirs`, and `out` resolve from the **server process's working
directory**, not the config file's directory; use absolute paths when uncertain.
Pass connector credentials and endpoint overrides through the client's server
environment settings or inherited environment, just as for CLI commands. Keep
secrets out of versioned configuration. Retrieval still needs an existing index
and any backend-specific configuration/consent.

### Offline connection smoke check

Reload/reconnect the client after saving its config. Its MCP initialization must
identify the server as `rforge`; `tools/list` must return exactly `search`,
`job_status`, `source_get`, `research_status`, `compare`, and `evidence_retrieve`.
Initialization and tool discovery use no network or project data. Do not call
`search` or `source_get` for this offline check: those operations use scholarly
APIs. Disconnecting should stop the server cleanly.

If launch fails, verify the executable path and the client's stderr log. If the
capability check fails, upgrade the engine; installing a skill alone does not add
MCP support. If artifact paths are wrong, check the server's working directory or
pass absolute paths.

### Tools and CLI equivalence

MCP is a thin adapter over the same services, persistence, search budgets and
provenance as CLI. Prefer typed MCP tools when registered in an agent host; use
CLI for shell scripts, offline batch automation, resuming jobs, or commands not
in this allowlist. Equivalent inputs produce equivalent research results, not a
separate MCP research workflow.

| Tool | Arguments | Equivalent CLI JSON operation |
| --- | --- | --- |
| `search` | `dir`, `queries` array, `sources` array; optional `limit` (25), `depth` (standard), `max_requests`, `continue_on_error`, `stats` | `search batch --out <dir> --query ... --sources ... --json`, asynchronously |
| `job_status` | `dir`, `id` | `jobs status <id> --dir <dir> --json` |
| `source_get` | `id` (identifier, URL, or title) | `paper <id> --json`, using `sources.ResolvePaper` |
| `research_status` | `dir`; optional `out` | `brief <dir> [--out <file>] --json` |
| `compare` | `dirs` array; optional `labels` array, `out` | `compare <dirs...> [--labels <csv>] [--out <file>] --json` |
| `evidence_retrieve` | `query`; optional `project`, `backend` (sqlite) | `--project <path> retrieve --query <query> [--backend sqlite\|opensearch\|qdrant\|hybrid] --json` |

Successful results preserve the CLI `{"ok":true,"data":...}` envelope in MCP
`structuredContent` and JSON text content. Results contain compact summaries,
metadata, bounded evidence passages, and artifact references, not full documents.
Invalid arguments or engine errors are MCP tool errors. Connector credentials,
endpoints, rate limits, and retrieval-backend privacy policy remain the CLI's.

For example, call `search` with
`{"dir":"./research/topic","queries":["battery recycling"],"sources":["scholarly-fast"],"depth":"quick"}`.
It returns `data.job_id` immediately. Poll `job_status` with that ID and the same
`dir`; a completed job's `data.job.result` contains the `search batch --json`
response, including its `manifest` path. Identity and timestamps differ between
separate executions; the shared engine publishes the same fixture results and
manifest fields. Search inherits the server's project for library imports.

Jobs run in the server process, not in the tool-call context or a detached daemon.
Searches sharing the configured project are serialized within the server to
protect library import/rollback, even with different output directories.
Disconnect or SIGINT/SIGTERM cancels active search work and paper resolution,
saves resumable search checkpoints, and joins the workers before exit. A queued
job waiting for a busy directory remains resumable. Use `rforge jobs resume <id> --dir <dir>` to continue
an interrupted or queued job. No approval, claim-acceptance, full-text acquisition,
or export tools are registered.

Implementation: [MCP adapter](../internal/mcp/server.go),
[CLI host wiring](../internal/cli/mcp_command.go), and
[offline parity/lifecycle tests](../internal/mcp/server_test.go).
Update this section when the tool allowlist, schemas, or lifecycle changes.

## Capability report and compatibility checks

`rforge capabilities --json` is an offline, read-only engine compatibility report.
It needs no project, credentials, network, or external tools. The usual JSON envelope
is `{"ok":true,"data":{...}}`; `data` contains:

- `name`, `version`, `commit`, `date`: the same build metadata as `rforge version`.
- `capabilities`: a map of family names to positive integer contract versions.
  Currently `search`, `paper`, `brief`, `compare`, `provenance`, and `mcp` each report `1`.
  `claims` is absent until implemented; absence never implies support.
- `error_codes`: the stable codes for this command, listed below (not a catalog of
  every error emitted by other commands).

Contract versions are minimum-compatible and monotonically increasing. Consumers
must check every required family, not infer support from the release number or
ignore unknown/missing entries. A reported version must be at least the required
contract version. Without `--json`, the command prints a human-readable report.

For a consumer with `min_rforge_version: "1.0.0"` and required contracts
`search: 1`, `paper: 1`, invoke:

```sh
rforge capabilities --json --min-rforge-version 1.0.0 --require search=1 --require paper=1
```

`--require` is repeatable; all requirements must pass. `--json` also works before
`capabilities`. No check flags means report-only mode. When checks are supplied:

- **Release:** the build version must be at least `--min-rforge-version` (when
  supplied), and all required capabilities must be present and sufficiently new.
- **Exact `dev` build:** skip the release-version comparison, but still verify all
  required capabilities. Never order the string `dev` against a release number.
- **Version syntax:** semantic `MAJOR.MINOR.PATCH` with an optional `v` prefix,
  prerelease, and build metadata. Use SemVer precedence, not lexical comparison;
  prereleases precede their release, and build metadata does not affect ordering.
  Non-SemVer build strings fail closed when a version minimum is requested.

Success exits `0`. Compatibility failures exit `1` with `ok:false` and an `error`
object containing `code`, `message`, `actual`, `required`, and `install` (always
`"install.sh"`); capability gaps also include `capability`. Values of `actual` and
`required` are strings; a missing capability uses `actual:"missing"`. The message
names the gap and points to `install.sh`. The command never installs anything or
changes host settings.

| Code | Meaning |
| --- | --- |
| `version_invalid` | Installed build version cannot be compared to the requested minimum. |
| `version_too_old` | Release version is below the requested minimum. |
| `capability_missing` | A required family is not reported. |
| `capability_too_old` | A reported family has an older contract than required. |
| `usage` | Invalid flags, minimum version, or contract requirement; exits `2`. |

Checks report the first failure: build version first, then required capabilities
in argument order. Usage errors use the standard `error.code`/`error.message`
envelope. A binary too old to implement `capabilities` (or a missing executable)
cannot pass a consumer's compatibility check; the consumer must name that gap and
point users to [`install.sh`](../install.sh), never silently fall back to a version
string. `rforge help --json` includes the capability command.

Behavior source: [command implementation](../internal/cli/capabilities_command.go)
and [public CLI tests](../internal/cli/capabilities_test.go). Update this section
when capability contracts or compatibility/error semantics change.
