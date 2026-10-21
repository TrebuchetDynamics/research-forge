# Blockers

## Active

- **V-B6-run — native-search qualification unavailable.** The retained worker report says Codex 0.163.0 cannot expose the provider-attempt controls required by the accepted comparison. Resume only with qualifying host-interface evidence or an explicit owner protocol decision; no replacement baseline is authorized. See the [evidence review](docs/tasks/v-b6-run-native-search-blocker.md). The task contract and hold remain unchanged.

<!-- pi-autowork:questions:start -->

### Q-V-R1 — USER_DECISION — effort decision — unblocks 1

- Task: V-R1 — ResearchForge 2.0 research-quality release gate
- Needs: 1 waiting task(s) depend on "V-R1", which is not a Now/Next task (an outside condition, or a task in Needs decision).
- Since: 2026-10-21T19:22:06Z
- Answer: Decide it: record the decision in the dependent tasks' `depends_on`, or move the task that owns it into Now or Next. To defer it, keep that task in Needs decision with a dated `Owner decision:` note; its dependents then wait without another question.
- Kind: outside

### Q-V-B6-run — USER_INPUT — effort input — unblocks 0

- Task: V-B6-run — Live agent-tier pilot
- Needs: worker reported a blocker: The installed Codex 0.163.0 native-search interface cannot satisfy the unchanged comparison protocol. Its generated app-server WebSearchThreadItem exposes id/query/action and optional opaque results, not interception/accounting of underlying provider HTTP attempts. A repository-local recorder cannot establish complete nested HTTP/provider receipts, enforce the reference-review/date gate before del…
- Since: 2026-10-21T19:22:06Z
- Answer: Answer in tasks/items/V-B6-run.md: widen `touches`, supply the input, narrow the acceptance, or add a dated `Owner decision:` note. The runner reruns it when the record changes.
- Kind: held

<!-- pi-autowork:questions:end -->
