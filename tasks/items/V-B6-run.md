+++
schema = 1
id = "V-B6-run"
title = "Live agent-tier pilot"
short = "Live agent-tier pilot"
status = "open"
goal = "RF-BENCH"
section = "Needs decision"
rank = 3000
depends_on = ["V-B4-lock", "V-B6"]
touches = ["benchmarks/agent-tier", "docs/benchmark"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Precondition (owner decision 2026-10-19): do this only after V-B11 makes the live comparison provider-independent. run the 48-run Pi pilot (8 questions × with/without rforge × 3 repetitions) with the V-B6 scripts, then Codex; record spend; commit the blinded report and effect-size estimates that size the expanded run."
acceptance = ["a committed pilot report with paired differences and 95% CIs per metric."]
checks = ["make check"]
exclusions = []
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner answers: agent tier).
- Ownership: ResearchForge owner (live spend authorization required before any run).
- Dependencies: V-B6, V-B4-lock.
