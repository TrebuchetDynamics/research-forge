# V0-012 baseline-search access review

> Historical access review. V0-012-88342f33 was removed from the [current task queue](../../tasks/); its pinned contract and the disposition below are historical, not authority to resume. Further RF-BENCH planning remains [owner-paused](rf-bench-offline-round-2.md#owner-decision-planning-remains-paused).

Disposition: blocked on owner-supplied provider access. The background count correction is already present; no further task-contract change or restart is justified.

## Sources

- [Historical task](https://github.com/TrebuchetDynamics/research-forge/blob/3fcac85048bc2c5d744d85f99cf76929562fb4e1/tasks/items/V0-012-88342f33.md): recorded-response replay with parseable results and a non-empty bibliography, plus a three-question live smoke under the existing cap or the exact external input blocking it. Scope explicitly requires stopping when a key, allowlist or external account is the only fix.
- [Task source: Pi pilot provider loss](../benchmark/pilot-pi-2026-10-21.md#spend-provider-loss-and-budget-stops): 48 DuckDuckGo search attempts failed; all 24 without-tools observations returned empty bibliographies. These are 48 failures, not 48 baseline observations. The pilot is outage-conditioned, not a release-gate result.
- [Accepted plan](../verifiable-engine-plan.md#owner-answers-2026-10-19-follow-up), [specification §3](../benchmark/spec.md#3-leakage-controls-and-access-separation) and [§8](../benchmark/spec.md#8-agent-tier-paired-protocol-v-b6): native search, equivalent date/leakage gates, complete nested accounting and separately registered expanded execution remain required.
- [Custodian runbook §3](../benchmark/agent-tier.md#3-qualify-the-host-and-trusted-gateway-live-operator): actual gateway qualification includes first-429 quarantine without retry/wait; offline fixtures do not establish live qualification.
- [Release-gate record](../../tasks/items/V-R1.md): preserve the owner's manual closure/deferral and the later baseline-qualification requirement. No expanded spend or superiority claim is authorized by the pilot.
- Retained operational evidence in the common Git directory: `.git/autowork-execution/slot-data/V0-012-88342f33/attempt-1/report.json`, `contract.txt` and `.git/autowork-execution/slots/V0-012-88342f33.json`. The current resumed slot records baseline `8c5b34a98285505c1c455524f429984d9382eda5`, attempt 1, held with cause `blocker`. These references are not public qualification artifacts.
- [Historical owner decision, 2026-10-22](https://github.com/TrebuchetDynamics/research-forge/blob/3fcac85048bc2c5d744d85f99cf76929562fb4e1/tasks/items/V0-012-88342f33.md): execution is deferred until the owner supplies a custodian-held Brave web-search key or a usable SearXNG endpoint. Only after that input is supplied does the decision call for moving the task back to Now; V-R1 remains deferred.

## Observed blocker

The earlier investigation reported three synthetic-query probes returning HTTP 202 human-verification pages with zero parseable results, including an undated control returning the same challenge. The current resumed report cites that prior access failure and the owner's acknowledgement; it did not repeat the probes. This review has not independently repeated network probes or qualified a provider.

The live code corroborates the adapter restriction: `benchmarks/agent-tier/live/pi-session.mjs:52–59` passes `env: {}` to the installed Search Hub, while `benchmarks/agent-tier/live/gateway.py:298–310` permits only `https://html.duckduckgo.com/html/` for native search. `NativeResults` at `gateway.py:48–75` expects `result__a` anchors. Simply supplying a key would not implement or qualify a different gateway path.

The owner decision and resumed worker report name either a valid **Brave Search API key with web-search endpoint access (`BRAVE_API_KEY`)**, confined to the trusted custodian, or a **usable SearXNG endpoint (`SEARCH_HUB_SEARXNG_URL` or `SEARXNG_URL`, with authentication if required)**. The worker reports `BRAVE_API_KEY`, `SEARCH_BRAVE_API_KEY`, `SEARCH_HUB_SEARXNG_URL` and `SEARXNG_URL` absent. These are worker-reported environment checks, not an independent credential audit. Neither option is a qualified benchmark backend or a provider choice made by this review. No credential value belongs in repository files or receipts.

The resumed report records six existing offline gateway tests passing on unchanged baseline `8c5b34a98285505c1c455524f429984d9382eda5`. It explicitly says they do not establish recorded-provider replay or successful live qualification. No new test, implementation, paid request, live probe, expanded benchmark or commit was produced. This is a blocker report, not evidence for closing the unmet replay acceptance.

## Maintenance decision

The earlier Scope correction is already present: the pilot had 48 failed native search attempts across 24 baseline observations, not 48 baseline runs. Leave the task record unchanged, preserving all required work, both Acceptance items, Checks, Touches, Dependencies, status, queue position and runner hold. The owner's explicit deferral remains controlling; a cosmetic contract edit cannot supply access or justify restarting execution. Do not close the task merely because the blocker was named: the replay acceptance remains unmet. RF-BENCH remains unmet, and V-R1's manual closure is not benchmark qualification.

Do not provision credentials, choose or substitute a provider, bypass the human-verification challenge, weaken date/leakage/accounting controls, change the historical registration, grant spending or promise an unavailable input through a speculative producer task. No Blocker resolution marker is warranted because access has not been supplied or qualified.

The resumed worker report has `commit: null` and `changed_files: []`; this attempt supplies no integrated candidate or saved task-file patch to apply. Ledger validation reports no errors or pending proposals. No proposal is applied, no task record is edited, and no newer planning or closure is undone. This maintenance pass refreshes only the blocker documentation against the current report and owner decision.

Revisit only when authorized provider access is available to the custodian, or an explicit owner decision supplies another permitted route. Any continuation still needs the implemented native gateway path, recorded-response replay and actual three-question smoke receipts under existing authority; credentials alone are not qualification. Expanded execution still requires its own approved set, registration and spending authority.
