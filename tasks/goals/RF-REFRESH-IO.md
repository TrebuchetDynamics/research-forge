+++
schema = 1
id = "RF-REFRESH-IO"
title = "Preserve prior search artifacts when refresh appends fail"
priority = 83
source = "internal/cli/source_commands.go#executeSearchRefresh"
status = "unmet"
focus = false
depends_on = []
+++

- Scope: One bounded offline error-boundary repair in executeSearchRefresh: all four incremental JSONL append errors stop processing before staged commit, expose the original persistence cause and preserve prior artifacts/checkpoint. Use deterministic CLI-flow regression tests and existing transaction behavior. No batch/resume, transaction-algorithm, prior-DOI parsing or source-policy redesign; no live operation, qualification, successor planning batch or expansion of existing bounded goals.
- Admission authority: the user's round-9 independent-maintenance discovery request authorizes this new building task only; the normal worker/review pipeline implements and verifies it. This unmet goal records the bounded intended outcome, not completion or scientific/host/release readiness.
