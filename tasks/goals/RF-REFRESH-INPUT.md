+++
schema = 1
id = "RF-REFRESH-INPUT"
title = "Reject unreadable or malformed refresh baselines before retrieval"
priority = 84
source = "internal/cli/source_commands.go#executeSearchRefresh"
status = "unmet"
focus = false
depends_on = []
+++

- Scope: One offline refresh-only input-boundary repair: propagate prior result read/decode errors before preview success or retrieval, preserving valid empty/no-DOI inputs and all existing delta/append safeguards. No change to shared readers, manifest/source/budget/transaction policy or runtime ownership.
- Admission authority: the user’s round-10 independent-maintenance request permits this bounded building task, implemented only by the normal worker/review pipeline. This is intended work, not completed implementation, scientific/live qualification or expansion of RF-BENCH or the append-only RF-REFRESH-IO goal.
