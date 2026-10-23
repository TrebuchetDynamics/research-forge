# Offline engine-report deltas

Compare two existing compatible engine replay reports without rerunning retrieval,
contacting providers, or modifying either file. From the repository root:

```sh
go run ./benchmarks/report-delta --baseline OLD.json --candidate NEW.json
```

The command writes deterministic canonical JSON (RFC 8785 plus one LF) to stdout.
Rejections write an error to stderr, exit nonzero, and emit no success JSON.
There is no output-file mode, compatibility override, or public `rforge` command.

## Compatibility and validation

Both inputs must be complete, error-free schema-v1 **engine replay** reports.
Fixture reports can compare with fixtures; evaluation reports with evaluations.
Purpose, set ID/checksum, non-null snapshot digest, protocol digest, split, K and
question/domain/gold-count matrix must match. Versions and revisions may differ
and are preserved in `baseline` and `candidate` identities.

The shared strict artifact decoder rejects malformed JSON, duplicate or unknown
fields, and missing fields (including nullable fields). The comparison API
validates row identities, counts, ratios, ranked-hit consistency and facet
statuses, then recomputes aggregates using the existing engine aggregation
function. Missing, duplicate or inconsistent rows/aggregates are rejected.
Supplied aggregates must match recomputation exactly, without rounded values.
When both sides of an aggregate are measured, they must contain the **same
contributing question IDs**, not merely equal counts.

Implementation: [`CompareEngineReports`](../../internal/benchmarks/comparison.go)
and the [command entry point](main.go). The scoring/format authority is the
[benchmark specification](../../docs/benchmark/spec.md#7-report-json-schema-aggregation-and-reproducibility).
Update this README when those compatibility rules or the output format change.

## Reading the result

`aggregates` is sorted by condition/domain/metric/K, with null domain first
(all domains) and null K for non-at-K metrics. Each row contains:

- `baseline_question_count` and `candidate_question_count`;
- `macro`: `baseline`, `candidate` and signed `delta` values;
- `micro`: the same values for ratio metrics only; omitted for non-ratio metrics.

Deltas are always **candidate minus baseline**, not percentages or a judgment
that positive is better. For example, positive request-count deltas mean more
requests. If either measurement is unavailable, its delta is null, never zero.
Measured self-comparisons yield zero; unavailable self-comparisons remain null.

The output carries explicit limitations:

- Replay latency is captured timing, not current implementation speed.
- Reports alone do not prove input custody, set approval or engine verification.
- Differences are not a release, superiority or agent-improvement verdict.
- Precision is gold-set precision; non-gold papers may still be relevant.

A mutually missing question cannot be detected without the original manifest;
this utility checks the reports' internal consistency and compatibility, not their
actual custody or completeness against an independently authenticated set.
It does not emit an `engine_verified` claim or change RF-BENCH's qualification.

## Offline checks

```sh
go test ./internal/benchmarks -run '^TestEngineReportComparison' -count=1
go test ./benchmarks/report-delta -count=1
# Read-only reproduction check; this is not a new implementation-win claim.
go run ./benchmarks/report-delta \
  --baseline docs/benchmark/baseline-2026-10-19.json \
  --candidate docs/benchmark/baseline-2026-10-19.json
```

API tests use the tiny synthetic replay and hand-scored synthetic outcomes,
including opposite-sign macro/micro recall changes, domain/K separation, missing
measurements and incompatible contributing populations. Command tests exercise
strict decoding, canonical deterministic stdout, error exits and unchanged
input bytes, including the immutable [dated baseline](../../docs/benchmark/baseline-2026-10-19.md).
No live services, credentials or held-out tuning are needed.
