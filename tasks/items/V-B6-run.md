+++
schema = 1
id = "V-B6-run"
title = "Live agent-tier pilot"
short = "Live agent-tier pilot"
status = "open"
goal = "RF-BENCH"
section = "Now"
rank = 3000
depends_on = ["V-B4-lock", "V-B6"]
touches = ["benchmarks/agent-tier", "docs/benchmark"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Precondition (owner decision 2026-10-19): do this only after V-B11 makes the live comparison provider-independent. Run the 48-run Pi pilot (8 questions × with/without rforge × 3 repetitions) with the V-B6 scripts; record spend; commit the blinded report and effect-size estimates that size the expanded run. The Codex experiment is V-B6-codex, not part of this task (owner decision 2026-10-21)."
acceptance = ["a committed Pi pilot report with paired differences and 95% CIs per metric."]
checks = ["make check"]
exclusions = []
live = "authorized"
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner answers: agent tier).
- Ownership: ResearchForge owner (live spend authorization required before any run).
- Dependencies: V-B6, V-B4-lock.
- Owner decision: 2026-10-21: after the Codex native-search blocker, narrowed to the Pi pilot; Codex moves to V-B6-codex
