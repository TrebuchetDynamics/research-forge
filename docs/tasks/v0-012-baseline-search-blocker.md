# V0-012 baseline-search access review

Disposition: blocked on external provider access. Correct only the task's background count of baseline observations; no change to required work or Acceptance is justified.

## Sources

- [Current task](../../tasks/items/V0-012-88342f33.md): recorded-response replay with parseable results and a non-empty bibliography, plus a three-question live smoke under the existing cap or the exact external input blocking it. Scope explicitly requires stopping when a key, allowlist or external account is the only fix.
- [Task source: Pi pilot provider loss](../benchmark/pilot-pi-2026-10-21.md#spend-provider-loss-and-budget-stops): 48 DuckDuckGo search attempts failed; all 24 without-tools observations returned empty bibliographies. These are 48 failures, not 48 baseline observations. The pilot is outage-conditioned, not a release-gate result.
- [Accepted plan](../verifiable-engine-plan.md#owner-answers-2026-10-19-follow-up), [specification §3](../benchmark/spec.md#3-leakage-controls-and-access-separation) and [§8](../benchmark/spec.md#8-agent-tier-paired-protocol-v-b6): native search, equivalent date/leakage gates, complete nested accounting and separately registered expanded execution remain required.
- [Custodian runbook §3](../benchmark/agent-tier.md#3-qualify-the-host-and-trusted-gateway-live-operator): actual gateway qualification includes first-429 quarantine without retry/wait; offline fixtures do not establish live qualification.
- [Release-gate record](../../tasks/items/V-R1.md): preserve the owner's manual closure/deferral and the later baseline-qualification requirement. No expanded spend or superiority claim is authorized by the pilot.
- Retained operational evidence in the common Git directory: `.git/autowork-execution/slot-data/V0-012-88342f33/attempt-1/report.json`, `contract.txt` and `.git/autowork-execution/slots/V0-012-88342f33.json`. The slot records baseline `3f14033b3c9f58e44ea392fb6cda1ccbfde2e1cd`, attempt 1, held with cause `blocker`. These references are not public qualification artifacts.

## Observed blocker

The worker reports three synthetic-query probes returning HTTP 202 human-verification pages with zero parseable results. A control without the date parameter returned the same challenge. This supports an access failure, not a changed-results-markup diagnosis. The report retains diagnostic locations and response digests; this review has not independently repeated the network probes or qualified a provider.

The live code corroborates the adapter restriction: `benchmarks/agent-tier/live/pi-session.mjs:52–59` passes `env: {}` to the installed Search Hub, while `benchmarks/agent-tier/live/gateway.py:298–310` permits only `https://html.duckduckgo.com/html/` for native search. `NativeResults` at `gateway.py:48–75` expects `result__a` anchors. Simply supplying a key would not implement or qualify a different gateway path.

The exact input named by the worker is a valid **Brave Search API subscription key with web-search endpoint access (`BRAVE_API_KEY`)**, available only to the trusted custodian, not the model sandbox. The worker reports neither a Brave key nor a SearXNG endpoint configured. Brave support in the original installed Search Hub is a worker-reported capability, not a qualified benchmark backend or a provider choice made by this review. No credential value belongs in repository files or receipts.

The report records six existing offline gateway tests passing on the unchanged baseline. It explicitly says they do not meet recorded-success replay or live-smoke acceptance. No new test, implementation, paid smoke, expanded benchmark or commit was produced; incremental model spend was USD 0.

## Maintenance decision

Correct V0-012-88342f33's Scope background through the task CLI: the pilot had 48 failed native search attempts across 24 baseline observations, not 48 baseline runs. Preserve all required work, both Acceptance items, Checks, Touches, Dependencies, status, queue position and runner hold. This factual correction does not resolve access or justify restarting the task. The contract already names the external-access stop condition; restating it cannot supply the missing input. Do not close the task merely because the blocker was named: the replay acceptance remains unmet. RF-BENCH remains unmet, and V-R1's manual closure is not benchmark qualification.

Do not provision credentials, choose or substitute a provider, bypass the human-verification challenge, weaken date/leakage/accounting controls, change the historical registration, grant spending or promise an unavailable input through a speculative producer task. No Blocker resolution marker is warranted because access has not been supplied or qualified.

The worker report has `commit: null` and `changed_files: []`; this attempt supplies no integrated candidate or saved task-file patch to apply. Initial ledger validation reported no errors or pending proposals; the only subsequent task edit is the factual count correction above. No newer planning or closure is undone.

Revisit only when authorized provider access is available to the custodian, or an explicit owner decision supplies another permitted route. Any continuation still needs the implemented native gateway path, recorded-response replay and actual three-question smoke receipts under existing authority; credentials alone are not qualification. Expanded execution still requires its own approved set, registration and spending authority.
