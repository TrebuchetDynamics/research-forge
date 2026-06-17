# revtools study note

- Repository/ecosystem: R `revtools` ecosystem.
- Area: systematic-review screening, deduplication, bibliographic clustering, visual review workflows.
- Disposition: `pattern-reference`.
- License/action constraint: study UX and workflow patterns; keep R integration optional and do not vendor package code or sample data without review.

## Why it matters

revtools is useful as a systematic-review workflow reference: import references, find duplicates, visualize clusters, and support human screening decisions. ResearchForge can combine these ideas with provenance-first source records and local HTMX review screens.

## Patterns to learn

- Deduplication is a human-confirmed review task, not only a batch algorithm.
- Visual clusters help reviewers understand topical neighborhoods and duplicate candidates.
- Decision histories should be exportable for PRISMA and audit appendices.
- Import formats and field cleanup need deterministic, reviewable diffs.

## ResearchForge status

Implemented nearby capabilities:

- Import/export for BibTeX, RIS, CSV, JSON, CSL-JSON, Zotero RDF.
- Duplicate report/merge/split UX and cross-source merge logic.
- Screening queues, conflicts, uncertain queues, and PRISMA counts.
- Web-ready citation/knowledge graph view models.

Missing features:

- Field-cleanup suggestions with before/after provenance.
- Topic cluster review beyond duplicate identity clusters.

Implemented:

- `/dedupe` provides a revtools-inspired visual clustering screen for duplicate review and screening triage, with exportable reversible cluster decisions (`library identity-decision log`) and PRISMA/audit provenance linked to screening audit bundles.

## Recommended next slice

Add field-cleanup suggestions and broader topic-cluster review on top of the existing duplicate-cluster decision/provenance workflow.
