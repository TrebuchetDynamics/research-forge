# V-FS2 budget-preflight review

Disposition: blocked; no task-contract or ledger change is justified.

## Sources

- [Current task](../../tasks/items/V-FS2.md): V-FS2, including both Acceptance items, the shared Budget field and the 2026-10-21 owner decisions. The owner raised the shared token ceiling from 200,000 to 1,000,000, preserving consumed usage and the USD 20 / 7,200-second ceilings. The later authorization permits one additional Majorana resume but explicitly leaves those limits unchanged.
- [Original accepted authorization](../benchmark/frontier-science-plan.md#authorized-v-fs2-campaign): the original combined USD 20 / 200,000-token / 7,200-second allowance, including setup, cached input, helpers and resumes. The task's subsequent explicit owner decisions amend its token/resume limits, not its cumulative accounting or human gates. Unknown consumption is not zero; unsafe metered operations must stop.
- [Engine boundary](../verifiable-engine-plan.md#owner-decisions-2026-10-19): host-agent semantic judgment remains separate from engine operations.
- Retained local worker evidence: `.git/autowork-execution/slot-data/V-FS2/attempt-1/report.json` and `.git/autowork-execution/slots/V-FS2.json` in the common Git directory. These are operational evidence references, not public research artifacts.

## Observed blocker

The initial preflight and 403,202-token stop documented in the earlier review are historical; the owner subsequently raised the shared ceiling to 1,000,000 tokens. That increase did not reset consumed usage.

The latest retained `attempt-1/report.json`, associated by `slots/V-FS2.json` with baseline `a5ae465e8c1a2d82999c256da690395dbbc070e8`, reports persisted cumulative **parent-plus-child usage of at least 1,108,473 input/output tokens, including cached input and all earlier attempts/resumes**, as of **2026-10-21T15:29:57.806821+00:00**. Reported model cost is **USD 0.75813320**. These are the worker's persisted-ledger observations, not independently recomputed totals or a final balance: subsequent closeout usage is additional, not zero. The reported lower bound exceeds the current shared token ceiling by 108,473 tokens even though reported dollar cost is below USD 20.

The latest report records no checks, no commit and no changed files on this dispatch. It describes an existing partial campaign checkpoint, not an absence of all prior execution: the newly authorized Majorana resume was not exercised, and the required scientific reports, evidence matrices, source audits, provenance indexes and reproduction instructions remain absent. No current-baseline tests or validators were run after the stop, and no public example qualifies. Historical offline passes do not meet V-FS2 Acceptance or unblock V-FS3. The slot remains held with cause `blocker`; this dispatch supplies no candidate commit or saved ledger patch to apply.

## Maintenance decision

The accepted contract already requires stopping at the first reached limit and on an unsafe unbounded metered operation. The latest report establishes exhaustion of the raised token allowance, not merely the earlier admission-control gap. The additional resume authorization does not override that stop. Rephrasing the contract, splitting the topics or adding another preflight cannot restore consumed tokens or authorize further research. No evidence supports closing the task, enlarging its write set, removing a dependency, increasing its allowance or treating cached input as free.

Leave V-FS2's contract, acceptance, budget, status and queue position unchanged. Leave V-FS3 dependent on V-FS2; do not promote synthetic or failed-before-search fixtures. No Blocker resolution marker is warranted.

Further research requires an explicit owner spending/scope decision accounting for all consumed usage, including the retained session's later closeout messages. A newly available runtime capability alone cannot replenish the exhausted allowance. Any authorized continuation must also safely bound and account for further metered work; budget approval alone does not create that capability. This documentation pass supplies neither authority nor runtime controls, authorizes no installation/settings change, and creates no speculative producer task in this engine repository.
