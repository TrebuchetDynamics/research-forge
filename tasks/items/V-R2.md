+++
schema = 1
id = "V-R2"
title = "Cross-host release gate"
short = "Cross-host release gate"
status = "done"
goal = "RF-RELEASE"
section = "Needs decision"
rank = 32000
depends_on = ["V-E4", "V-E5-run"]
touches = ["docs/release.md"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "2.0 also requires V-E4 parity green on the release commit and V-E5-run receipts for all four hosts on the release version. Skill-only plugin releases before 2.0 need only the V-E1 conformance test."
acceptance = ["docs/release.md lists both checks with links to the evidence."]
checks = ["make check"]
exclusions = []
closure = "manual"
closure_actor = "xel"
closure_timestamp = "2026-10-22T12:44:41.868871+00:00"
closure_reason = "2026-10-22 owner decision: defer the live 2.0 release-gate/benchmark work and close these queue rows. No 2.0 release, benchmark superiority, host receipt or live evidence is claimed; offline work and recorded evidence remain, and any gate can be reopened when an authorized live campaign is funded."
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner requirements: release acceptance).
- Ownership: ResearchForge owner (release decision).
- Owner decision: 2026-10-21: release decision waits for V-E5-run's host receipts
