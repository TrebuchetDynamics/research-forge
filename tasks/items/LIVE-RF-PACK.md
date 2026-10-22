+++
schema = 1
id = "LIVE-RF-PACK"
title = "Live qualifications for RF-PACK"
short = "Live qualifications"
status = "open"
goal = "RF-PACK"
section = "Needs decision"
rank = 1001
depends_on = ["V0-013-2a4d9252"]
touches = ["install.sh", "internal/installer/install_test.go", "internal/installer/hermes_safety_test.go", "docs/install.md"]
resources = []
kind = "implementation"
ownership = "owner"
scope = "Owner-authorized live checks accumulated from offline task closures in this goal."
acceptance = ["V0-013-2a4d9252: Real default-profile installation/no-op and runtime skill discovery remain the separately owner-gated V0-014-6706388b qualification. No live Hermes, network installation, gateway restart or user-profile access was performed."]
checks = ["Live qualification by an authorized person"]
exclusions = ["Offline evidence does not count"]
+++
