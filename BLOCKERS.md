# Blockers

## Active

- **V0-012-88342f33 — baseline search blocked on provider access.** The worker's three synthetic DuckDuckGo probes and undated control returned HTTP 202 human-verification pages without parseable results. It names a Brave Search API key with web-search endpoint access, confined to the trusted custodian, as the missing input; supplying it would still leave gateway implementation and qualification work. No credentials, provider substitution or expanded spend are authorized by this review. See the [evidence review](docs/tasks/v0-012-baseline-search-blocker.md). Only the task's background count is corrected (24 baseline observations, 48 search attempts); required work, Acceptance and hold remain unchanged.

- **V-B6-run — native-search qualification unavailable.** The retained worker report says Codex 0.163.0 cannot expose the provider-attempt controls required by the accepted comparison. Resume only with qualifying host-interface evidence or an explicit owner protocol decision; no replacement baseline is authorized. See the [evidence review](docs/tasks/v-b6-run-native-search-blocker.md). The task contract and hold remain unchanged.

<!-- pi-autowork:questions:start -->

<!-- pi-autowork:questions:end -->
