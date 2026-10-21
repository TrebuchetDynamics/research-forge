# V-FS2 budget-preflight review

Disposition: blocked; no task-contract or ledger change is justified.

## Sources

- [Current task](../../tasks/items/V-FS2.md): V-FS2, including both Acceptance items and the shared Budget field.
- [Accepted authorization](../benchmark/frontier-science-plan.md#authorized-v-fs2-campaign): one combined USD 20 / 200,000-token / 7,200-second allowance, including setup, cached input, helpers and resumes. Unknown consumption is not zero; unsafe metered operations must stop.
- [Engine boundary](../verifiable-engine-plan.md#owner-decisions-2026-10-19): host-agent semantic judgment remains separate from engine operations.
- Retained local worker evidence: `.git/autowork-execution/slot-data/V-FS2/attempt-1/report.json` and `.git/autowork-execution/slots/V-FS2.json` in the common Git directory. These are operational evidence references, not public research artifacts.

## Observed blocker

Attempt 1 initially stopped during model-budget preflight, reporting historical usage but no available task-scoped token/cost admission control. Its 2026-10-20T19:26:55.419480+00:00 snapshot reported 172,411 input/output tokens including cached input and USD 0.15496280. The reported 27,589 remaining tokens excluded continuation/final responses and future calls; it was neither a reservation nor a current balance.

The latest retained `attempt-1/report.json`, on baseline `5e4f6d989e1ebbac15c6a32ba3c4d75fa347b13d`, reports that the resumed session `pi-autowork-V-FS2-attempt-1` had consumed **403,202 combined input/output tokens including cacheRead/cacheWrite** across nine assistant messages at **2026-10-20T23:21:08.139646+00:00**, versus the shared **200,000-token ceiling**. Host-reported model cost was **USD 0.31493680**, with **zero missing usage records**. These are the worker's persisted-ledger observations, not independently recomputed totals or a final balance: subsequent closeout messages are excluded. The token ceiling is exhausted even though the reported dollar cost is below USD 20. Earlier setup and resumed messages share the same allowance.

The earlier worker reported passing four offline packet tests, packet validation and a whitespace check. The latest report records no checks, no commit and no changed files; it explicitly says those historical passes were not rerun on the current baseline. Neither topic ran; no campaign, scientific report or source audit was produced. Campaign validation, both real provenance validations and substantive source audits remain unrun. Historical offline passes do not meet V-FS2 Acceptance or unblock V-FS3.

## Maintenance decision

The accepted contract already requires stopping at the first reached limit and on an unsafe unbounded metered operation. The latest report establishes an exhausted token allowance, not merely the earlier admission-control gap. Rephrasing the contract, splitting the topics or adding another preflight cannot restore consumed tokens or authorize further research. No evidence supports closing the task, enlarging its write set, removing a dependency, increasing its allowance or treating cached input as free. Attempt 1 has no commit, changed files or saved ledger proposal to apply.

Leave V-FS2's contract, acceptance, budget, status and queue position unchanged. Leave V-FS3 dependent on V-FS2; do not promote synthetic or failed-before-search fixtures. No Blocker resolution marker is warranted.

Further research requires an explicit owner spending/scope decision accounting for all consumed usage, including the retained session's later closeout messages. A newly available runtime capability alone cannot replenish the exhausted allowance. Any authorized continuation must also safely bound and account for further metered work; budget approval alone does not create that capability. This documentation pass supplies neither authority nor runtime controls, authorizes no installation/settings change, and creates no speculative producer task in this engine repository.
