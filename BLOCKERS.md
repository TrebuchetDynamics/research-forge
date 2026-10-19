# Blockers

## Active

<!-- pi-autowork:questions:start -->

### Q-V-B4-lock — USER_DECISION — effort decision — unblocks 1

- Task: V-B4-lock — Owner approves and locks the gold sets
- Needs: 1 waiting task(s) depend on "V-B4-lock", which is not a Now/Next task (an outside condition, or a task in Needs decision).
- Since: 2026-10-19T13:08:59Z
- Answer: Decide it: record the decision in the dependent tasks' `Dependencies:` lines, or move the task that owns it into Now or Next.

### Q-LEDGER — USER_DECISION — effort investigate — unblocks 28

- Task: LEDGER — goals.json validation
- Needs: invalid goals.json: RF-CLAIMS: unmet goal has no open task
- Since: 2026-10-19T16:38:22Z
- Answer: DOCS-MAINTAINER could not repair the ledger. Read the validation errors above and its run record under the runner state directory (maintainer.json); add a task for remaining goal proof or correct evidence/status in TODO.md and goals.json, then run goals.py validate.

<!-- pi-autowork:questions:end -->
