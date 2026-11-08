# V0-038 goal admission repair

## Bounded maintenance goal

RF-EGRESS-DOCS records only the already-admitted outcome in [V0-038-372c7752](../../tasks/items/V0-038-372c7752.md): correct `docs/outbound-api-data-flow.md` against the implemented metadata clients and GROBID, distinguishing outbound requests, local provenance and gated package redaction. It is an unmet engineering-maintenance goal, not a new product outcome, a successor documentation batch or an invitation to discover more work. Its remaining proof is exactly that task's existing Acceptance and Checks, including its before/after source audit, offline tests, changed-link review and unchanged `make check`.

The task's existing [trust-boundary source](../../README.md#trust-boundaries) and [documentation maintenance triggers](../README.md#maintenance-and-update-triggers) support this ordinary repair. No scope, acceptance, check, dependency, resource, ownership or authorization changes are needed. Keep Next/rank 1000; use priority 27 and focus=false, matching the original goal assignment without promoting the task above the owner's Now work.

## Evidence and disposition

At `9571f1bcfabb95d0d380fd7ae7ba1644f7cbb54f`, `tasks why V0-038-372c7752 .` returned `eligible: false` solely for `GOAL_MET: RF-DOCS`. The [RF-DOCS record](../../tasks/goals/RF-DOCS.md) correctly closes the six-task batch and its completion-audit follow-up; the [accepted plan's closeout](documentation-roi-plan.md#execution-closeout--subsequent-do-all-tasks-instruction) does not qualify this later maintenance repair. Preserve that met record and all earlier closures.

The current outbound guide still has seven integration rows, omits Semantic Scholar/PubMed/Europe PMC and calls OpenAlex credentials optional contact/user-agent. In contrast, `internal/sources/adapters/openalex.go::NewOpenAlexConnector` sets optional `RFORGE_OPENALEX_API_KEY` as an Authorization Bearer header. The existing task already records the fuller source audit and required correction; closure as no-change is not justified.

Read-only runner status showed zero running workers and zero awaiting integration. No V0-038 execution slot, slot-data directory, worker report or candidate patch exists. There is no retained proposal to replay. Reassign only this task to RF-EGRESS-DOCS and correct its contradictory Sources note; do not reopen RF-DOCS, supersede the task or edit runtime state.

## Admission verification

Red: the committed task's sole selection blocker is the met goal. Green: task-store validation must pass and selection against the edited store must report no blocker. Compare the original and edited task headers to confirm that only `goal` changes, and confirm RF-DOCS remains byte-identical. Run `git diff --check` and the normal offline `make check` gate. These validate the admission repair, not implementation of the guide or fresh provider/scientific/release qualification. The worker still owes every original Acceptance item.
