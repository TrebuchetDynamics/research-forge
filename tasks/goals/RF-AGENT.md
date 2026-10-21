+++
schema = 1
id = "RF-AGENT"
title = "Make the agent contract correct: install docs, search depth and provenance levels"
priority = 20
source = "docs/verifiable-engine-plan.md#verifiable-research-engine-plan"
status = "met"
focus = false
depends_on = []
+++

- Evidence: executed: 8b6408e: go test ./internal/provenance ./internal/reviewpkg ./internal/cli; make check via pre-commit with home TMPDIR (pass)
- Evidence: executed: Merge verification: go test ./internal/docs ./internal/provenance ./internal/cli ./internal/reviewpkg ./internal/report ./internal/webui; make check (pass)
