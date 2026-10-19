# Blockers

## Active

<!-- pi-autowork:questions:start -->

### Q-V-B4-lock — USER_DECISION — effort decision — unblocks 1

- Task: V-B4-lock — Owner approves and locks the gold sets
- Needs: 1 waiting task(s) depend on "V-B4-lock", which is not a Now/Next task (an outside condition, or a task in Needs decision).
- Since: 2026-10-19T13:08:59Z
- Answer: Decide it: record the decision in the dependent tasks' `Dependencies:` lines, or move the task that owns it into Now or Next.

### Q-V-E0 — USER_INPUT — effort input — unblocks 5

- Task: V-E0 — Capability report for compatibility checks
- Needs: changed files the coordinator owns: TODO.md, goals.json
- Since: 2026-10-19T13:53:12Z
- Answer: Answer in V-E0's TODO.md entry: widen `Touches:`, supply the input, narrow the acceptance, or add a dated `Owner decision:` line. The runner reruns it when the entry changes.

<!-- pi-autowork:questions:end -->
