+++
schema = 1
id = "RF-RELIABILITY-CLOSEOUT"
title = "Reconcile delivered reliability frontier with its unmet goal"
short = "Reconcile delivered reliability evidence"
status = "open"
goal = "RF-RELIABILITY"
section = "Now"
rank = 1000
depends_on = []
touches = ["docs/tasks/engineering-reliability-frontier.md"]
resources = []
kind = "inspection"
ownership = "worker"
scope = "Perform one bounded local documentation closeout of the accepted two-fix engineering reliability frontier. Map each original goal invariant to current implementation, nearest regression tests and existing coordinator receipts. Write a dated closeout addendum preserving the historical discovery text. Read but do not edit RF-RELIABILITY: regular workers cannot change goal records. Record the evidence-backed goal disposition recommended to the coordinator after this inspection receipt, or exact missing acceptance evidence, without claiming that the current unmet goal has changed. Stop after this document-only inspection; no new implementation task or successor planning batch."
acceptance = ["A traceability table covers quiet native failure, quiet source-build failure, quiet/verbose parity, prior installation preservation, positive quiet/dry-run behavior, Unicode-blank rank usage before side effects, CLI text/JSON error parity and preservation of valid query bytes.", "Each conclusion cites current file/test symbols and the V0-020/V0-021 coordinator receipts; historical reported checks are distinguished from any newly executed checks.", "The addendum accurately distinguishes the current unmet goal from a recommended coordinator-only disposition; goal metadata and historical receipts remain unchanged, with no duplicated implementation task or fabricated verification receipt.", "RF-BENCH, RF-RELEASE, RF-PACK, RF-MCP, existing deferred tasks and owner gates remain unchanged; no successor loop or release/scientific qualification claim is created."]
checks = ["autowork tasks validate --worktree .", "git diff --check", "Manually verify every closeout code/test/receipt citation and confirm the diff is limited to docs/tasks/engineering-reliability-frontier.md."]
exclusions = ["No production or test-code changes and no recreation of V0-020 or V0-021.", "No benchmark planning, corpus/scorer/protocol changes, provider repair, model calls or live/network tests.", "No installation, release, publication, credentials, spending or reopening deferred owner decisions.", "No changes to other goals, historical task contracts or coordinator receipts; no automatic follow-up batch."]
+++
- Sources: docs/tasks/engineering-reliability-frontier.md:29-35,99-107; tasks/goals/RF-RELIABILITY.md:1-11; tasks/evidence/V0-020-b78e6981.toml:1-9; tasks/evidence/V0-021-b479f92a.toml:1-9; install.sh:261-298; internal/installer/quiet_failure_test.go:9-133; internal/research/rank.go:45-60; internal/research/rank_input_test.go:21-74; internal/cli/rank_input_test.go:13-52; docs/tasks/rf-bench-offline-round-2.md:44-48
