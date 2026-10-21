+++
schema = 1
id = "V0-010-ec-LIVE"
title = "Live checks for V0-010-ec"
short = "Live checks for V0-010-ec"
status = "open"
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
+++
- Owner decision: 2026-10-21: deferred to the release gates (V-R1/V-R2); kept as their live-check checklist
