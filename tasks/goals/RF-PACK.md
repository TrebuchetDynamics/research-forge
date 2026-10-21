+++
schema = 1
id = "RF-PACK"
title = "Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine"
priority = 60
source = "docs/verifiable-engine-plan.md#verifiable-research-engine-plan"
status = "unmet"
focus = false
depends_on = []
+++

- Evidence: executed: 79606f1: go test ./internal/cli ./internal/docs; rforge capabilities --json smoke; make check via pre-commit (pass)
- Evidence: executed: a30a567: go test ./internal/docs; node --test adapters/test_pi.mjs; make check via pre-commit (pass)
