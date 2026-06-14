# ASReview study note

- Repository/ecosystem: `asreview/asreview`.
- Area: systematic-review screening, active learning, human-in-the-loop prioritization.
- Disposition: `pattern-reference`.
- License/action constraint: learn workflow concepts; do not port models or UI code without review.

## Why it matters

ASReview is a strong reference for reducing screening burden while preserving reviewer decisions. ResearchForge needs similar reviewer-auditable prioritization, not opaque automated exclusion.

## Patterns to learn

- Human labels drive ranking; the tool should not silently decide final inclusion.
- Screening effort and recall diagnostics are central deliverables.
- Reviewer decision history must be exportable and auditable.
- Active learning should be reproducible from a frozen dataset and seed labels.

## ResearchForge status

Implemented nearby capabilities:

- Screening configuration and decisions.
- PRISMA counts.
- Conflict detection and uncertain queues.
- Deterministic seed-overlap prioritization through `rforge screen prioritize`.

Missing features:

- Model-based ranking over abstracts/titles.
- Uncertainty sampling and balanced exploration/exploitation.
- Recall/effort curve simulation.
- Reviewer progress metrics.
- Stopping criteria and sensitivity diagnostics.
- Multi-reviewer adjudication workflow beyond conflict listing.

## Recommended slice

Extend `screen prioritize` to write a persisted prioritization run with input hashes, seed counts, ranking method, and output IDs. This gives reproducibility before adding heavier models.

Acceptance target:

```sh
rforge screen prioritize --stage title_abstract --method lexical-seed --out screening-priority.json
```

The output should include seed decisions, candidate count, ranking method, timestamp/provenance, and ranked records.
