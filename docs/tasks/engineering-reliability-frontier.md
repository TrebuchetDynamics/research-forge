# Engineering reliability frontier

Status: Accepted

This is a standalone, objective-free Plan For Me discovery result at repository baseline `95e5506`. It authorizes this planning batch, not implementation, live calls, installation or publishing. The goal below is a **derived engineering goal / planner's call inside accepted ResearchForge behavior**, not a new product strategy.

## Project model and protected decisions

ResearchForge is a local-first, retrieval/provenance-first research engine for researchers, reviewers and host agents. Important journeys are installation → capability check; query → scholarly metadata/dedupe/citations → auditable report; and project-based, human-gated evidence/analysis/package workflows. [README](../../README.md), [PRD](../../RESEARCH-FORGE-PRD.md) and the [accepted engine plan](../verifiable-engine-plan.md) own intent.

Implementation is a Go CLI (`cmd/rforge`, `internal/cli`) over shared services (`internal/research`, search/library/claims/provenance/etc.). CLI/MCP/native plugins are interfaces rather than independent research implementations. Research folders own results and receipts; project stores own accepted evidence; `tasks/` owns the queue. `install.sh` is the user-scoped multi-host distribution entry point. Go/adapter tests and `make check` own the offline guard; actual scientific or host qualification is separate.

Maturity: substantial engine functionality and public scientific examples are implemented. Recent completed work includes Hermes profile/pin/skill-preservation hardening and offline benchmark comparison/pilot readback. Coordinator-verified completion is distinct from manual deferral/decision closure. A green suite or a present report is not proof of scientific correctness, release superiority or live host qualification.

