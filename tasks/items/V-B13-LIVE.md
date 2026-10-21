+++
schema = 1
id = "V-B13-LIVE"
title = "Live checks for V-B13"
short = "Live checks for V-B13"
status = "open"
goal = "RF-BENCH"
section = "Needs decision"
rank = 25000
depends_on = ["V-B13"]
touches = ["**"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Run the live checks the offline work for V-B13 left pending."
acceptance = ["Each check passes on the real target: - Repeat the approved pool A capture against live providers; this task used only injected transports and offline checks."]
checks = ["Live qualification by an authorized person; offline evidence does not count."]
exclusions = []
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); generated live follow-up of V-B13, as indexed in goals.json.
- Touches: none before conversion; ** stands for the whole repository
- Owner decision: 2026-10-21: deferred to the release gates (V-R1/V-R2); kept as their live-check checklist
