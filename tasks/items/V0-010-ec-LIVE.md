+++
schema = 1
id = "V0-010-ec-LIVE"
title = "Live checks for V0-010-ec"
short = "Live checks for V0-010-ec"
status = "done"
goal = "RF-PACK"
section = "Needs decision"
rank = 19000
depends_on = ["V0-010-ec"]
touches = ["hermes-plugin/", "adapters/", "internal/docs/", "docs/agent-install.md", "docs/install.md", "README.md"]
resources = []
kind = "implementation"
ownership = "owner"
scope = "Run the live checks the offline work for V0-010-ec left pending."
acceptance = ["Owner-run Hermes subdirectory install/enable, skill discovery, and update/remove lifecycle qualification; no Hermes installation, profile or user-configuration changes were performed."]
checks = ["Live qualification by an authorized person"]
exclusions = ["Offline evidence does not count"]
closure = "manual"
closure_actor = "xel"
closure_timestamp = "2026-10-22T12:20:39.071675+00:00"
closure_reason = "2026-10-22 owner decision: waived this auto-generated live follow-up to clear the owner queue. The parent task's offline acceptance remains; live evidence was not produced and is not claimed. Reopen if an authorized live window becomes available."
+++
- Owner decision: 2026-10-21: deferred to the release gates (V-R1/V-R2); kept as their live-check checklist
