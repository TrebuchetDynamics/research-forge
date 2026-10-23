# Blockers

## Active

- **V0-012-88342f33 — owner-deferred pending provider access.** The 2026-10-22 owner decision requires either a custodian-held Brave web-search key (`BRAVE_API_KEY`) or a usable SearXNG endpoint (`SEARCH_HUB_SEARXNG_URL` or `SEARXNG_URL`, with authentication if required). The resumed worker reports both options absent and made no changes or new live probes; it cites the earlier DuckDuckGo human-verification block. Supplying access would still leave gateway implementation, recorded-response replay and live qualification work. No credentials, provider choice or expanded spend are granted by this review. See the [current evidence review](docs/tasks/v0-012-baseline-search-blocker.md). The count correction is already present. The task is placed in Needs decision to honor the existing owner deferral rather than advertise it as ready work; scope, Acceptance, provider inputs and the retained held attempt are unchanged. This is queue reconciliation, not provider recovery or successful live qualification.

- **V-B6-run — native-search qualification unavailable.** The retained worker report says Codex 0.163.0 cannot expose the provider-attempt controls required by the accepted comparison. Resume only with qualifying host-interface evidence or an explicit owner protocol decision; no replacement baseline is authorized. See the [evidence review](docs/tasks/v-b6-run-native-search-blocker.md). The task contract and hold remain unchanged.

### Q-V0-018-8f442e1d — Planning stopped

- Goal: RF-BENCH — still unmet after offline planning round 2 of 10.
- Task: [V0-019-ea266b81](tasks/items/V0-019-ea266b81.md) — owner continuation question, not executable building work.
- Evidence: [Round-2 review](docs/tasks/rf-bench-offline-round-2.md). V0-016 comparison and V0-017 public-pilot readback are delivered. Baseline search still waits on V0-012's existing provider input; convincing expanded agent-tier proof remains unqualified and live work remains owner-deferred. The current citation pipeline lacks pinned evaluation corpus/occurrence inputs; the pilot cannot be retrofitted with invented measurements.
- Question: For unmet RF-BENCH, should planning remain paused under the existing provider/live deferrals, or which further offline benchmark outcome should be scoped next?
- Default if no answer: Keep the goal unmet and the continuation task in Needs decision; save no successor plan. Preserve V0-012 and all owner live deferrals; no new execution, credentials, data approval or spend is authorized.

<!-- pi-autowork:questions:start -->

<!-- pi-autowork:questions:end -->
