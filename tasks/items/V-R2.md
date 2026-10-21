+++
schema = 1
id = "V-R2"
title = "Cross-host release gate"
short = "Cross-host release gate"
status = "open"
goal = "RF-RELEASE"
section = "Needs decision"
rank = 6000
depends_on = ["V-E4", "V-E5-run"]
touches = ["docs/release.md"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "2.0 also requires V-E4 parity green on the release commit and V-E5-run receipts for all four hosts on the release version. Skill-only plugin releases before 2.0 need only the V-E1 conformance test."
acceptance = ["docs/release.md lists both checks with links to the evidence."]
checks = ["make check"]
exclusions = []
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner requirements: release acceptance).
- Ownership: ResearchForge owner (release decision).
