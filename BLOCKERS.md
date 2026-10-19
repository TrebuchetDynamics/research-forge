# Blockers

## Active

<!-- pi-autowork:questions:start -->

### Q-V-B4-lock — USER_DECISION — effort decision — unblocks 1

- Task: V-B4-lock — Owner approves and locks the gold sets
- Needs: 1 waiting task(s) depend on "V-B4-lock", which is not a Now/Next task (an outside condition, or a task in Needs decision).
- Since: 2026-10-19T13:08:59Z
- Answer: Decide it: record the decision in the dependent tasks' `Dependencies:` lines, or move the task that owns it into Now or Next.

### Q-V-B4 — USER_DECISION — effort investigate — unblocks 0

- Task: V-B4 — Draft candidate gold sets
- Needs: report.json does not match worker commit/files or reports failure: a check did not pass
- Since: 2026-10-19T14:50:35Z
- Answer: Read the evidence (the worker's report and validation log under .git/autowork-execution/slot-data/V-B4/), then rescope or split V-B4 in its TODO.md entry. The runner reruns it when the entry changes.

<!-- pi-autowork:questions:end -->
