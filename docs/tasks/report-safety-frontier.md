# Report safety frontier

Status: Accepted

The original discovery and delivery sections below are historical. [The next frontier](#next-frontier--visible-input-failures-and-correct-shared-counts) records current delivery status and the follow-on engineering batch; do not reopen delivered tasks.

Accepted for planning under the owner's standalone Plan For Me request, “high roi improvements”. The engineering choices below are delegated **planner's calls**, not new owner product decisions. Implementation, dispatch, installation, live calls and publication are not authorized by this planning pass.

## Brief and project model

ResearchForge serves researchers, reviewers and host agents with a local-first retrieval/provenance engine. The [README](../../README.md#resolve-rank-and-synthesize) explicitly offers `brief` and `compare` on existing search artifacts. The shared Go services own these operations; CLI adapters parse flags and display shared results. Search manifests and deduplicated JSONL are inputs, Markdown and JSON are outputs. No new subsystem or dependency is needed.

Discovery baseline: `1d486dbd8a82d56ec8b52dc2010f915b602c8937`. Recent installer/ranking reliability work and its closeout are delivered in history; [the existing reliability plan](engineering-reliability-frontier.md#reliability-closeout-addendum--2026-10-24) owns that evidence. This pass does not recreate its tasks or expand its narrowly scoped goal. The worktree task inventory has 75 records (74 done, one superseded), no open tasks, and eight goals. Manual deferrals are not verification. [RF-BENCH's owner pause](rf-bench-offline-round-2.md#owner-decision-planning-remains-paused) remains binding; benchmark planning, provider access and live/release qualification remain excluded.

Explicit user direction: high return on engineering effort, planning only. Interpretation / planner's default: prioritize small, evidenced correctness repairs in existing public workflows over speculative features or broad refactors. No performance baseline, ROI percentage, crash-frequency estimate or research-quality improvement is claimed.

## Goal and traceability

**RF-REPORT-SAFETY (derived engineering):** prevent misleading comparison overlap from ambiguous labels and prevent brief rendering panics on short optional timestamp metadata, preserving ordinary valid reports. The follow-on derived engineering scope below extends report safety to visible input-read failures and truthful per-directory shared counts; the original two-task acceptance and history remain unchanged.

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

## Next frontier — visible input failures and correct shared counts

Status: Accepted

This is the next standalone high-ROI planning batch, based on `3bbe919e358b73ed0f9534b4a78adb4a8f2ef4c9`. Its acceptance is delegated engineering scope under the repeated owner request, not permission to implement, dispatch, install or publish. The product model, normal Go environment, offline fixtures, no strategic expansion and RF-BENCH pause above still apply.

### Resume and traceability

**OBSERVED:** original plan delivery completed in `fc47a0a`. Both original implementation tasks are now closed with coordinator-verified receipts: [comparison labels](../../tasks/evidence/V0-022-e34bafb7.toml) (candidate `3999f16`, integration `3d0c0a2`) and [brief dates](../../tasks/evidence/V0-023-ebb3c50d.toml) (candidate `ab5d11a`, integration `f145a8f`). Current source and nearest tests contain both fixes. Receipt result/review are pass/passed; their compact checks arrays are not expanded historical command logs. Do not recreate either task or claim fresh execution of its tests from inspection. The original planning commit's full-hook pass was already reported; this batch uses a fresh normal hook at its own commit.

At this discovery baseline the worktree queue has 77 records, no open tasks, nine goals, and no drafts. The repository runner is idle with no workers, waiting jobs or holds. `RF-REPORT-SAFETY` remains unmet; this batch reuses its ID, title, source, priority 26 and focus=false, without editing completed contracts or receipts. The derived engineering extension is two new truthful-report boundaries within the same public brief/compare journeys, not a new product strategy or goal-completion attestation.

| Accepted behavior → source → observed gap → new task |
| --- |
| A brief summarizes a completed search directory ([CLI reference](../cli.md), line 99) → `Brief` in `internal/research/brief.go:63–79` → every results-file read error is ignored, so a missing file/directory read failure becomes a successful “No deduplicated records found” report → brief read-error task. |
| Compare counts shared vs unique papers ([CLI reference](../cli.md), line 100) → `Compare` in `internal/research/compare.go:209–222` → `inThis` includes every indexed paper in the directory, including unique papers, but is printed as “shared” → shared-count task. |
| Producer writes the required deduped artifact even with zero records → `internal/cli/source_commands.go:1489–1502`, `writeSearchBatchJSONL` before manifest/output commit → missing artifact is not the same as a readable empty artifact → preserve empty success, expose missing/read failure. |

These are high-confidence source observations, not new runtime repros. INFERRED impact: quiet input-loss misleads readers and callers about available evidence; inflated shared counts misstate coverage. Both are small, offline-verifiable fixes at existing shared-service seams. No measured frequency, ROI percentage or whole-repository audit claim is made.

### Decisions and executable specifications

| Choice / affected task | Planner's selected behavior and reason | Alternative rejected / exact verification |
| --- | --- | --- |
| Missing or unreadable brief results | `Service.Brief` returns typed failure `brief_results_read_failed`, exit 1, on any `os.ReadFile` error for `results-deduped.jsonl`, before output directory creation/writes. Keep the underlying cause/path through the normal wrapping error. A valid manifest still precedes results reads. | Do not emit a success warning/empty brief or fall back to raw results: these conceal lost deduped inputs or change corpus identity. `TestBriefRejectsResultsReadFailure` proves missing and directory-at-file-path cases. |
| Empty brief input | A readable zero-length or blank-lines-only deduped file still yields the existing successful empty report. Manifest/results count reconciliation and malformed-row policy are unchanged. | Do not equate an empty file with a missing file or add strict JSONL/schema validation. The same focused table's positive controls and `TestServiceBrief` / `TestBriefTimestampBounds` prove compatibility. |
| Meaning of shared | Count each existing identifier-indexed union row present in the directory AND at least one other directory (`countTrue(Matches) > 1`). Unique means exactly one directory, unchanged. “In all directories” remains stricter than shared for three-plus inputs. | Do not merely rename the wrong number or count only papers present everywhere. `TestCompareSharedCounts` covers disjoint, partial and all-directory overlap. |
| Existing count domains | Preserve the displayed total `len(s.papers)` as input-record count; shared/unique remain distinct identifiable-union counts, excluding no-ID papers and not inflating duplicate IDs. JSON result fields/membership stay unchanged. | Do not normalize DOI strings, dedupe all input metadata, redefine totals or add output fields. Duplicate-ID/no-ID controls in `TestCompareSharedCounts` explicitly pin the chosen domains. Total need not equal shared + unique for such inputs. |

**Non-goals / anti-goals:** no strict-reader refactor, changed malformed/titleless-row handling, `ReadPaperResults`/draft/history/lit policy change, identifier canonicalization, manifest count validation, transaction/provenance redesign, connector work, benchmark work, live qualification, docs/help registry edits or strategic feature. Rejecting bad I/O must not create a fresh report directory or overwrite an existing report. Correcting counts must not change prior labels/membership/order. Fixes belong in shared services rather than adapters. Reversible implementation details remain with the worker.

### Task A — brief read failure is not an empty corpus

Touches: `internal/research/brief.go`, new `internal/research/brief_read_error_test.go`, new `internal/cli/brief_read_error_test.go`. Highest priority because false-success input loss should be visible before a report is trusted.

1. Start with failing `TestBriefRejectsResultsReadFailure`: temporary valid manifest, then absent results file or directory occupying its path (deterministic even as root; do not rely on chmod). For each, require typed code/exit 1, useful filename/cause, zero result, unchanged existing output sentinel AND no absent output-parent creation. Retain unchanged manifest/input snapshots. Include readable empty/blank and valid-paper positive controls, retaining date bounds.
2. Add the minimum read-error branch. No shared reader extraction is needed; that would change unrelated draft/compare parsing policy.
3. Add `TestBriefResultsReadFailureCLI`: text and JSON calls preserve error code/exit 1, no success report/advice and unchanged/no output. CLI production plumbing already carries `research.Error`. Existing adapter/MCP guards remain in the full suite; no new tool or schema is needed.
4. Red-green-refactor, focused checks and normal independent implementation review; bounded owning-path fixture repairs required by these checks are allowed without weakening them. All inputs are locally creatable temporary fixtures, not external prerequisites.

Proposed/unrun check: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/research ./internal/cli -run 'TestBriefRejectsResultsReadFailure|TestBriefResultsReadFailureCLI|TestServiceBrief|TestBriefTimestampBounds|TestExecuteBrief' -count=1`.

### Task B — truthful per-directory shared summaries

Touches: `internal/research/compare.go`, new `internal/research/compare_summary_test.go`, new `internal/cli/compare_summary_test.go`.

1. Start with failing `TestCompareSharedCounts`, using distinct labels and temporary deduped JSONL: disjoint inputs require zero shared; mixed two-directory input requires only actual overlap; three-directory input distinguishes shared-in-two from in-all-three. For A containing AB, ABC and A-only, B containing AB and ABC, C containing ABC, expect A: 3 papers (2 shared, 1 unique), B: 2 (2,0), C: 1 (1,0), and one in all directories. Include empty input plus duplicate/no-ID records to pin existing count domains. Compare returned JSON membership against the expected unchanged union.
2. Correct only the shared count at the existing summary seam. Keep input totals, unique counts, global overlap, labels, row order and result schema unchanged.
3. Add `TestCompareSharedCountsCLI`: text output and `--out` use the corrected summary; JSON mode retains exact result fields/membership and writes the same corrected Markdown through `--out`. Reuse fixture helpers rather than changing the shared existing test file.
4. Red-green-refactor, focused checks and normal independent implementation review; bounded owning-path prerequisite fixture repairs are allowed without check weakening.

Proposed/unrun check: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/research ./internal/cli -run 'TestCompareSharedCounts|TestCompareSharedCountsCLI|TestServiceCompare|TestCompareRejectsDuplicateLabelsBeforeRead|TestExecuteCompare' -count=1`.

### Scheduling, Verify, Guard and closure

Two independent Now lanes, A first by rank, no product dependencies. Six disjoint files; no CLI/help table or agent-doc edits and therefore no `cli-surface`/`agent-docs` resource, live or exclusive resource. Shared read-only fixture helpers do not create write conflicts. Actual ownership/capacity checks still belong to the existing coordinator. No dispatch, service restart or publication is performed here; its existing separate authority may consume committed tasks independently.

Verify observable error/report semantics, not a synthetic numeric metric; direction is fewer false successes/miscounts without normal-case drift. Guard: focused selectors above and required `make check` at the execution/merged-tree gate. New tests are proposed/unrun during planning. Existing pre-commit hook runs the current suite in the normal Go environment; do not reintroduce the prior command-local offline overrides. No experiment or rolling successor: stop after saving this known frontier.

Pre-mortem: permission-based negative fixture accidentally passes as root (use missing path and directory); empty results become a false failure (positive empty/blank controls); three-way shared becomes “in all” or duplicates inflate counts (distinct membership fixtures). Inputs/results shape, original delivered fixes, isolated Git environments and offline deterministic tests are invariants across both tasks.

No owner-only choice remains. Deferred malformed JSONL/output-transaction concerns remain future evidence-dependent work, not hidden tasks in this batch.

### Saved next-frontier handoff and decision closure

- [V0-024-fb839930](../../tasks/items/V0-024-fb839930.md): Now, highest new priority; make brief input read failures visible. First action is the failing `TestBriefRejectsResultsReadFailure`.
- [V0-025-279390b1](../../tasks/items/V0-025-279390b1.md): Now, independent second priority; truthful per-directory shared counts. First action is the failing `TestCompareSharedCounts`.
- Existing `RF-REPORT-SAFETY` ID/goal metadata, completed IDs/history and all owner-paused work are preserved. No open questions, strategic proposals, rolling successor or live qualification task is added.
- Fresh independent read-only builder verdict: **nothing important left**. It confirmed evidence, reachable fixtures/checks, permitted bounded prerequisite repairs and disjoint lanes. **Worker-reversible disposition:** exact local names and fixture organization remain delegated within the saved contracts. This finding/disposition is persisted in both task records; no material owner/external blocker remains. Reviewer made no implementation edits or test runs.
- Planning readback and final checks cover the affected document and task records. A saved/selection-ready task is not implemented code. Exact validation and mandatory hook results are reported at this batch's planning commit; newly specified product regressions remain proposed until execution.