**Protected owner decision:** [RF-BENCH round-2 owner answer](rf-bench-offline-round-2.md#owner-decision-planning-remains-paused) explicitly pauses further offline benchmark planning, keeps the goal unmet and retains provider/live deferrals. This new engineering batch does not reopen RF-BENCH, V0-012, the 48-run/expanded campaign, citation evaluation, Hermes local qualification, credentials, spend or publishing. No rolling successor plan is created.

## Goal model and traceability

| Class / outcome | Intent → implementation → verification | Current disposition |
| --- | --- | --- |
| DOCUMENTED, accepted: dependable passive host installation | README / docs/install.md / accepted packaging principles → install.sh → installer sandbox and recent V0-013 Hermes receipt | Hermes-specific safeguards delivered; a separate pre-existing shared quiet-wrapper gap remains below. |
| DOCUMENTED, accepted: reusable research workflows with useful errors and source identity | README resolve/rank journey / engine plan → internal/cli/rank_command.go → internal/research/rank.go → service/CLI fixture tests | Empty-topic rejection exists, but whitespace-only input is not rejected at the service boundary. |
| DOCUMENTED, accepted: narrow provenance facets, one engine behind adapters | engine plan / ADR 0009 → shared services and provenance/claims packages → existing task receipts and offline tests | Preserve implemented behavior. No new facet, scoring model or scientific-approval claim is introduced. Full current scientific adequacy is not inferred from those receipts. |
| DOCUMENTED, accepted but owner-paused: RF-BENCH quality proof | accepted benchmark spec → engine/agent harness → recorded replay/pilot infrastructure and owner deferrals | Remains unmet and paused. Lack of provider input is not a prerequisite of either engineering fix. |
| DERIVED ENGINEERING: trustworthy failure boundaries | existing error/side-effect/installation contracts → two bounded corrections below → new red-green fixtures plus existing owning suites | One new RF-RELIABILITY goal, with two independent Now tasks. |
| UNKNOWN / unmeasured | whole-project performance, scientific quality outside saved evidence, every remaining roadmap item | No invented benchmark baseline, feature gap or improvement percentage. No sweeping rewrite is justified. |

## Derived engineering goal

**RF-RELIABILITY:** preserve trustworthy error boundaries in the existing installer and ranking workflow: a failed native/build command must not turn into success merely because output is quiet, and a blank ranking request must fail as usage before reading input artifacts, selecting/searching connectors or writing output/provenance.

**Goals:** correct observable exit/error behavior; prevent avoidable side effects; preserve positive existing workflows. **Non-goals:** new features, ranking relevance algorithms, generalized parser/framework refactors, caller-side orchestration, new providers, benchmark work, new runtime controls or live qualification. **Anti-goals:** masking failures, claiming arbitrary blank-query results as research, changing valid query bytes/scoring, touching user installations during a fixture run, or calling fixture success scientific verification.

**Invariants:** preserve Bash 3.2 compatibility, current flags and host-specific safety; retain nonblank ranking behavior, artifact/provenance schema and error-code conventions. All new tests are offline/deterministic. Git subprocesses in fixtures scrub inherited repository selectors. Normal full guard remains `make check`. This planning pass writes documents/task records only.

## Evidence and priorities

### P1 — shared installer quiet-mode failure propagation

**OBSERVED (high confidence):** `install.sh::run` and `run_in` execute a child inside `if ...; then return 0; fi`, then assign `local status=$?` *after* the if statement. A safe, nonmutating Bash semantics check (`if (exit 7); then :; fi; printf '%s' "$?"`) returns 0 there. Thus the wrappers lose the child's nonzero status in quiet mode. Verbose branches return the child's status correctly.

`hermes_run` already captures status on failure explicitly, with a comment that it does not change other hosts. The older shared helpers are still used by source builds and non-Hermes host operations. This is a pre-existing shared bug, not a regression attributed to the Hermes fix.

**INFERRED impact, to pin with the new tests:** if a previous compatible engine/plugin remains on disk after a failed update/install, later existence/capability checks can pass and conceal the failed operation. Source inspection plus shell semantics establishes the wrapper defect; this planning pass did not run a new installer failure fixture or change a local installation.

Existing `TestQuietHidesCommandsAndVerboseShowsThem` covers successful quiet output, not native/build failure propagation. Isolated sandbox, fake host/Go helpers, release fixtures and pin-preservation tests already exist under `internal/installer`.

### P2 — ranking whitespace-only input boundary

**OBSERVED (high confidence):** `internal/research/rank.go::Service.Rank` rejects only `topic == ""`. It reads an optional graph and selects/searches configured connectors for whitespace-only input. `internal/cli/rank_command.go` forwards the topic verbatim to that service; existing flag-position and paper blank-input tests do not cover this case. Service error helpers already map usage to exit 2.

**INFERRED impact, to pin with the new tests:** callers can get an irrelevant query, graph/read failure or no-results error rather than a usage error, and potentially reach provider work for an unusable topic. No live ranking command was executed. Even if an individual connector rejects whitespace later, shared service validation is still inconsistent and cannot guarantee zero connector selection/search.

Use `strings.TrimSpace(topic) == ""` (or equivalent Go Unicode-whitespace check) only as the rejection predicate. Preserve every nonblank topic's original bytes, scoring, output structure and existing flag semantics; do not trim/rewrite a valid user's query. Fix the common service, not duplicate research logic in an adapter.

### Evidence bindings

At discovery, source SHA-256 values were:

- install.sh: `f89fb4d58972c69f48a1b42d55bd5060265bc388498d10190b22bfc4dc49bd93`
- internal/research/rank.go: `dccc715a7fdd70d17e4eafe562cb8b74b8b151b47ee8bd1b096c910846f67845`
- internal/cli/rank_command.go: `00bfea7c31889a66c6131d710aee322ae895dafed9699b62f7056323fd4e8224`

No open task currently owns either fix. Existing provider-blocked and completed/manual task history were inspected; these are not replacements for paused work.

## Design, alternatives and bounded decomposition

**Planner's call:** two small independent implementation lanes, quiet-wrapper fix first by priority. They have disjoint file sets and no product dependencies. They may execute in parallel if the runner has capacity and normal ownership checks pass; no `exclusive` or `live` resource is justified. Stop after this known batch rather than adding a self-replenishing discovery/planning loop.

- Installer task owns `install.sh` and new `internal/installer/quiet_failure_test.go`. Reuse the existing sandbox/fake helpers; do not append tests to the shared suite. Capture child failure inside the actual failing branch while preserving logging/tails and all other behavior. Do not opportunistically remove `hermes_run` or refactor host lifecycles.
- Ranking task owns `internal/research/rank.go`, new `internal/research/rank_input_test.go` and new `internal/cli/rank_input_test.go`. Production change belongs only to the shared service; CLI behavior follows through existing error plumbing. The new tests do not alter the command table or help, so no shared cli-surface edit is planned.

Rejected alternatives: remove quiet mode or rely on post-install probes (breaks an existing feature or leaves failure masking); rewrite the installer/process framework (unnecessary risk); validate only the CLI (leaves direct service callers exposed); change ranking/scientific metrics or do more benchmark research (outside this frontier and the explicit pause).

### Installer acceptance design

Begin with `TestInstallerQuietHostFailure`: seed an already installed Pi package in the existing fake sandbox, make native `pi install` fail, and show that quiet mode currently hides it while verbose mode fails. This prevents a later successful plugin-list check from accidentally masking the test's root cause. Assert a failed component/nonzero installer result, retained native failure diagnostic, no successful installation claim and unchanged pre-existing package/standalone state.

`TestInstallerQuietSourceBuildFailure` supplies a buildable fixture marker, a fake Go build that fails before replacing anything, and a pre-existing compatible fake rforge binary. Require quiet/verbose failure parity, preserved prior binary bytes and visible build failure, not false success from checking that old binary. A positive successful quiet/dry-run control retains current behavior. All commands operate only inside temporary HOME/PATH; no real host/network or new fixture package is needed. Preserve the existing script-level exit convention (nonzero aggregate failure), not a new requirement to expose the child's exact exit number as the top-level code.

Proposed focused check: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/installer -run 'TestInstallerQuiet|TestQuiet|TestHermes' -count=1`; `bash -n install.sh`. Tests do not exist yet and were not run.

### Ranking acceptance design

Begin with `TestRankRejectsBlankTopicBeforeSideEffects`: table cases empty, ASCII whitespace, tabs/newlines and Go-recognized Unicode spaces. Use a nonexistent graph path plus connector/search counters and a temporary output destination; each blank input must return the typed usage error/exit 2 **before** graph access, connector selection/search or output/provenance creation. Empty already works; whitespace cases provide the new red test. Use injected connectors, never live providers.

`TestRankBlankTopicCLI` covers text/JSON CLI errors via existing service plumbing, expected exit 2/code usage, no result file or provenance artifact, and a no-network fixture configuration even if validation regresses. An unsupported source plus nonexistent graph is a safe negative path; an injected loopback counter may additionally prove zero dispatch. A nonblank control with surrounding spaces verifies unchanged source query bytes and existing ranking/provenance behavior. Do not add MCP tools or normalize valid queries.

Proposed focused check: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/research ./internal/cli -run 'TestRankRejectsBlankTopicBeforeSideEffects|TestRankBlankTopicCLI|TestServiceRank|TestRankPersistsRankingsWithoutReplacingProvenance' -count=1`. New tests are proposed/unrun; the full merged-tree guard is `make check`.

## Pre-mortem and verification

1. The installer still passes through an old working binary after the new build fails: the preseeded-binary negative test requires failure despite a passing old capability probe.
2. A general process-wrapper refactor regresses Hermes or quiet success/dry-run: retain Hermes lifecycle tests and success controls; exclude that refactor.
3. Blank-input repair strips meaningful nonblank queries or creates artifacts before rejecting: assert unchanged query bytes and no graph/connector/output/provenance side effects for rejected inputs.

**Verify / direction:** observable error and side-effect properties, not a fabricated scalar score or research-quality delta. Desired outcomes are no false-success failure cases and no side effects for rejected topics, with existing valid behavior preserved. The new test selectors are named future checks, not measured passing baselines. Planning performed only source/contract inspection, task-store readiness and the read-only shell semantics check; no product tests were created or code repaired.

No strategic new feature is accepted here. Learned ranking, source routing, new benchmark corpus/campaigns, all-profile installation and further product directions stay outside runnable scope. The paused RF-BENCH source/task/goal records remain untouched. Save and independently review the two task records, validate the store, commit this planning batch once and stop.

## Saved handoff

- [V0-020-b78e6981](../../tasks/items/V0-020-b78e6981.md): first by priority, quiet installer failure propagation; begin with `TestInstallerQuietHostFailure`.
- [V0-021-b479f92a](../../tasks/items/V0-021-b479f92a.md): independent second lane, whitespace-only rank rejection; begin with `TestRankRejectsBlankTopicBeforeSideEffects`.
- [RF-RELIABILITY](../../tasks/goals/RF-RELIABILITY.md) is an unmet derived engineering goal scoped to these error-boundary fixes. Existing goals, focus and paused benchmark task records are not rewritten.
- Fresh independent read-only builder review: **nothing important left**; no acceptance contradiction, missing implementation input or unjustified authority was identified. Disjoint write sets permit parallel execution, subject to actual runner ownership/capacity. The reviewer did not implement or run tests.
- Store preparation reports READY; worktree task validation reports no drafts. New implementation tests remain proposed/unrun. A planning commit is not a claim that either issue is fixed or independently qualified.
