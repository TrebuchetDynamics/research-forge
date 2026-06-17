# Meta-analysis spine phased roadmap

This roadmap orders ResearchForge super-tool work around the **Meta-analysis spine**. The first product milestone is not a polished prose-only report; it is a **Reproducible review package** that can be audited, replayed, restored, and inspected locally.

Broad research-cockpit features are intentionally deferred until the package is audit/replay safe. They may be designed as seams, but they should not displace the review-package path.

## Guiding constraints

- The CLI remains the reproducible automation path.
- The Go + HTMX cockpit is a review/control surface over shared Go services.
- Normal tests stay local, deterministic, fixture/fake-backed, and network-free.
- External OSS projects are used according to `opensource/inventory/manifest.json` dispositions.
- Every irreversible scientific or data-sharing decision requires a review gate.
- Every accepted evidence item and report claim must trace to source support.

## Milestone 0 — Blueprint and gates

Goal: make the spine buildable without ambiguity.

Deliverables:

- `docs/meta-analysis-spine-blueprint.md` kept current with modules, command families, pages, storage, provenance, validation, and OSS dispositions.
- A `rforge forge` state-machine specification.
- Reproducible review package acceptance criteria.
- Acceptance-test matrix covering unit, CLI e2e, handler, Playwright, screenshot, provenance, and replay tests.

Exit criteria:

- The first implementation slice can be picked without re-litigating the product sequence.
- Every later phase has a validation target.

## Milestone 1 — Question and source plan

Goal: turn a research question into an auditable protocol and source plan.

Deliverables:

- PICO/PECO/SPIDER/freeform question compiler.
- Inclusion/exclusion criteria drafts.
- Extraction schema seeds.
- Connector capability registry.
- Source-plan dry runs and live-smoke snapshot storage.
- Reviewer-approved query-expansion suggestions.

Exit criteria:

- A project can record exactly what it intends to search, why, and with which connector constraints before network calls happen.

## Milestone 2 — Import, identity, and dedupe

Goal: ingest source/library records without losing identity or reviewability.

Deliverables:

- Zotero/JabRef fidelity improvements.
- Reference-manager interchange matrix.
- Source-fusion identity resolver.
- Reversible merge/split decisions.
- Visual dedupe/cluster review surface.

Exit criteria:

- Imported records can be traced back to original formats/sources, and duplicate decisions can be audited and reversed.

## Milestone 3 — Legal full text

Goal: acquire full-text candidates only through explicit legality/privacy gates.

Deliverables:

- DOAJ/CORE OA discovery adapters.
- Unified full-text candidate comparison across OA sources and local files.
- Legal acquisition queue.
- Attachment/note/local-path privacy gates.
- PubMed/Europe PMC/PMC structured full-text workflow.

Exit criteria:

- No document enters the review package without recorded OA/license/shareability metadata and reviewer approval where needed.

## Milestone 4 — Parser arbitration and references

Goal: parse full text without treating any parser as silent truth.

Deliverables:

- Parser-output manifests.
- Multi-engine parser arbitration.
- Stable offsets, layered annotations, citation spans, confidence, and reconciliation outputs.
- Parsed-reference adjudication.
- Cross-source reference normalization.
- Bibliography-to-citation graph import.

Exit criteria:

- Parsed passages, references, and selected fields have tool/version/checksum provenance and reviewable conflict decisions.

## Milestone 5 — Retrieval and domain map

Goal: make source material discoverable while preserving privacy and reproducibility.

Deliverables:

- OpenSearch lockfiles and partial-failure provenance.
- Qdrant provider registry, embedding compliance profiles, and vector invalidation.
- Hybrid ranking tuning files.
- Retrieval benchmarks.
- Topic/domain-map artifacts.
- Accessible/no-JS graph tables.

Exit criteria:

- Retrieval results cite passages and ranking configuration; graph/topic views remain inspectable without JavaScript.

## Milestone 6 — Screening and review assistance

Goal: support systematic-review screening without opaque automation.

Deliverables:

- Persistent ASReview-style active-learning runs.
- Exploration/exploitation policies and recall/effort diagnostics.
- Reviewer assignment, conflict/adjudication panels, uncertain queues.
- Risk-of-bias suggestion queues.
- HTMX screening cockpit.

Exit criteria:

- Screening assistance is reproducible, reviewer-audited, and exportable as part of the review package.

## Milestone 7 — Evidence and gap analysis

Goal: convert accepted studies into auditable analysis-ready evidence.

Deliverables:

- Evidence extraction grid.
- Scientific entity suggestions.
- Citation-locked LLM/extraction suggestions.
- Evidence gap analyzer.
- Per-passage provenance in reports/package audits.

Exit criteria:

- Every analysis-ready value has exact source support, correction history, and inclusion status.

## Milestone 8 — Statistics and method comparison

Goal: make statistical analysis reproducible and defensible.

Deliverables:

- Additional effect-size calculators.
- Better subgroup/meta-regression UX.
- Influence/sensitivity/publication-bias diagnostics.
- Publication-ready analysis artifact manifests.
- PyMARE secondary-engine comparison.
- Method-comparison workbench.

Exit criteria:

- Analysis outputs include inputs, model settings, engine versions, warnings, checksums, and visible method disagreements.

## Milestone 9 — Reproducible review package

Goal: produce the first done artifact of the Meta-analysis spine.

Deliverables:

- Claim-to-evidence trace views.
- Claim export blockers.
- Package format and manifest.
- Package replay/audit commands.
- Package archive/restore compatibility tests.

Exit criteria:

- A package can be moved to a fresh machine/workspace, audited, restored, and replayed without private local state.

## Milestone 10 — Cockpit and guided Forge workflow

Goal: make the spine usable without weakening reproducibility.

Deliverables:

- Forge home timeline.
- HTMX workbenches for each spine phase.
- Dashboard information architecture.
- Dashboard privacy/permissions model.
- One-command `rforge forge` guided workflow.
- Playwright and screenshot coverage.

Exit criteria:

- Dashboard buttons map to CLI-equivalent actions and cannot bypass review gates.

## Milestone 11 — Broader research cockpit after the package

Goal: expand from meta-analysis package production into a wider research cockpit.

Deliverables:

- Project knowledge graph queries.
- Live research-map cockpit features.
- Lab-notebook timeline.
- OSS-inventory-to-roadmap reports.
- Cross-tool benchmarks.

Exit criteria:

- Broader exploration features reuse package-provenance primitives instead of creating unaudited parallel state.

## Deferred until after package safety

The following are valuable but must not become prerequisites for the first Reproducible review package:

- open-ended knowledge graph exploration;
- broad topic-modeling cockpit polish;
- generalized lab notebook beyond spine events;
- publication-perfect prose generation;
- learned rerankers beyond calibrated/fake-backed retrieval benchmarks;
- real-service integration in normal test gates.
