+++
schema = 1
id = "V-R1"
title = "ResearchForge 2.0 research-quality release gate"
short = "ResearchForge 2.0 research-quality release gate"
status = "done"
goal = "RF-RELEASE"
section = "Needs decision"
rank = 31000
depends_on = ["V-B6-run"]
touches = ["docs/benchmark", "docs/release.md"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "run the expanded agent-tier benchmark. 2.0 ships only if the same agent with rforge demonstrably beats itself without rforge on evidence quality (citation correctness), retrieval effectiveness (recall) or research efficiency (tokens/requests/time at equal-or-better quality). Planner's operationalization: the paired 95% CI excludes zero in rforge's favor on at least one of those primary metrics, and no other primary metric is worse beyond its CI. Checked on Pi; the Codex result is reported alongside. Work that moves a gate metric takes priority over work that does not."
acceptance = ["a committed gate report referenced from docs/release.md."]
checks = ["make check"]
exclusions = []
closure = "manual"
closure_actor = "xel"
closure_timestamp = "2026-10-22T12:44:41.173554+00:00"
closure_reason = "2026-10-22 owner decision: defer the live 2.0 release-gate/benchmark work and close these queue rows. No 2.0 release, benchmark superiority, host receipt or live evidence is claimed; offline work and recorded evidence remain, and any gate can be reopened when an authorized live campaign is funded."
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Release gate).
- Ownership: ResearchForge owner (release decision and live spend).
- Owner decision: 2026-10-22: the Pi pilot (docs/benchmark/pilot-pi-2026-10-21.md) does not meet the gate: the without-rforge baseline returned no parseable search results in all 48 runs, so the positive recall contrast is descriptive, not a demonstrated improvement, and citation correctness stayed unmeasured. 2.0 does not ship on this evidence. The gate stays deferred until the baseline search is qualified (new task under RF-BENCH) and an expanded run is registered with its own spending cap; no expanded spend is authorized yet.
