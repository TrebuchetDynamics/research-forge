# V0-031 — documentation-link repair blocked by protocol identity

Status: resolved; historical evidence review. The original review below describes baseline `a81dd564f776ead0df0dde348d6d617bd5c7adfb` before the later execution ruling. It is retained to explain the original hold, not to deny the current task's explicitly amended scope.

## Resolution after execution authorization

The user subsequently requested completion of all six documentation tasks. The [same task](../../tasks/items/V0-031-026e9f06.md) was amended to permit only the mechanically required `ProtocolSHA256` constant refresh in `internal/benchmarks/engine.go` alongside the documented link corrections. Its typed scope, touches, exclusions, acceptance and checks explicitly record this exception; the old general no-code restriction is superseded only for that constant. No test was weakened and no historical scientific/benchmark artifact, dataset, scoring algorithm or protocol semantics was rewritten.

The existing coordinator recovered the retained work and [verified completion](../../tasks/evidence/V0-031-026e9f06.toml), candidate `7c6cc9f11776467ac002ab763537394838bff437` integrated at `47242a3cf21b69723dcb7dfc957737a40342902a`. Current spec bytes and the engine constant both identify `1f5a1cd4a7ce62b1f063d66c9df1de9eaa3e61df4d2e7f0e0d683ceeccea9db6`. Exact-byte integrity and incompatible-protocol comparison checks remain intact; a newer hash does not retroactively requalify older reports or make their identities interchangeable. This is task completion, not new live/provider/benchmark authority. See [resolved blocker](../../BLOCKERS.md#v0-031--spec-link-repair-and-fingerprint-resolved).

## Historical review (not current instructions)

All present-tense status, digest and hold statements below apply to that original review baseline.

## Contract and retained evidence

The [accepted T6 plan](documentation-roi-plan.md#t6--repair-retired-contract-links-without-reviving-work) requires replacing the deleted TODO.md reference in `docs/benchmark/spec.md` and four removed V0-012 links across three historical handoffs. It preserves benchmark semantics, frozen inputs and the RF-BENCH pause. The plan's execution setup and the task's Worker-reversible note explicitly exclude production code; the task requires docs-only content edits and an unchanged `make check` gate.

Read in place under the repository's Git common directory:

- `autowork-execution/slot-data/V0-031-026e9f06/attempt-1/report.json`: candidate `d253c97327b38b8f0c881e7d2d663784e9be1e3f` repairs the four documentation files and reports passing scoped docs/link checks. The report is not a passing full-check receipt.
- `autowork-execution/slots/V0-031-026e9f06.json`: the first candidate failed integration verification at `ba6e1aacbfe101273cb9f2ab7bd9ee837f611cd7`; `TestEngineRecordPublication` failed in capture, redirect_and_retry, request_budget, URL-only_identity and Crossref_precise_online_date_and_cursor. Attempt 2 is held with cause `blocker`, no candidate commit, and its recorded worker PID no longer exists at inspection.
- `autowork-execution/slot-data/V0-031-026e9f06/attempt-2/report.json`: a temporary link-only spec edit reproduced the fingerprint mismatch; restoring the baseline restored the publication test. No changes were retained. The report requests explicit authority for the fingerprint update or another approved identity strategy.

The retained review finding under runner state, `review-findings/V0-031-026e9f06.json`, names the same integration revision and publication failures. Neither attempt's slot record has a dropped-ledger proposal (`ledger_edit_dropped` is null). The first candidate is not an ancestor of review baseline `a81dd564f776ead0df0dde348d6d617bd5c7adfb`; this is not an integrated task whose closure should be replayed.

## Source verification

- [`engine.go:19–21`](../../internal/benchmarks/engine.go) defines `ProtocolSHA256` as the exact-byte digest of the spec and says contract changes require regenerating it. Line 87 emits that value in engine reports.
- [`record_test.go:134–139`](../../internal/benchmarks/record_test.go) hashes the current spec and requires the emitted identity to match. Link-only edits are not exempt.
- [`comparison.go:53–56, 69–85`](../../internal/benchmarks/comparison.go) includes the protocol hash in report identity and rejects incompatible identities. Changing the constant therefore affects executable report identity, not only navigation text.
- The current spec digest is `cb3e2eb801d09108cd3f689d9ca855ec98e94e35849c531c48d88f756708dd11`, identical to the current constant. The retained candidate's spec digest is `1f5a1cd4a7ce62b1f063d66c9df1de9eaa3e61df4d2e7f0e0d683ceeccea9db6`, which differs. These were recomputed read-only from working-tree bytes and `git show d253c97327b38b8f0c881e7d2d663784e9be1e3f:docs/benchmark/spec.md` in an isolated Git environment.

Thus there is no pre-existing fingerprint defect in the current code to repair under the maintainer's check-unblocking allowance. A fingerprint update is an implementation consequence of the requested spec edit, but it lies outside this task's explicit code exclusion. Widening Touches alone would conceal that contradiction rather than resolve it. The engine comment explains the implementation mechanism; it does not override the task's authorization boundary.

## Disposition

Leave V0-031 unchanged and held. Do not drop its spec Acceptance, weaken the publication test, restore TODO.md, rewrite frozen reports/inputs, fabricate an owner decision, or treat scoped docs checks as full acceptance. Do not split an already-attempted task merely to bypass this blocker. No new benchmark work, successor task, dependency or goal qualification is added.

The unresolved boundary is explicit authority to reconcile the exact-byte protocol fingerprint while making the required spec-link edit, or an approved alternative identity strategy. This review supplies neither. Existing owner pause and live/provider deferrals remain binding. The historical blocker is now recorded under [resolved blockers](../../BLOCKERS.md#v0-031--spec-link-repair-and-fingerprint-resolved); the resolution above supersedes this earlier hold.

## Verification

At baseline `a81dd564f776ead0df0dde348d6d617bd5c7adfb`:

- Read-only digest comparison above: current spec matches the engine constant; retained candidate does not.
- `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/benchmarks -run '^TestEngineRecordPublication$' -count=1`: passed. This verifies the unchanged baseline, not completion of T6.
- `tasks validate --worktree .` using the requested pi-autowork command: passed before and after the documentation edits, 85 tasks and 10 goals, no drafts or pending proposals.
- `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check`: passed after the documentation edits, including all Go packages, 15 adapter tests, vet, completion audit, inventory and whitespace checks. This is baseline/review validation, not completion of T6.
- Fresh-context read-only reviewer: approved the evidence and unchanged-task disposition; found no justified contract-only repair overriding the code exclusion.

No worker report, runtime state or evidence bundle was modified or copied into this checkout.
