+++
schema = 1
id = "V-B6-codex"
title = "Live agent-tier pilot on Codex"
short = "Codex agent-tier pilot"
status = "open"
goal = "RF-BENCH"
section = "Needs decision"
rank = 30000
depends_on = ["V-B6-run"]
touches = ["benchmarks/agent-tier", "docs/benchmark"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Run the agent-tier pilot on Codex as a separate experiment and report, under the same pre-registered protocol as the Pi pilot (V-B6-run). Blocked until the Codex host exposes native-search provider identity, retry and HTTP accounting that pass the gateway 429 probe, or the owner permits a custom, fully logged non-native search baseline with an updated pre-registered protocol."
acceptance = ["a committed Codex pilot report with paired differences and 95% CIs per metric, never pooled with Pi."]
checks = ["make check"]
exclusions = []
+++

- Sources: docs/benchmark/agent-tier.md#2-pre-register-before-live-work
- Owner decision: 2026-10-21: split from V-B6-run; waits for a Codex search interface that passes the protocol, or a later owner decision on a custom baseline
