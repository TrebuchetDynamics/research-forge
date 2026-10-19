# Blockers

## Active

<!-- pi-autowork:questions:start -->

### Q-V-B4-lock — USER_DECISION — effort decision — unblocks 1

- Task: V-B4-lock — Owner approves and locks the gold sets
- Needs: 1 waiting task(s) depend on "V-B4-lock", which is not a Now/Next task (an outside condition, or a task in Needs decision).
- Since: 2026-10-19T13:08:59Z
- Answer: Decide it: record the decision in the dependent tasks' `Dependencies:` lines, or move the task that owns it into Now or Next.

### Q-FULL-CHECK — USER_INPUT — effort input — unblocks 0

- Task: the repository's whole offline test suite
- Needs: no full-suite gate: a commit can land while breaking tests its own task never ran, and main is never checked
- Since: 2026-10-19T13:08:59Z
- Answer: Add one line to AGENTS.md, `Full check: <command>` (the command that runs the whole offline suite), or set PI_AUTOWORK_VALIDATE in the runner's settings.

<!-- pi-autowork:questions:end -->
