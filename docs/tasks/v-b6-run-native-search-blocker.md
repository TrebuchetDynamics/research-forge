# V-B6-run native-search qualification review

Disposition: blocked on a host-interface change or an explicit owner scope decision. No task-contract change is justified by the retained evidence.

## Sources

- [Current task](../../tasks/items/V-B6-run.md): 48 Pi observations, then Codex, actual spend and a committed pilot report with paired differences and 95% confidence intervals. The 2026-10-21 owner decision authorizes live spend; it does not replace the native-search baseline.
- [Accepted owner answers](../verifiable-engine-plan.md#owner-answers-2026-10-19-follow-up): Pi primary, Codex secondary, same model per pair, equal token/search budgets and blind scoring.
- [Benchmark specification §3](../benchmark/spec.md#3-leakage-controls-and-access-separation) and [§8](../benchmark/spec.md#8-agent-tier-paired-protocol-v-b6): both conditions require the reference-review/date gate and complete nested accounting; opaque accounting disqualifies a host. Native search remains native.
- [Agent-tier runbook §§3–4](../benchmark/agent-tier.md#3-qualify-the-host-and-trusted-gateway-live-operator): before accepting observations, qualify both gateways, intercept the first provider 429 before retries/waits and retain every HTTP-attempt receipt. A repository-local recorder cannot by itself prove an opaque external host enforced these controls.
- [Archived V-B6 and V-B11 contracts](../../TODO.archive.md): their acceptance covers offline stub tooling, not live host qualification. [V-B6-LIVE](../../tasks/items/V-B6-LIVE.md) and [V-B11-LIVE](../../tasks/items/V-B11-LIVE.md) retain qualification checklists; the owner deferred these to V-R1/V-R2 on 2026-10-21.
- Retained local evidence in the common Git directory: `.git/autowork-execution/slot-data/V-B6-run/attempt-1/report.json`, `contract.txt` and `.git/autowork-execution/slots/V-B6-run.json`. These are operational references, not committed pilot artifacts. The slot records baseline `fbec36b8e0669992e629f196f14d1501cd7235d2`, attempt 1, held with cause `blocker`.

## Observed blocker

The worker reports that installed Codex 0.163.0 exposes native-search `WebSearchThreadItem` fields `id`, `query`, `action` and optional opaque results, not controls over underlying provider HTTP attempts. Under that interface, it cannot establish complete nested receipts, enforce the reference-review/date gate before delivery, or intercept the first provider 429 before internal retries/waits. This review relies on the retained worker report for that installed-host observation; it does not claim to have qualified a live host independently.

The worker distinguishes the missing local live adapter/gateway, which remains implementation work, from the unavailable external interface. It does not claim missing authentication or spend authorization. Its report records passing focused offline tests (`go test ./benchmarks/agent-tier ./internal/benchmarks -count=1`) and a clean diff, but no live qualification probes, pilot observations, held-out scoring, effect estimates or confidence intervals. Reported pilot execution spend is USD 0, not an estimate of a completed campaign.

## Maintenance decision

Leave V-B6-run's Scope, Acceptance, Checks, Touches, Dependencies, live authorization, status and queue position unchanged. Do not substitute custom search for the native baseline, split away the required Codex experiment, invent provider receipts, grant permissions, or treat offline tests as live qualification. No Blocker resolution marker is warranted: the blocker has not been addressed. RF-BENCH remains unmet.

The report has `commit: null` and `changed_files: []`; this attempt supplies no integrated candidate task-file patch to apply. Ledger validation reports no errors or pending proposals. No newer planning or task closure is undone.

Resumption requires evidence of a host/provider interface that supplies the existing native-search controls, or an explicit owner decision permitting a changed comparison and a correspondingly revised, pre-registered protocol. This review chooses neither and grants no new spending, credentials or scope. A producer task promising an unavailable third-party interface would not resolve the blocker. Revisit this note when qualifying interface evidence or an explicit owner protocol decision is recorded; local adapter implementation and actual host qualification still precede accepted pilot observations.
