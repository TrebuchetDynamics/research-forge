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

- The forge workflow DAG includes screen checkpoints with inputs, outputs, provenance actions, and restart-safe skips.

- Screening configuration and decisions.
- PRISMA counts.
- Conflict detection, adjudicated conflict resolution, and uncertain queues.
- Deterministic seed-overlap prioritization through `rforge screen prioritize`.
- Boundary-focused uncertainty sampling through `rforge screen uncertainty`.
- Smoothed naive-Bayes model-based active-learning ranking through `rforge screen model-prioritize`.
- Reviewer progress metrics through `rforge screen progress --stage <stage>`.
- Recall/effort curve scaffold through `rforge screen recall --stage <stage>`.
- Simple recall-threshold stopping recommendation through `rforge screen stopping --stage <stage> [--target-recall 0.95]`.
- The `/map` local web cockpit combines concept maps, citation neighborhoods, screening priority, parser quality, retrieval hits, and evidence coverage with no-JS server rendering and `/map/snapshot.json` audit exports.
- Evidence gap analysis cross-checks the research question, screened-in studies, parsed passages, accepted evidence fields, full-text acquisition, citation-locked claims, and analysis inputs before final inclusion.
- Reproducible review packages bundle the meta-analysis spine first-done artifact with project manifests, lockfiles, source plans, dedupe decisions, parser manifests, screening audit, extraction schema, accepted evidence, analysis/report artifacts, redaction policy, replay helper, audit placeholder, and checksums.
- Cross-tool benchmarks report deterministic fixture metrics for discovery recall, dedupe precision, parser field accuracy, reference normalization, retrieval quality, screening effort savings, and report/package reproducibility.
- Citation-locked synthesis can draft query expansions, screening rationales, extraction candidates, and report prose only when every suggested sentence has exact source support and remains unaccepted until reviewer review.
- The `/notebook` lab-notebook timeline records human and automated provenance events across imports, source refreshes, parser runs, reviewer decisions, extraction edits, analysis reruns, and report builds as a browsable journal with JSON snapshots.

Missing features:

- Richer model-based ranking over abstracts/titles beyond the current smoothed naive-Bayes scaffold.
- Balanced exploration/exploitation policy beyond standalone uncertainty sampling.
- Richer recall/effort simulation beyond observed-decision cumulative curves.
- Richer reviewer progress dashboards and trend metrics.
- Richer stopping criteria and sensitivity diagnostics beyond observed recall thresholding.
- Rich multi-reviewer adjudication workflow beyond adjudicated final decisions.

## Recommended slice

Extend `screen prioritize` to write a persisted prioritization run with input hashes, seed counts, ranking method, and output IDs. This gives reproducibility before adding heavier models.

Acceptance target:

```sh
rforge screen prioritize --stage title_abstract --method lexical-seed --out screening-priority.json
```

The output should include seed decisions, candidate count, ranking method, timestamp/provenance, and ranked records.
