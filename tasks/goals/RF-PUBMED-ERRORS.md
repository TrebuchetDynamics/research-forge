+++
schema = 1
id = "RF-PUBMED-ERRORS"
title = "Keep configured PubMed API keys out of rendered request failures"
priority = 50
source = "docs/outbound-api-data-flow.md#provenance-is-not-package-redaction"
status = "unmet"
focus = false
depends_on = []
+++
- Admission authority: round-5 repository-work discovery request permits bounded ordinary offline maintenance and a new goal only when needed. A reproduced PubMed failure exposes the configured API key in newly saved search artifacts; this goal covers only that error-rendering boundary, not universal artifact sanitization or a new provider capability.
- Existing boundaries: RF-RELIABILITY remains limited to its delivered installer/ranking frontier; RF-EGRESS-DOCS owns the completed guide repair, not production code. Neither existing goal is reopened or changed. RF-BENCH's planning pause and all provider/live/release/host and human gates remain intact.
- Outcome pending: PubMed request-failure text excludes the configured key while retaining failure meaning and diagnostic classification; real requests still receive the configured authentication. Offline regression evidence is not live/provider, scientific, host or release qualification.
