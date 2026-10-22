# Agent-tier protocol and custodian runbook

V-B6 supplies offline protocol preparation, blinding, scoring and reporting.
**The Go utility does not run Pi/Codex, authorize spending, qualify a live search
gateway, or establish an improvement claim.** The separate opt-in
[Pi adapter](../../benchmarks/agent-tier/live/README.md) supplies V-B6-run's live
execution; see its [pilot record](pilot-pi-2026-10-21.md).

Authority: [accepted owner answers](../verifiable-engine-plan.md#owner-answers-2026-10-19-follow-up)
and [benchmark specification §§6–8](spec.md#6-scoring-definitions-shared-by-both-tiers).
The [utility](../../benchmarks/agent-tier/main.go) is external benchmark tooling,
not LLM orchestration inside rforge. No new dependencies are required.

## 1. Offline acceptance (start here)

From the repository root, with Go from `go.mod` and an existing `$TMPDIR`:

```sh
work=$(mktemp -d "$TMPDIR/agent-tier.XXXXXX")
go build -o "$work/agent-tier" ./benchmarks/agent-tier
setdir=internal/benchmarks/testdata/fixture-tiny
"$work/agent-tier" stub --set "$setdir" --out "$work/private"
"$work/agent-tier" blind --private "$work/private" --out "$work/blind"
"$work/agent-tier" score --set "$setdir" --blind "$work/blind" --out "$work/scores.json"
"$work/agent-tier" report --set "$setdir" --private "$work/private" \
  --blind "$work/blind" --scores "$work/scores.json" --out "$work/report.json"
go test ./benchmarks/agent-tier ./internal/benchmarks -count=1
```

The canned stub is restricted to the existing, checksum-locked `fixture-tiny`
package. It emits six observations (one question × two conditions × three
repetitions), including duplicate ranks and private attribution markers. It
never queries a service or starts a host. Canned DOI aliases, not a read of gold,
produce recall 2/3 with tools and 1/3 without. These are **fixture behavior, not
measured research quality**. Reports explicitly say `purpose: fixture` and
`mode: replay`; this offline agent-fixture exception is not a live evaluation.
With Q=1, contrast means exist but confidence bounds are null.

To exercise provider loss offline, replace the `stub` invocation above with:

```sh
"$work/agent-tier" stub --set "$setdir" --rate-limit-provider crossref \
  --out "$work/private-outage"
```

Use that private directory with fresh blind/score/report output paths. This is
still canned, offline fixture data: one Crossref 429 (`Retry-After: 3600`) and
one subsequent local skip per observation; the other two providers complete both
queries. Both conditions retain the failure and complete without spending the
run budget on retries. The provider acceptance case additionally checks a pair
with an outage in only one condition and an unaffected pair. The scored report
flags both members of an affected pair without dropping their measured outputs.
This fixture does **not** qualify a real gateway or demonstrate live resilience.

The acceptance test exercises all four command boundaries, checks scorer input
for condition/host/counter/provenance leakage, checks rank credit and paired Q/N,
and rejects attribution injected into the scorer envelope. Numerical tests cover
Student-t intervals, including Q=8 and the Q<2/zero-variance cases.

## 2. Pre-register before live work

Run **Pi first**, then **Codex as a separate experiment/report**. Owner decision
2026-10-21 narrows **V-B6-run to Pi only**; Codex belongs to **V-B6-codex** and is
not a prerequisite for publishing the Pi pilot. Do not pool hosts or models. The pilot is 8 independent held-out questions × 2 conditions ×
N=3 = 48 Pi runs; it estimates effect sizes, not the release-gate result.

Use the stub-produced `private/protocol.json` as a field-complete authoring
example, **not as a live registration**. Create and commit a new JSON under
`benchmarks/agent-tier/` before execution, replacing fixture identities with exact
approved values. [Protocol types/validation](../../benchmarks/agent-tier/protocol.go)
define the machine-readable contract:

| Pin | Required content |
| --- | --- |
| `schema_version` | 2 (provider receipts required; v1 registrations must be re-authored before new execution) |
| `host` | `name`: `pi` or `codex`; exact installed `version` |
| `model` | exact `provider` and model `version`, not an auto-routing alias |
| `sampling` | temperature, top-p, whether paired seeds are supported |
| `prompts` | exact shared `system` and `task` bytes; separate `with_rforge` / `without_rforge` tool/skill instructions |
| `rforge_version`, `revision` | exact engine/build under evaluation |
| `set_checksum`, `question_ids` | validated lock and all held-out IDs in manifest order |
| `repetitions` | 3 for the pilot |
| `caps` | identical per-pair token, search-page, tool, HTTP, wall-time caps |
| `run_order_seed` | signed integer locking schedule generation |
| `scorer_version` | `agent-v1` |
| `citation_metric_version` | `not_implemented` for this pipeline |
| `primary_metrics` | `["recall"]` |
| `secondary_metrics` | all other metrics, in the example's fixed order; never select after scoring |
| `provider_policy` | `with_sources`: at least two distinct canonical provider names; `continue_on_error: true`; `rate_limit_action: "skip_for_run"` |

Default caps per run: **32,000 total input+output tokens** (cached input included),
**20 search pages**, **60 host tool calls**, **100 HTTP attempts**, **600,000 ms**.
Any change must be pre-registered and equal across both members. Record billed
USD separately; caps are not usage estimates. The shared task prompt must require
an explicitly ordered bibliography and the same structured final/partial-answer
format in both conditions. Prompts cannot identify gold or the reference review.

`protocol_sha256` is SHA-256 of RFC 8785 JCS bytes without LF, using the existing
benchmark canonicalizer. It locks exact prompt strings as well as all fields.
Changing model, instructions, questions, budget, N, scorer or analysis means a
new registration, not a selective retry. Expanded experiments need a new
protocol/version after the pilot; this utility validates the three-repetition
pilot, not an arbitrary expanded design.

After owner approval and a separate spending cap, prepare inputs:

```sh
"$work/agent-tier" prepare --set /path/to/approved-locked-set \
  --protocol benchmarks/agent-tier/pilot-pi.json --out "$work/pi-private"
```

This validates manifest/review/snapshot/body checksums before writing:
`protocol.json`, deterministic balanced `schedule.json`, and `inputs/0000.json`,
etc. First-condition assignment differs by at most one pair; seeded shuffling
changes pair order, with both members adjacent to limit service drift. Supported
provider seeds are equal within a pair. Archive the exact schedule; do not reorder
or cherry-pick outcomes. Preparation itself performs no live work.

## 3. Qualify the host and trusted gateway (live operator)

Before accepting any live results, record host/model/engine receipts and verify:

- A fresh session, workspace and empty cache **per observation**, with no previous
  answer, hidden reference, condition partner, scored feedback or memory.
- With-tools: the ResearchForge skill/tools. Without-tools: the host's **native
  search**, with no rforge binary/plugin/skill/output/cache. The remaining host
  policy, exact model, sampling, prompt and budgets are identical. Never change
  approvals, permissions or runtime policy to make a host qualify.
- Agents receive **only their selected input file**, not the inputs directory,
  schedule, registration/private directory, repository checkout, `set.json`,
  review sheet, snapshots, gold, exclusion list or scoring artifacts.
- A trusted intermediary that knows the excluded review family and date cap
  mediates **all** searches, fetched pages, citation expansion and seeds in both
  conditions. Before delivering raw results (and again after merges), it rejects
  review identifiers/aliases/title as `reference_review_leak`. It never silently
  drops that review and scores the remainder. Neither condition may bypass it
  via an unrestricted shell/browser/network tool.
- The gateway applies the provider date filter and raw-record date upper-bound
  guard in [spec §3](spec.md#3-leakage-controls-and-access-separation). Unknown
  dates are excluded and counted; an unenforceable cap invalidates execution.
  A modern capture is not a historical index reconstruction.
- Accounting observes every nested search page and HTTP attempt, including
  retries, redirects and fetches; a batch is not one search. Host invocations
  count as tools; cache hits still count tools; tokens include cached input.
  Missing/opaque accounting **disqualifies** the host.
- Check remaining caps before every invocation; stop at the cap and keep the
  partial answer. Preserve measured overrun from an indivisible in-flight
  response, with `stop_reason: budget_exhausted`; never truncate counters.

These are host/gateway requirements, **not guarantees provided by offline
receipts**. The utility can validate receipt consistency, not independently
prove an external host executed its promised restrictions. If Pi or Codex cannot
interpose the gate or measure usage, do not run an unmatched comparison.

### Provider-independent search and outage qualification

The [v2 protocol](../../benchmarks/agent-tier/protocol.go) locks the provider
policy into its hash and each prepared input. The default with-tools mix is
**Crossref, Semantic Scholar and arXiv**, without OpenAlex. Author the exact
condition prompt to use the registered mix through the date/leakage gateway:

```sh
rforge search batch --query "$question" --sources crossref,semantic-scholar,arxiv \
  --continue-on-error --out "$run_dir/search"
```

The gateway must enforce the date cap and remaining budgets for this command;
the command alone is not a qualified benchmark gateway. Qualify at least two
independent providers with the same leakage/date/accounting controls. OpenAlex
is optional only when explicitly registered, never a required fallback. Do not
use a preset such as `all` that silently reintroduces it. Changing the mix or
prompt after registration requires a new registration, not a retry.

**`--continue-on-error` is necessary but insufficient:** current batch search
continues to later query/provider pairs, and connectors may retry HTTP internally.
The trusted gateway must intercept the **first** 429 before any retry/backoff,
quarantine that canonical provider across all queries, pages, expansions and
fetches for the rest of this observation, and locally skip later calls. Do not
sleep until `Retry-After`, resume failed pairs, switch aliases/endpoints to evade
the quarantine, or spend the remaining budget retrying the same provider.
Continue with remaining qualified providers and keep the observed partial answer
if none remain. Reset quarantine only for the next scheduled observation.

Apply the same policy to native search in the without-tools condition, logging
its actual provider names (not invented rforge sources). If the host hides
provider identity/status, retries or HTTP accounting, it is not qualified. Keep
`failures.jsonl`, `manifest.json`, `batch-state.json`, native gateway receipts and
redacted HTTP traces privately. Record every failure, HTTP status, raw sanitized
`Retry-After` value, operation stop reason and local skip; a 429 is not a zero-hit
successful search. Each HTTP attempt counts against usage; skips do not count as
HTTP attempts, though a host tool invocation still counts as a tool.

Before V-B6-run, inject a deterministic 429 into **each condition's real gateway**
and verify only one HTTP attempt reaches that provider, no retry wait occurs,
other providers proceed, and complete receipts reach the frozen-score report.
The offline stub tests import/scoring, not this external enforcement. A gateway
that cannot pass this probe is unqualified; do not substitute an unmatched run.

## 4. Import observations; retain failures

The live adapter/operator writes `experiment.json` in the prepared private
directory, following [the RawRun/Experiment structs](../../benchmarks/agent-tier/protocol.go)
and the stub's field-complete example. It contains the protocol hash, exact
schedule, and one run **in schedule order** per slot. Each run records model/caps,
private run ID/transcript/artifact locations, stop reason, measured usage, trusted
gateway receipts and structured answer. The required `provider_events` array
contains ordered trusted-gateway receipts for every HTTP attempt counted in
`usage.http_requests` (including retries, redirects, expansions and fetches) and
local quarantine skip in **both** conditions. For non-search traffic, `query`
identifies the operation or sanitized locator. Use canonical lowercase provider
names consistently across endpoints. Each event has exactly:

```json
{"provider":"crossref","query":"synthetic energy","action":"failed","http_status":429,"retry_after":"3600","error":"rate limited","stop_reason":"error_threshold"}
```

`action` is `completed`, `failed`, or `skipped`; `http_status` is the actual status
or 0 for transport failure/local skip. `retry_after` is the sanitized raw header
(or empty string); never retain credentials in errors/queries/headers. Failure
and skip errors are nonempty; successful errors are empty. A malformed HTTP 200
response is still a failed operation with status 200 and the parsing error.
`stop_reason` uses the run's five-value vocabulary but describes the provider operation; the
observation can still end `completed`. A skipped event has status 0, empty
`retry_after`, and `stop_reason: error_threshold`. Only a previous 429 permits
a quarantine skip. Any attempted retry of that provider after 429 invalidates
import, including a retry that later succeeded. Log non-429 failures too.

An empty event array is permitted only when no search or HTTP calls occurred
(e.g. a pre-search refusal). [Validation](../../benchmarks/agent-tier/providers.go)
rejects missing receipts for search/network usage, hidden failure statuses,
illegal skips, unregistered with-tools providers and any mismatch between
non-skip receipt count and HTTP usage. Fetches/redirects must not be left as
unexplained extra HTTP counts that could conceal an omitted 429. It checks
receipt consistency, not the honesty/completeness of an external recorder;
qualification and retained HTTP traces remain essential.

`answer` has only:

```json
{"bibliography":[{"identifiers":[{"scheme":"doi","value":"10.9999/p1"}]}],"claims":[{"text":"Claim text","quote":"Exact source text","locator":"https://example.org/source"}]}
```

Bibliography order and duplicates are preserved. Missing/unresolved identifiers
may use an empty identifier array (an occupied rank with no hit). Empty answers
use empty bibliography/claim arrays, never null. Claims/quotes are not rewritten
to improve scores. Preserve legally permitted raw source passages privately for
future facet checking; do not copy copyrighted full text into committed outputs.

Valid refusals, source/tool errors and budget stops stay as observed rows,
including zero-recall empty answers. Keep error details in the private transcript.
A host crash/missing pre-execution recording is infrastructure failure: retain
superseded attempts outside the scored matrix, rerun **both** members under the
same registration, and record the replacement relation in the private audit.
A leakage/protocol failure invalidates the experiment, not just its worse row.

Strict decoding rejects unknown/missing fields, duplicate keys, null collections,
non-finite values and mismatched model/caps/schedule. Invalid commands exit nonzero
with a JSON diagnostic (`{"error":"<code>"}`) on stdout and details on stderr.
Once both the set and registration validate, a failed `report` writes a full
`status: invalid` matrix with null metrics/aggregates, empty contrasts and the
failure code. Unreadable/invalid set or registration yields only the diagnostic,
never fabricated checksums or a usable report. Outputs are create-only: use a
new directory for a corrected attempt, retaining the invalid artifacts.

## 5. Blind, score, freeze, then join

1. **Custodian** runs `blind --private DIR --out NEW_BLIND_DIR`. It drops raw
   transcript, tool traces, provenance paths, run IDs, host/condition labels,
   counters and workspace artifacts by projecting only structured bibliography,
   claims, quotes and locators. Host/condition names and common tool names
   (including underscore-connected forms) are rejected in substantive fields.
   Cryptographically random 128-bit labels and a
   separately shuffled presentation order replace run order. Known attribution
   markers in substantive fields fail closed: arrange blind extraction rather
   than modifying a quote/claim/identifier. Before release, the custodian must
   also inspect for other tool names and formatting signatures that an automated
   marker list cannot recognize. Keep all substantive content/ranks unchanged.
2. `mapping.json` stays private. `NEW_BLIND_DIR` contains **only `answers.json`**.
   Give the scorer a separate OS account/container/filesystem mount with only
   the built utility, locked set/legal passages, blind answer file and new score
   output directory. Do **not** expose the custodian checkout, transcripts,
   protocol, mapping or operational counters. Separate command arguments are
   not an OS security sandbox; directory mode 0700 alone does not separate two
   processes under the same account. The offline demo uses one account solely
   for testing, not independent human scoring.
3. **Scorer** runs `score --set SET --blind BLIND_DIR --out scores.json`.
   The strict input has no condition field. Giving `--private` or `--protocol`
   to this command is rejected. Use only locked aliases, not titles, semantic
   guesses or new relevance labels. Freeze any blind extraction adjudications
   before this command. It writes `scores.json` and a create-only
   `scores.json.freeze.json` digest receipt **before** joining conditions.
4. **Custodian**, after freeze, runs `report` as in the offline example. Its
   `provider_availability` array has one entry per scheduled run: `slot`,
   `provider_lost` (any failed provider attempt), `pair_provider_lost` (either
   member failed), and the ordered `events`. It flags transient failures too,
   conservatively; it does not imply the provider was unavailable all run.
   Availability is joined only after freeze and is never sent to scorers.
   Reports retain affected rows and their primary/secondary metrics; no automatic
   exclusion, selective retry or unregistered outage-adjusted contrast is allowed.
   Discuss affected pair IDs and provider failures alongside all live conclusions.
   An invalid report has an empty availability array, not invented clean receipts.
   It verifies the protocol, schedule/full matrix and checksum links for original
   answers, mapping, frozen scores and freeze receipt. Only now does it join
   conditions and operational counters. Preserve the report and its separate
   `.audit.json` checksum links together with all private artifacts. Hashes guard
   accidental mislabeling, not malicious rewriting/signing.

Never send the joined report back to a scorer or agent until adjudication is
finished and the experiment is closed. Live held-out gold is not tuning material.

## 6. Metrics and interpretation

[Shared engine scoring](../../internal/benchmarks/agent.go) uses V-B3 identifier
normalization, unique first-hit credit, duplicate occupied ranks, no credit for
ambiguous multi-gold entries, recall and recall/precision at every locked K.
Precision uses denominator K even for short lists. It is **gold-set precision**;
unlisted papers can still be relevant. Never expand gold after seeing results.

Raw native-tool cluster membership is not imported by this version: all four
dedupe counts remain null with `dedupe_status: unavailable`, in both conditions.
Although a standalone V-D3 checker exists, this pipeline has no pinned passage
corpus/facet adapter: `citation_status: not_implemented`, all citation counts and
ratios null. Enabling citations requires a new registered scorer version wired
to that checker, not trusting an agent's facet annotations. Neither mechanical
citation correctness nor this benchmark establishes scientific semantic support.

[Analysis](../../benchmarks/agent-tier/statistics.go) reports all metrics, even
unavailable ones, for each condition overall and by domain. First average equal-N
repetitions within question, then macro-average questions. Ratio micro sums use
all N runs only for questions with all measurements. Paired deltas are
`with_rforge - without_rforge`, averaged over repetitions **within each question**,
then over questions. Two-sided 95% Student-t intervals use df=Q−1, not 48 runs or
24 independent repetitions. Q=8 uses t≈2.364624. Q<2 gives null bounds; Q=0 gives
null mean; zero variance gives a point interval. Bounds are never clipped.
Missing measurements require all N in both conditions or exclude that question
for that metric, disclosing Q; unmeasured values are never imputed as zero.

Positive retrieval deltas favor tools; negative resource/false-merge deltas favor
lower usage/errors. Secondary intervals are descriptive, not multiplicity-adjusted
significance claims. Disclose small-Q uncertainty, independent-review-family
assumptions, live-service/model drift, retrospective indexing and unmeasured
facets. Publish nulls and negative outcomes, not a favorable composite score.

Update this runbook when protocol/scorer fields, blinding rules, host accounting,
locked-set rules, citation support or analysis changes. Register a new protocol
before using changed behavior on held-out questions.
