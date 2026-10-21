+++
schema = 1
id = "V-R1"
title = "ResearchForge 2.0 research-quality release gate"
short = "ResearchForge 2.0 research-quality release gate"
status = "open"
goal = "RF-RELEASE"
section = "Needs decision"
rank = 5000
depends_on = ["V-B6-run"]
touches = ["docs/benchmark", "docs/release.md"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "run the expanded agent-tier benchmark. 2.0 ships only if the same agent with rforge demonstrably beats itself without rforge on evidence quality (citation correctness), retrieval effectiveness (recall) or research efficiency (tokens/requests/time at equal-or-better quality). Planner's operationalization: the paired 95% CI excludes zero in rforge's favor on at least one of those primary metrics, and no other primary metric is worse beyond its CI. Checked on Pi; the Codex result is reported alongside. Work that moves a gate metric takes priority over work that does not."
acceptance = ["a committed gate report referenced from docs/release.md."]
checks = ["make check"]
exclusions = []
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Release gate).
- Ownership: ResearchForge owner (release decision and live spend).
