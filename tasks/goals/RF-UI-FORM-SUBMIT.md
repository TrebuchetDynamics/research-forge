+++
schema = 1
id = "RF-UI-FORM-SUBMIT"
title = "Make local cockpit forms submit to served routes and preserve inputs"
priority = 93
source = "docs/web-gui-plan.md#purpose-boundaries"
status = "unmet"
focus = false
depends_on = []
+++

- Admission authority: independent-maintenance discovery request round 19 permits bounded ordinary offline building tasks and a new goal only when needed. Existing UI goals cover source-plan input round-trips (RF-UI-FORMS), local paper-asset resolution (RF-UI-ASSETS), static graph bounds (RF-UI-GRAPH-BOUNDS) and project-form rejection feedback (RF-UI-PROJECT-FEEDBACK); none is reopened or broadened.
- Outcome: the local project create/open/switch forms submit through their already-registered POST handlers when HTMX/JavaScript is unavailable, and the local search form submits to a served route that preserves the query and source selections instead of the unregistered `/search/results` target. The CLI remains the search authority; the cockpit still executes no network or provider search.
- Boundary: two bounded form repairs with focused handler tests. No route, handler, active-folder, HTTP-status, feedback-asset, provider, search-backend or search-execution change.
- Next placement under nonfocused priority 93 preserves owner priorities. This goal remains unmet pending normal implementation and review. Admission and offline tests are not completed implementation or browser/accessibility/feature/provider/host/release qualification. Preserve RF-BENCH's 2026-10-23 planning pause, Brave/SearXNG baseline-input requirements, Codex native-search limits, 2026-10-22 live benchmark/release/host deferrals, Hermes default-profile limits and all scientific/acquisition/privacy/export human gates. No Live authorization is granted.
