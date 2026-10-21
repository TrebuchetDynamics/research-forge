# Scientific research examples

Start with either report below. These are genuine ResearchForge workflows with
**host-agent scientific synthesis**, not synthetic demonstrations, peer-reviewed
reviews, or human-approved scientific conclusions. Both remain exploratory,
`agent_reported` drafts; workflow completion is not scientific approval.

| Question / depth | Original report | Inspect the support | Reproduce |
| --- | --- | --- | --- |
| Which nanowire signatures discriminate topological Majorana modes from trivial alternatives? **Majorana / Comprehensive** | [Report](frontier-20261021-vfs2-live/majorana/report.md) | [Evidence CSV](frontier-20261021-vfs2-live/majorana/evidence.csv), [citation audit](frontier-20261021-vfs2-live/majorana/citation-audit.json), [provenance index](frontier-20261021-vfs2-live/majorana/provenance.json) | [Offline inspection / opt-in live rerun](frontier-20261021-vfs2-live/majorana/reproduce.md) |
| Which nickelate structural conditions and measurements are comparable, and what do they constrain about pairing? **Nickelates / Standard** | [Report](frontier-20261021-vfs2-live/nickelates/report.md) | [Evidence CSV](frontier-20261021-vfs2-live/nickelates/evidence.csv), [citation audit](frontier-20261021-vfs2-live/nickelates/citation-audit.json), [provenance index](frontier-20261021-vfs2-live/nickelates/provenance.json) | [Offline inspection / opt-in live rerun](frontier-20261021-vfs2-live/nickelates/reproduce.md) |

## What is saved, and who did what

The [discovery manifest](examples.json) links the original artifacts without copying
or rewriting them. Campaign
[`frontier-20261021-vfs2-live`](frontier-20261021-vfs2-live/summary.md) started at
**2026-10-21T15:06:50.640852Z**; [run.json](frontier-20261021-vfs2-live/run.json)
records identities, actual usage, failures and access gaps. Selection follows the
[accepted handoff](../../docs/benchmark/frontier-science-plan.md): latest parsed UTC
`started_at`, then lexically greatest `campaign_id`, among complete two-topic
campaigns with the required artifacts and passing real provenance validation.
Incomplete or synthetic dossiers are not public examples.

- **ResearchForge engine:** actual searches, normalization/deduplication, paper
  resolution, citation expansion, coverage summaries and checksum/run-event checks.
  Each topic retains its manifest, query plan, results, commands and immutable
  engine receipts under `provenance/`.
- **Pi host agent (`openai-codex/gpt-6.1-sol`):** source selection, passage assessment,
  competing explanations, evidence matrices and narrative conclusions. The
  [Majorana primary checks](frontier-20261021-vfs2-live/majorana/primary-checks.json)
  and [nickelates primary checks](frontier-20261021-vfs2-live/nickelates/primary-checks.json)
  retain attributed metadata and short excerpts; citation audits link claim anchors
  to these sources and 1-based CSV data rows. Semantic support is this attributed
  host assessment, not a result of the mechanical tests.
- **Human gates:** no independent human scientific acceptance, privacy approval,
  full-text acquisition or package export is claimed. Repository discovery grants
  none of those approvals.

## Read the limitations with the findings

These are selected abstract/metadata-led syntheses, not exhaustive systematic
reviews, recall benchmarks, proof of tool superiority, or gold sets. Numerical
conditions and uncertainties not stated in the inspected sources remain unknown.
Majorana retrieval exhausted its 100-call cap without saturation; nickelates
completed its sweep with all six Semantic Scholar pairs failing. Provider failures
are not negative scientific evidence. Retracted, disputed and unfavorable studies
remain visible in the reports, as do unresolved replication and mechanism questions.

The original executable is unavailable. A pinned-source rebuild has a different
hash, an explicitly disclosed campaign audit gap. Majorana's generic search receipt
says `standard` while its argv, manifest and checkpoint record `comprehensive`;
receipts were not rewritten to conceal this. See the reproduction pages for the
original/rebuilt hashes, source revision and exact historical commands. Separate
[tool/QC findings](frontier-20261021-vfs2-live/findings.json) and the
[campaign review](frontier-20261021-vfs2-live/review.md) do not replace scientific
limitations or constitute peer review.

## Inspect offline

From repository root, use Python's standard library and a compatible current
`rforge` build. This builds an **inspection executable**, not a reconstruction of
the unavailable original binary. With Go and dependencies already cached, these
commands require no scholarly API access; building may otherwise download Go
modules. Set `TMPDIR` to a writable scratch directory first.

```sh
go build -o "$TMPDIR/rforge-frontier-inspect" ./cmd/rforge
RUN=research/frontier-science/frontier-20261021-vfs2-live
PYTHONDONTWRITEBYTECODE=1 python3 docs/benchmark/frontier-science/check.py \
  --cases docs/benchmark/frontier-science/cases.json --run "$RUN"
PYTHONDONTWRITEBYTECODE=1 python3 "$RUN/validate-audits.py" "$RUN"
"$TMPDIR/rforge-frontier-inspect" --json provenance validate "$RUN/majorana/provenance.json"
"$TMPDIR/rforge-frontier-inspect" --json provenance validate "$RUN/nickelates/provenance.json"
go test ./internal/docs -run 'TestFrontierExamples' -count=1
```

The engine validator checks receipt/event identities and engine-output hashes;
the curation guards additionally check saved synthesis-output hashes, safe paths,
nonempty evidence and audit/report/source references. **Passing checks do not
establish scientific truth or human approval.** Read the cited passages and the
reports' rival explanations and access limits, rather than interpreting a green
check as endorsement.

## Request a live rerun

Use the topic reproduction pages for exact inputs, prerequisites, caps and actual
stop reasons. A new live run needs explicit authorization for provider access and
scope, a fresh output directory, and host-agent source assessment; any acquisition,
claim acceptance, privacy or export gates remain separate. Do not overwrite or
resume this completed campaign. Live metadata can change or rate-limit, so reruns
are not byte-deterministic, and CLI commands alone cannot recreate semantic
judgments. These discovery changes execute no new research or live rerun.
