# Report safety frontier

Status: Accepted

Accepted for planning under the owner's standalone Plan For Me request, “high roi improvements”. The engineering choices below are delegated **planner's calls**, not new owner product decisions. Implementation, dispatch, installation, live calls and publication are not authorized by this planning pass.

## Brief and project model

ResearchForge serves researchers, reviewers and host agents with a local-first retrieval/provenance engine. The [README](../../README.md#resolve-rank-and-synthesize) explicitly offers `brief` and `compare` on existing search artifacts. The shared Go services own these operations; CLI adapters parse flags and display shared results. Search manifests and deduplicated JSONL are inputs, Markdown and JSON are outputs. No new subsystem or dependency is needed.

Discovery baseline: `1d486dbd8a82d56ec8b52dc2010f915b602c8937`. Recent installer/ranking reliability work and its closeout are delivered in history; [the existing reliability plan](engineering-reliability-frontier.md#reliability-closeout-addendum--2026-10-24) owns that evidence. This pass does not recreate its tasks or expand its narrowly scoped goal. The worktree task inventory has 75 records (74 done, one superseded), no open tasks, and eight goals. Manual deferrals are not verification. [RF-BENCH's owner pause](rf-bench-offline-round-2.md#owner-decision-planning-remains-paused) remains binding; benchmark planning, provider access and live/release qualification remain excluded.

Explicit user direction: high return on engineering effort, planning only. Interpretation / planner's default: prioritize small, evidenced correctness repairs in existing public workflows over speculative features or broad refactors. No performance baseline, ROI percentage, crash-frequency estimate or research-quality improvement is claimed.

## Goal and traceability

**RF-REPORT-SAFETY (derived engineering):** prevent misleading comparison overlap from ambiguous labels and prevent brief rendering panics on short optional timestamp metadata, preserving ordinary valid reports.

| Goal → accepted interface → implementation → verification gap → task |
| --- |
| Correct search-folder overlap → README `compare` journey → `Service.Compare` → existing happy-path tests use distinct labels and miss collision behavior → comparison-label guard task below. |
| Robust brief generation → README `brief` journey → `Service.Brief` → fixtures use full-length dates and miss short metadata → bounds-safe timestamp task below. |
| Trustworthy scientific/benchmark proof → separate accepted benchmark plans → separate harness and receipts → owner-paused/live-deferred → untouched, no replacement task. |

Goals: fix these two observable failures with offline regression coverage. Non-goals: new report schemas, broader CLI parsing, duplicate DOI normalization, ingestion policy, provenance redesign, atomic-output refactoring, benchmarks, UI, new connectors or strategic features. Anti-goals: silently claiming identical directory membership, introducing a stricter timestamp schema as an incidental crash fix, changing valid labels/dates, touching real research inputs in tests, or calling fixture passes scientific validation.

## Evidence and priority

### P1 — reject comparison label collisions

**OBSERVED, high confidence:** [compare.go](../../internal/research/compare.go), `Compare` lines 34–42, defaults labels to `filepath.Base` and checks only the count. Lines 63–83 store membership as label strings; lines 110–119 mark a column present by string equality. Therefore two different directories with the same basename, or explicit equal labels, are indistinguishable. A paper present only in A gets marked in B too. The membership algorithm establishes this by inspection; no product repro was executed in planning.

**DOCUMENTED:** the [README](../../README.md#resolve-rank-and-synthesize) promises an overlap matrix across search directories. [services_test.go](../../internal/research/services_test.go), `TestServiceCompare`, and [brief_compare_test.go](../../internal/cli/brief_compare_test.go), `TestExecuteCompareLabels`, use distinct labels. No existing task owns this new guard.

**INFERRED impact:** silent false overlap is more damaging to research interpretation than a visible usage error. Effort and risk are low: a shared-service validation guard plus focused service/CLI cases; no measured cost estimate.

**Planner's call / specification:** after default-label construction and existing count validation, require labels to be unique by exact string equality. A duplicate returns typed usage error `duplicate_labels`, exit 2, with a diagnostic directing callers to distinct `--labels`. Reject before opening any result file, creating parent directories or replacing output. Apply equally to inferred and explicit labels; do not trim/case-fold labels or silently suffix them. Preserve distinct-label results and return shape byte-for-byte where deterministic.

Given `/a/topic` and `/b/topic` without labels, comparison fails clearly rather than claiming overlap. Given the same directories with `--labels alpha,beta`, a paper only in A has membership `[alpha]`, none in B, and the summary does not count it in all directories. Given explicit `same,same` and unreadable inputs, the usage error wins before an input-read error and leaves pre-existing output unchanged.

Rejected alternatives: change membership to positional identity alone (duplicate display labels and JSON `In` entries remain ambiguous); silently rename labels (changes caller-controlled identities); redesign JSON membership/schema (larger compatibility surface). Explicit unique labels already provide a supported escape hatch.

### P2 — bounds-safe brief date display

**OBSERVED, high confidence:** [brief.go](../../internal/research/brief.go), `Brief`, renders any nonempty `manifest.CreatedAt` using `manifest.CreatedAt[:10]`. A valid JSON string of one through nine bytes reaches an out-of-range Go slice. The manifest struct accepts arbitrary strings. The service fixture uses a full RFC3339 timestamp. This is static evidence, not a newly run crash test.

**Planner's call / specification:** keep optional metadata permissive. Empty date omits the Searched row; nonempty dates shorter than ten bytes render in full; values of at least ten bytes retain the existing ten-byte prefix behavior. Do not parse or reject a previously accepted timestamp, fabricate a date, alter manifest input, or change JSON result fields. This is bounds safety, not general timestamp validation or Unicode-format redesign.

Given otherwise valid manifest/results with `createdAt: "2026"`, `Brief` succeeds without panic and displays `Searched: 2026`. Empty, nine-byte, ten-byte date-only and ordinary RFC3339 cases pin boundary and compatibility behavior. CLI text/JSON calls with `--out` succeed and write the same safe Markdown. All data is temporary synthetic input.

Rejected alternatives: strict RFC3339 validation (unnecessary compatibility change for a presentation-only field); omit all short values (hides supplied metadata); catch panics in CLI (leaves shared callers unsafe).

## Bounded task handoff

Both tasks consume existing public service options/results unchanged. They produce guarded behavior and regressions, not an API used by the other task. No product dependencies; both are Now, comparison first by rank. Their exact write sets are disjoint; no shared named resource is required. They do not edit `internal/cli/cli.go`, `docs/cli.md`, README or agent skill instructions, so neither `cli-surface` nor `agent-docs` is reserved. Actual runner admission/capacity remains separate from selection readiness.

### Comparison task

Touches: `internal/research/compare.go`, new `internal/research/compare_labels_test.go`, new `internal/cli/compare_labels_test.go`.

- Start with failing `TestCompareRejectsDuplicateLabelsBeforeRead`: default same-basename and explicit duplicate labels; nonexistent inputs, an existing output sentinel, and a second absent output-parent case prove usage precedes reading/writing or directory creation. Assert typed code and exit 2.
- Extend that focused behavior table with unique explicit labels and real temporary JSONL: first-only/shared/second-only identifiers, accurate `Papers.In`, total and Markdown summary; case/whitespace-distinct labels remain distinct and valid output still works.
- Add `TestCompareDuplicateLabelsCLI`: text and JSON error envelopes, actionable message and unchanged sentinel; unique explicit-label recovery uses the same fixture. No network.
- Add the smallest shared-service guard, run focused tests red then green, refactor only this seam, and submit normal independent implementation review.

Proposed/unrun focused command: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/research ./internal/cli -run 'TestCompareRejectsDuplicateLabelsBeforeRead|TestCompareDuplicateLabelsCLI|TestServiceCompare|TestExecuteCompare' -count=1`.

### Brief task

Touches: `internal/research/brief.go`, new `internal/research/brief_timestamp_test.go`, new `internal/cli/brief_timestamp_test.go`.

- Start with failing table `TestBriefTimestampBounds`: empty, one-byte, nine-byte, ten-byte and RFC3339 values. Call the service normally (a panic must fail the test), assert exact Searched display or omission, returned report and saved Markdown, and unchanged input files.
- Add `TestBriefShortTimestampCLI`: synthetic short-date session in text/JSON modes with `--out`; success envelope/output agree with service-rendered Markdown and no panic.
- Make rendering bounds-safe, preserving the full-length date prefix; run focused tests red then green, refactor only this seam, and submit normal independent implementation review.

Proposed/unrun focused command: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/research ./internal/cli -run 'TestBriefTimestampBounds|TestBriefShortTimestampCLI|TestServiceBrief|TestExecuteBrief' -count=1`.

Full execution guard for both: `make check`. The authorized runner owns its merged-tree gate; do not claim affected-only checks as a full-main pass. Normal tests stay offline/deterministic; any Git subprocess uses an isolated Git environment. No product repro tests, installs or build checks are run for discovery. The mandatory pre-commit hook runs the existing `make check` guard when committing the planning batch; it does not implement new tests. Named new tests remain proposed, not existing passing baselines.

## Verify, Guard and stopping rule

Verify observable behavior rather than a fabricated numeric metric: ambiguous labels cannot produce a successful comparison; short timestamp input cannot panic; unique labels and normal dates retain prior behavior. Direction: fewer false reports and crashes, with no compatibility drift outside the specified cases. Guard: focused owning suites and `make check` at execution. One red-green-refactor cycle per task, bounded to these seams; stop for genuinely new consequential scope, not ordinary reversible coding choices. No experiment config, iteration campaign or rolling successor plan.

Pre-mortem: CLI-only validation leaves direct callers exposed (guard shared services); label normalization changes identities (exact comparison only); a short-date fix imposes unrequested schema validation (render permissively); a guard writes before rejecting (sentinel/read-precedence cases); fixtures accidentally hit real artifacts (temporary directories only).

## Planning checklist and validation

- [x] Discover accepted direction, current source, prior plans, closed backlog and active ownership.
- [x] Specify/design these two narrow outcomes; distinguish evidence from inference and planner defaults.
- [x] Save complete goal/task records through `autowork tasks`, read them back, and perform builder review.
- [x] Validate the affected batch and check local links/whitespace: store valid (77 tasks, nine goals, no drafts), 11 local targets exist, whitespace check passes.
- [ ] Finish the plan-document commit with the normal required pre-commit hook; result reported in the final handoff.

Autowork capability check observed prepare API 4. `autowork tasks prepare .` returned READY with no drafts. Existing diagnostics include goal-without-work and historical legacy references; these are not authority to rewrite unrelated goals, history or application `goals.json` usage. Runner inspection at planning time showed the repository service running but idle, zero workers, no held/root jobs. This pass neither starts nor restarts it; an existing separately authorized runner may independently consume newly committed queue entries.

Outstanding owner questions: none for these two engineering repairs. Strategic opportunities are not introduced. Other possible report concerns (silent corpus parse skips and direct output writes) need their own impact/compatibility investigation before becoming tasks; this bounded frontier does not claim an exhaustive audit.

## Saved handoff

- [V0-022-e34bafb7](../../tasks/items/V0-022-e34bafb7.md): Now rank 1000, comparison correctness first. Next action: add the failing `TestCompareRejectsDuplicateLabelsBeforeRead`.
- [V0-023-ebb3c50d](../../tasks/items/V0-023-ebb3c50d.md): Now rank 2000, independent bounds-safe brief lane. Next action: add the failing `TestBriefTimestampBounds`.
- [RF-REPORT-SAFETY](../../tasks/goals/RF-REPORT-SAFETY.md): new derived engineering goal; unmet, priority 26, no focus change. Existing goal statuses/IDs and paused scope remain untouched.
- Fresh independent read-only builder review found no blocking gaps. Its two minor suggestions (absent-parent assertion and case/whitespace-distinct label controls) are incorporated in P1 and its task acceptance. Both lanes have disjoint writes and no prerequisites; selection readiness is not execution authority or worker admission.

### Delivery continuation

The planning commit was attempted with `GOPROXY=off GOSUMDB=off` to prevent dependency installation/network activity during planning. The mandatory `.githooks/pre-commit` runs `make check`; `fmt-check` passed, then `go mod tidy -diff` failed before tests: `golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64: verifying module: checksum database disabled by GOSUMDB=off`. This is an environment/toolchain qualification failure under the planner's offline settings, not evidence of a product regression. `GOTOOLCHAIN=local go version` reports installed Go 1.26.2; `go.mod` specifies Go 1.26 and suggests toolchain 1.26.6. No toolchain was installed, no manifest changed, no hook bypassed, and no lower-toolchain substitution was used to manufacture a pass.

On continuation, the normal `go version` reports Go 1.26.6 successfully. The earlier failure was caused by the planner's unnecessary `GOSUMDB=off` override, not a demonstrated missing toolchain. Recovery is to remove those command-local overrides and use the normal hook, without installing dependencies, bypassing checks or changing manifests.

The existing runner independently committed the goal and task records in `66e58b5` and merged them into the consumer at `77c1821`. Their contracts remain unchanged; the plan document is the remaining staged file. This session has not dispatched or implemented either task. Finish this document's planning-only commit with the normal pre-commit hook, then inspect committed-store selection. Store and local-link/whitespace checks passed independently; final hook results belong to the continuation report, and new product regressions remain proposed.

Update trigger: changes to the cited service guards, output shape, task disposition or commit blocker. Proposed product regressions remain unrun.
