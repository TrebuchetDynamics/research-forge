+++
schema = 1
id = "V-E5-run"
title = "Live host qualification receipts"
short = "Live host qualification receipts"
status = "open"
goal = "RF-PACK"
section = "Next"
rank = 1000
depends_on = ["V-E5", "V-R1"]
touches = ["docs/qualification"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "execute the V-E5 runbook on each host for the release version and store receipts as `docs/qualification/<version>-<host>.md`."
acceptance = ["four receipts for the same version, each with every runbook observation recorded."]
checks = ["make check"]
exclusions = []
+++

- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner requirements: release acceptance).
- Ownership: ResearchForge owner (live host environments).
- Owner decision: 2026-10-21: run on the release candidate after the research-quality gate V-R1 passes
